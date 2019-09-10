package model

import (
	. "fisheep/conf"
	"fmt"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/afocus/captcha"
	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/session"
	"github.com/zltgo/node"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	MS          *mgo.Session
	LruMemCache                  = cache.DefaultLruMemCache
	Cap         *captcha.Captcha = captcha.New()
	MinSmsCode  int
	SmsCodeRand int

	Users    = "users"    // 用户集合名
	Areas    = "areas"    // 地区集合名
	Products = "products" // 产品分类集合名
	Smses    = "smses"    // 手机短信集合名
	Managers = "managers" // 管理员集合名
	Goods    = "goods"    // 商品集合名

	KeyCaptcha      = "captcha"
	SuperGrp        = "超级用户" // 拥有所有权限
	DefaultSuperUsr = "dawang"
	DefaultSuperPwd = "sunwukong"
)

func init() {
	//初始化随机种子
	unix := time.Now().UnixNano()
	rand.Seed(unix)
	MinSmsCode = int(math.Pow10(Pub.SmsCodeNum - 1))
	SmsCodeRand = MinSmsCode*9 - 1

	// 初始化图形验证码
	err := Cap.SetFont(filepath.Join(ConfDir, "comic.ttf"))
	if err != nil {
		log.Panicln("set font:", err)
	}

	Cap.SetSize(Pub.CaptchaWide, Pub.CaptchaHeight)
	Cap.SetDisturbance(captcha.DisturLevel(Pub.CaptchaDisturb))
	Cap.SetFrontColor(color.RGBA{255, 0, 0, 255}, color.RGBA{0, 0, 255, 255}, color.RGBA{0, 153, 0, 255})

	MS, err = mgo.Dial(Pub.MongoUrl)
	if err != nil {
		log.Panicln("dial mongo:"+Pub.MongoUrl, err)
	}

	MS.SetMode(mgo.Primary, true)
	MS.SetSafe(&mgo.Safe{WMode: "majority", WTimeout: Pub.WTimeout})

	// Users集合创建索引
	if err = MS.DB("").C(Users).EnsureIndex(mgo.Index{
		Key:        []string{"phone"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Panicln(err)
	}
	if err = MS.DB("").C(Users).EnsureIndex(mgo.Index{
		Key:        []string{"openid"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Panicln(err)
	}

	// Sms集合创建索引
	if err = MS.DB("").C(Smses).EnsureIndex(mgo.Index{
		Key:         []string{"smsat"},
		ExpireAfter: time.Hour * 24,
		Background:  true,
	}); err != nil {
		log.Panicln(err)
	}

	// 区域划分集合创建索引
	if err = node.EnsureIndex(MS.DB("").C(Areas)); err != nil {
		log.Panicln(err)
	}

	// 商品分类集合创建索引
	if err = node.EnsureIndex(MS.DB("").C(Products)); err != nil {
		log.Panicln(err)
	}

	// 商品集合创建索引
	if err = MS.DB("").C(Goods).EnsureIndex(mgo.Index{
		Key:        []string{"areaid, productid, order"},
		Background: true,
	}); err != nil {
		log.Panicln(err)
	}

	// 管理员集合创建索引
	if err = MS.DB("").C(Managers).EnsureIndex(mgo.Index{
		Key:        []string{"usr"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Panicln(err)
	}
	if err = MS.DB("").C(Managers).EnsureIndex(mgo.Index{
		Key:        []string{"openid"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Panicln(err)
	}

	// 没有超级管理员则增加一个默认的，拥有所有权限
	cnt, err := MS.DB("").C(Managers).Find(bson.M{"grp": SuperGrp}).Count()
	if err != nil {
		log.Panicln(err)
	}
	if cnt < 1 {
		hashed, err := bcrypt.GenerateFromPassword([]byte(DefaultSuperPwd), 0)
		if err != nil {
			log.Panicln(err)
		}
		if err = MS.DB("").C(Managers).Insert(ManagerDb{
			Id:      bson.NewObjectId(),
			OpenId:  bson.NewObjectId().Hex(),
			Usr:     DefaultSuperUsr,
			HashPwd: string(hashed),
			RegTime: time.Now(),
			Grp:     SuperGrp,
			AreaId:  node.RootId,
		}); err != nil {
			log.Panicln(err)
		}
	}
	return
}

// 主页重定向：GET: /
func Index(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	http.Redirect(w, r, "/static/index.html", http.StatusMovedPermanently)
	return nil, nil
}

// 获取图形验证码：GET:/captcha
// 错误码：429,500
// 输入：无
// 返回值：png（4-6个字符组成的验证码图片）
func GetCaptcha(w http.ResponseWriter, se session.Session) error {
	img, str := Cap.Create(Pub.CaptchaNum, captcha.ALL)
	// save captcha to session
	se.Set(KeyCaptcha, str)

	return png.Encode(w, img)
}

// 测试验证码：GET:/captcha
// 错误码：429,500
// 输入：无
// 返回值：string，用于测试
func TestCaptcha(w http.ResponseWriter, se session.Session) string {
	_, str := Cap.Create(Pub.CaptchaNum, captcha.ALL)
	// save captcha to session
	se.Set(KeyCaptcha, str)

	return str
}

// 测试短信验证码：GET:/test/sms
// 错误码：400,429,500
// 输入：UsrForm(只需提交手机号即可)
// 返回值：string，用于测试
// 注意事项：同一个手机号请求短信验证码有最小时间间隔限制和每日请求次数限制
func TestSms(db *mgo.Database, se session.Session, uf *UsrForm) (interface{}, error) {
	// 更新到数据库
	sms := newSmsCode()
	smsat := time.Now().Unix()

	if _, err := db.C(Smses).Upsert(bson.M{
		"_id":   uf.Phone,
		"smsat": bson.M{"$lte": smsat - int64(Pub.SmsMinAge)}, // 最小时间间隔
		"daily": bson.M{"$lte": Pub.SmsMaxCntDaily},           // 每日最大请求次数
	}, bson.M{
		"$set": bson.M{
			"sms":   sms,
			"smsat": smsat,
		},
		"$inc": bson.M{"daily": 1},
	}); err != nil {
		if mgo.IsDup(err) {
			return http.StatusTooManyRequests, fmt.Errorf(uf.Phone + "请求短信验证码过于频繁")
		} else {
			return http.StatusInternalServerError, err
		}
	}

	return sms, nil

	//	var ub usrDb
	//	if _, err := db.C(Smses).Find(bson.M{
	//		"phone": uf.Phone,
	//		"smsat": bson.M{"&lte": smsat - int64(Pub.SmsMinAge)},
	//	}).Apply(mgo.Change{
	//		ReturnNew: true,
	//		Update: bson.M{
	//			"$set": bson.M{
	//				"sms":   sms,
	//				"smsat": smsat,
	//			},
	//		}}, &ub); err != nil {
	//		if err == mgo.ErrNotFound {
	//			// 确认错误类型
	//			if n, _ := db.C(Users).Find(bson.M{"phone": uf.Phone}).Count(); n < 1 {
	//				return StatusNotRegistered, errors.New("手机号" + uf.Phone + "尚未注册")
	//			} else {
	//				return http.StatusTooManyRequests, fmt.Errorf("短信验证码的请求间隔不得小于%d秒", Pub.SmsMinAge)
	//			}
	//		}
	//		return http.StatusInternalServerError, err
	//	}
}

// 获取新的验证码
func newSmsCode() string {
	rd := MinSmsCode + rand.Intn(SmsCodeRand)
	return strconv.Itoa(rd)
}
