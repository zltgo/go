package session

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	. "github.com/zltgo/fileserver/utils"
)

const (
	AccessToken    = "ATK"
	CreateTimeKey  = "CTK"
	UpdateTimeKey  = "UTK"
	StdLifeTime    = 30 * time.Minute
	MaxCreateTimes = 10000  //允许某IP地址在lifeTime内的最大创建次数（一般在登录时才会创建）
	MaxGetTimes    = 100000 //允许某IP地址在lifeTime内的最大获取次数（控制调用API的频率）
	c_Dimension    = 10000  //量纲，两个统计信息合成一个int64
)

type Session struct {
	// MaxAge=0 means no 'Max-Age' attribute specified.
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	// MaxAge>0 means Max-Age attribute present and given in seconds
	cookieMaxAge int
	countStore   *MemStore //保存调用Create的统计信息
	timeStore    *MemStore //保存创建时间与更新时间
	block        cipher.Block
}

func NewStd(maxage int) (s *Session) {
	return New(maxage, RandBytes(16), StdLifeTime)
}

//blockKey的长度必须为16，24或32
func New(maxage int, blockKey []byte, lifetime time.Duration) *Session {
	var s Session
	s.cookieMaxAge = maxage
	s.countStore = NewMemStore(lifetime)
	s.timeStore = NewMemStore(lifetime)

	var err error
	s.block, err = aes.NewCipher(blockKey)
	CheckErr(err)
	return &s
}

//获取一个session
var ErrOutmoded = errors.New("Session: update time is outmoded") //updatetime时间错误
var ErrIp = errors.New("Session: remote ip is invalid")          //验证IP会导致用户体验差，手机上网IP会经常变化
var ErrOverrun = errors.New("Session: too much call")

func (m *Session) Get(r *http.Request) (url.Values, error) {
	ck, err := r.Cookie(AccessToken)
	//此处只有一个错误，那就是http.ErrNoCookie
	if err != nil {
		return nil, err
	}
	//1, Decrypt
	b, err := DecryptUrl(m.block, ck.Value)
	if err != nil {
		return nil, err
	}
	//2, parse to url.vs
	vs, err := url.ParseQuery(string(b))
	if err != nil {
		return nil, err
	}

	//3, 该IP的Get操作计数加1
	ip, err := IPv4(r.RemoteAddr)
	if err != nil {
		return nil, ErrIp
	}
	f := func(v int64) bool {
		return (v / c_Dimension) < MaxGetTimes
	}
	if !m.countStore.AddFunc(ip, c_Dimension, f) {
		return nil, ErrOverrun
	}

	//4, check in local memstore
	ct := GetInt(vs, CreateTimeKey)
	pt := GetInt(vs, UpdateTimeKey)
	if ct == 0 || pt == 0 {
		return nil, ErrDecrypt
	}

	//store中保存了每个session的创建时间和更新时间
	tmpt := m.timeStore.Get(ct)
	if tmpt <= 0 {
		//没有找到说明半小时没有使用了，重新保存一下
		m.timeStore.Set(ct, pt)
		return vs, nil
	}
	if pt != tmpt {
		return nil, ErrOutmoded
	}

	return vs, nil
}

//创建一个Session
func (m *Session) Create(r *http.Request) (url.Values, error) {
	//该IP的Get操作计数加1
	ip, err := IPv4(r.RemoteAddr)
	if err != nil {
		return nil, ErrIp
	}
	f := func(v int64) bool {
		return (v % c_Dimension) < MaxCreateTimes
	}
	if !m.countStore.AddFunc(ip, 1, f) {
		return nil, ErrOverrun
	}

	vs := make(url.Values)
	t := strconv.FormatInt(time.Now().UnixNano(), 10)
	vs.Set(CreateTimeKey, t)
	vs.Set(UpdateTimeKey, t)

	return vs, nil
}

//更新一个Session
func (m *Session) Set(w http.ResponseWriter, vs url.Values) {
	//1, update time
	pt := time.Now().UnixNano()
	SetInt(vs, UpdateTimeKey, pt)

	//2, encrypt
	tk := EncryptUrl(m.block, []byte(vs.Encode()))

	//3, store
	ct := GetInt(vs, CreateTimeKey)
	Assert(ct != 0, "session的格式不正确，无创建时间")
	m.timeStore.Set(ct, pt)

	//4, set cookie
	cookie := &http.Cookie{
		Name:     AccessToken,
		Value:    tk,
		Path:     "/",
		HttpOnly: true,
		MaxAge:   m.cookieMaxAge}
	http.SetCookie(w, cookie)

	return
}

//原值不存在或者解析失败的话，则当原值为0
//返回增加后的结果
func AddInt(vs url.Values, key string, v int64) int64 {
	old, _ := strconv.ParseInt(vs.Get(key), 0, 64)
	v += old
	vs.Set(key, strconv.FormatInt(v, 10))
	return v
}

//原值不存在或者解析失败的话，则当原值为0
func GetInt(vs url.Values, key string) int64 {
	v, _ := strconv.ParseInt(vs.Get(key), 0, 64)
	return v
}

func SetInt(vs url.Values, key string, v int64) {
	vs.Set(key, strconv.FormatInt(v, 10))
	return
}
