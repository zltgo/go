package model

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/zltgo/reflectx"
	"github.com/zltgo/webkit/cache"
	"github.com/zltgo/webkit/cache/lru"
	"github.com/zltgo/webkit/ginx"
	gm "github.com/zltgo/webkit/graphorm"
	"github.com/zltgo/webkit/ratelimit"
	"golang.org/x/exp/errors/fmt"
	"net/http"
	"reflect"
	"time"
)

var (
	AdminRole   = "admin"
	ManagerRole = "manager"
	GET         = "GET"
	POST        = "POST"
	PUT         = "PUT"
	DELETE      = "DELETE"
)

type Account struct {
	gm.Model         `json:",squash"`
	Corp             string    `json:"corp" gorm:"type:varchar(128);not null"`
	ManagerID        gm.UUID   `json:"managerID" gorm:"type:char(12);not null"`
	Mobile           string    `json:"mobile" gorm:"type:char(11);unique_index;not null"`
	MaxStores        int       `json:"maxStores" gorm:"not null"`
	MaxUsersPerStore int       `json:"maxUsersPerStore" gorm:"not null"`
	Remarks          string    `json:"remarks" gorm:"type:varchar(255)"`
	ExpiresAt        time.Time `json:"expiresAt" gorm:"not null"`

	//belongs to, the tag does not work in related, use association("manager") instead
	Manager *User `json:"manager" gorm:"ForeignKey:ManagerID"`

	//has many
	//Stores []Store `json:"stores" gorm:"ForeignKey:AccountID;save_associations:false"`
}

type Store struct {
	gm.Model  `json:",squash"`
	AccountID gm.UUID `json:"accountID" gorm:"type:char(12) REFERENCES accounts(id) on update no action on delete no action;unique_index:idx_acc_name;not null"`
	Name      string  `json:"name" gorm:"type:varchar(128);unique_index:idx_acc_name;not null"`
	Remarks   string  `json:"remarks" gorm:"type:varchar(255)"`
	//has many
	//Users []User `json:"users" gorm:"ForeignKey:StoreID;save_associations:false"`
}

type User struct {
	gm.Model     `json:",squash"`
	AccountID    gm.UUID  `json:"accountID" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
	StoreID      *gm.UUID `json:"storeID" gorm:"type:char(12) REFERENCES stores(id) on update no action on delete no action"`
	Empno        string   `json:"empno" gorm:"type:char(12);unique_index:idx_acc_emp;not null"`
	PasswordHash gm.MD5   `json:"-" gorm:"type:char(16);not null"`
	Salt         string   `json:"-" gorm:"type:char(16);not null"`
	RealName     string   `json:"realName" gorm:"type:varchar(50);not null"`
	Role         string   `json:"role" gorm:"type:varchar(50);not null"`
}

type Model struct {
	validator ginx.Validator
	db        *gorm.DB
	opts      Options
	cacheDB   cache.Cache
	limitIP   *lru.Cache  //cache for rate limit ip
	limitUser  *lru.Cache  //cache for rate limit user
	ipRates   ratelimit.RateMap
	userRates  ratelimit.RateMap
}

type Options struct {
	Admin struct {
		Name         string `default:"root"`
		PasswordHash string `default:"b7da45e2975c3f12971e3b3d39a72883"` //MD5(salt +"112358")
		Salt         string `default:"root"`
		MaxAge       int    `default:"86400"` // life time of admin token in seconds
	}
	Driver     string `default:"sqlite3"`
	Source     string `default:"./db/dress.sqlite3?charset=utf8&parseTime=True&loc=Local"`
	QueryLimit int    `default:"100"` //查询限制
	DBCacheEntries int  `default:"10000"` //数据库信息缓存个数
	IPCacheEntries int  `default:"10000"` //IP调用频率限制的缓存个数
	UserCacheEntries int  `default:"1000"` //用户调用频率限制的缓存个数
	IPRates     map[string][]int //每个ip的调用频率限制，单位为秒
	UserRates    map[string][]int //每个用户的调用频率限制，单位为秒
}

func NewModel(opts Options) (*Model, error) {
	reflectx.SetDefault(&opts)

	db, err := OpenDB(opts.Driver, opts.Source)
	if err != nil {
		return nil, err
	}

	//save rate limit of ip
	ipRates := make(map[interface{}][]ratelimit.Rate, len(opts.IPRates))
	for k, v := range opts.IPRates {
		ipRates[k] = ratelimit.SecOpts(v...)
	}

	//save rate limit of user
	userRates := make(map[interface{}][]ratelimit.Rate, len(opts.UserRates))
	for k, v := range opts.UserRates {
		userRates[k] = ratelimit.SecOpts(v...)
	}

	return &Model{
		validator: ginx.NewValidator(),
		db:        db,
		opts:      opts,
		cacheDB: cache.New(opts.DBCacheEntries),
		limitIP: lru.New(opts.IPCacheEntries),
		limitUser:lru.New(opts.UserCacheEntries),
		ipRates: ipRates,
		userRates:userRates,
	}, nil
}

// NewDB returns a new DB connection
func OpenDB(driver, source string) (*gorm.DB, error) {
	// connect to the db, create if it doesn't exist
	db, err := gorm.Open(driver, source)
	if err != nil {
		return nil, err
	}

	//sqlite3 需要手动启用外键约束
	if err = db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, err
	}
	if err = db.AutoMigrate(&Account{}, &User{}, &Store{}).Error; err != nil {
		return nil, err
	}

	//sqlite3不支持创建表以后再添加外键，只能使用tag的方式
	//if err = db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
	//	return nil, err
	//}
	//if err = db.Model(&Store{}).AddForeignKey("account_id",
	//	"accounts(id)",
	//	"no action",
	//	"no action").Error; err != nil {
	//	return nil, err
	//}
	//if err = db.Model(&User{}).AddForeignKey("store_id",
	//	"stores(id)",
	//	"no action",
	//	"no action").Error; err != nil {
	//	return nil, err
	//}

	return db, nil
}

// CloseDB close current db connection
func (m *Model) CloseDB() error {
	return m.db.Close()
}
func (m *Model) CacheStatus() lru.Stats{
	return m.cacheDB.Stats()
}

//从缓存或数据库获取数据
//pp must be a point of a point of a struct(**struct).
//The value of the struct is read only, do not write.
func (m *Model) GetFromCacheOrDB(id gm.UUID, pp interface{}) error {
	//这里有可能直接在缓存中找到该元素，ptr返回时有可能还是空值，只能使用函数返回值作为结果
	//_, err :=  m.cacheDB.Get(id, func(cache.Key) (interface{}, error) {
	//	err := m.db.Unscoped().Model(ptr).Where("id = ?", id).First(ptr).Error
	//	return ptr, err
	//})
	rv, err :=  m.cacheDB.Get(id, func(cache.Key) (interface{}, error) {
		typ := reflect.TypeOf(pp).Elem().Elem()
		val := reflect.New(typ)
		ptr := val.Interface()
		err := m.db.Unscoped().Model(ptr).Where("id = ?", id).First(ptr).Error
		return ptr, err
	})
	reflect.ValueOf(pp).Elem().Set(reflect.ValueOf(rv))
	return err
}

// 权限管理
func (m *Model) CheckPermission(af *AuthInfo, clientIp, OpName string) (int, error) {
	//rate limit of ip
	if clientIp != "" {
		key := OpName
		if _, ok := m.ipRates[OpName]; !ok {
			key = "Any"//means any operations
		}
		limiters := m.limitIP.Getsert(clientIp, func()interface{}{
			return ratelimit.NewLimiters(m.ipRates)
		})
		if limiters.(ratelimit.Limiters).Reached(key) {
			return http.StatusTooManyRequests, fmt.Errorf("rate limited: %s, IP: %s", OpName, clientIp)
		}
	}

	//rate limit of user
	if af != nil {
		limiters := m.limitUser.Getsert(af.UserId, func()interface{}{
			return ratelimit.NewLimiters(m.userRates)
		})
		if limiters.(ratelimit.Limiters).Reached(OpName) {
			return http.StatusTooManyRequests, fmt.Errorf("rate limited: %s, UserId: &s", OpName, af.UserId)
		}
	}

	//check permission
	return http.StatusOK, nil
}

//登录表单
type AuthForm struct {
	AccountID gm.UUID `json:"accountID" validate:"hexadecimal,len=24"`
	Empno     string  `json:"empno" validate:"number,min=3,max=12"`
	Password  string  `json:"password" validate:"alphanum,min=8,max=32"`
}

//用户身份信息
type AuthInfo struct {
	UserId       gm.UUID   `json:",omitempty"`
	AccountID    gm.UUID   `json:",omitempty"`
	Empno        string    `json:",omitempty"`
	PasswordHash string    `json:",omitempty"`
	Role         string    `json:",omitempty"`
	ExpiresAt    time.Time `json:",omitempty"`
}

// Account查询条件
type AccountFilter struct {
	CreatedAt        []*time.Time `json:"createdAt,omitempty" validate:"max=2"`
	UpdatedAt        []*time.Time `json:"updatedAt,omitempty" validate:"max=2"`
	DeletedAt        []*time.Time `json:"deletedAt,omitempty" validate:"max=2"`
	ExpiresAt        []*time.Time `json:"expiresAt,omitempty" validate:"max=2"`
	Unscoped         int          `json:"unscoped"` //0:in-use;1:all; 2:only deleted records
	Corp             string       `json:"corp" validate:"omitempty,max=128,name"`
	Mobile           string       `json:"mobile" validate:"omitempty,max=12,number"`
	Remarks          string       `json:"remarks" validate:"omitempty,max=255,name"`
	MaxStores        []*int       `json:"maxStores,omitempty" validate:"max=2"`
	MaxUsersPerStore []*int       `json:"maxUsersPerStore,omitempty" validate:"max=2"`
	Orders           []string     `json:"orders,omitempty" validate:"omitempty,max=3,dive,order,ne=id,ne=-id"`
}

type NewAccount struct {
	Empno            string    `json:"empno" validate:"number,min=3,max=12"`
	Password         string    `json:"password" validate:"alphanum,min=8,max=32"`
	RealName         string    `json:"realName" validate:"name,max=50"`
	Mobile           string    `json:"mobile" validate:"mobile"`
	Corp             string    `json:"corp" validate:"name,max=128"`
	MaxStores        int       `json:"maxStores" validate:"min=0,max=1000"`
	MaxUsersPerStore int       `json:"maxUsersPerStore" validate:"min=0,max=1000"`
	ExpiresAt        time.Time `json:"expiresAt"`
	Remarks          string    `json:"remarks" validate:"max=255"`
}

type NewStore struct {
	AccountID gm.UUID `json:"accountID" validate:"hexadecimal,len=24"`
	Name      string  `json:"name" validate:"name,max=128"`
	Remarks   string  `json:"remarks" validate:"max=255"`
}

type NewUser struct {
	AccountID gm.UUID `json:"accountID" validate:"hexadecimal,len=24"`
	StoreID   gm.UUID `json:"storeID" validate:"hexadecimal,len=24"`
	Empno     string  `json:"empno" validate:"number,min=3,max=12"`
	Password  string  `json:"password" validate:"alphanum,min=8,max=32"`
	RealName  string  `json:"realName" validate:"name,max=50"`
	Role      string  `json:"role" validate:"name,max=50"`
}

type ModAccount struct {
	ID               gm.UUID   `json:"id" validate:"hexadecimal,len=24"`
	Mobile           string    `json:"mobile" validate:"mobile"`
	Corp             string    `json:"corp" validate:"name,max=128"`
	MaxStores        int       `json:"maxStores" validate:"min=0,max=1000"`
	MaxUsersPerStore int       `json:"maxUsersPerStore" validate:"min=0,max=1000"`
	ExpiresAt        time.Time `json:"expiresAt"`
	Remarks          string    `json:"remarks" validate:"max=255"`
}

type ModStore struct {
	ID      gm.UUID `json:"id" validate:"hexadecimal,len=24"`
	Name    string  `json:"name" validate:"name,max=128"`
	Remarks string  `json:"remarks" validate:"max=255"`
}

type ModUser struct {
	ID           gm.UUID `json:"id" validate:"hexadecimal,len=24"`
	Empno        string  `json:"empno" validate:"number,min=3,max=12"`
	Password     string  `json:"password" validate:"omitempty,alphanum,min=8,max=32"` //为空时不改密码
	RealName     string  `json:"realName" validate:"name,max=50"`
	Role         string  `json:"role" validate:"name,max=50"`
	StoreID      gm.UUID `json:"storeID" validate:"hexadecimal,len=24"`
	PasswordHash gm.MD5  `json:"-"` //服务端自己生成
}

//修改管理员信息
type ModManager struct {
	ID           gm.UUID `json:"id" validate:"hexadecimal,len=24"`
	Empno        string  `json:"empno" validate:"number,min=3,max=12"`
	Password     string  `json:"password" validate:"omitempty,alphanum,min=8,max=32"` //为空时不改密码
	RealName     string  `json:"realName" validate:"name,max=50"`
	PasswordHash gm.MD5  `json:"-"` //服务端自己生成
}
