package model

import (
	"errors"
	"net/http"
	"time"

	"github.com/zltgo/api"
	"github.com/zltgo/api/jwt"
	"github.com/zltgo/api/session"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// collection: managers
type ManagerDb struct {
	Id      bson.ObjectId `bson:"_id"`
	OpenId  string        `json:"-"` // 用于登录验证，修改密码后会重新生成
	Usr     string
	HashPwd string `json:"-"`
	RegTime time.Time

	Grp    string // 用户组，如系统管理员，普通管理员，操作员
	AreaId string // 权限区域，为node.RootId表示可管理全地区
}

type putMngDb struct {
	OpenId string `bson:",omitempty"`
	Usr    string `bson:",omitempty"`
	Pwd    string `bson:",omitempty"`
	Grp    string `bson:",omitempty"`
	AreaId string `bson:",omitempty"`
}

//管理主页重定向
func Manage(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	http.Redirect(w, r, "/static/manage.html", http.StatusMovedPermanently)
	return nil, nil
}

// 登录：POST:/login/manager
// 错误码：400,401,403,429,500,800,802,803
// 输入：LoginMngForm
// 返回值：
// type AuthToken struct {
//	 AccessToken  string
//	 MaxAge       int
//	 RefreshToken string
// }
// AccessToken用于身份验证，客户端自己保存，每次请求都要放入http头部中：
// http.Header.Set("ACCESS-TOKEN", AccessToken)
// MaxAge为AccessToken的有效期，单位为秒(默认值为半小时)，过期后需使用RefreshToken
// 获取新的AccessToken，参见"GET:/token"
func ManagerLogin(db *mgo.Database, se session.Session, lf *LoginMngForm) (int, string) {
	captcha := se.GetString(KeyCaptcha)
	se.Del(KeyCaptcha)
	if captcha != lf.Captcha {
		return StatusCaptchaError, "图形验证码不正确"
	}

	// 读取 managers 集合
	var mb ManagerDb
	if err := db.C(Managers).Find(bson.M{"usr": lf.Usr}).One(&mb); err != nil {
		if err == mgo.ErrNotFound {
			return StatusNotRegistered, lf.Usr + "：尚未注册"
		} else {
			return http.StatusInternalServerError, err.Error()
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(mb.HashPwd), []byte(lf.Pwd)); err != nil {
		return StatusPwdError, lf.Usr + "：密码不正确"
	}
	return http.StatusOK, mb.OpenId
}

// 登录：POST:/manager
// 错误码：400,401,403,429,500,801
// 输入：AddMngForm
// 返回值：新增的管理员ID
func AddManager(db *mgo.Database, af *AddMngForm) (interface{}, error) {
	// 计算新密码
	hashed, err := bcrypt.GenerateFromPassword([]byte(af.Pwd), 0)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	id := bson.NewObjectId()
	if err := db.C(Managers).Insert(ManagerDb{
		Id:      id,
		OpenId:  bson.NewObjectId().Hex(),
		Usr:     af.Usr,
		HashPwd: string(hashed),
		RegTime: time.Now(),
		Grp:     af.Grp,
		AreaId:  af.AreaId,
	}); err != nil {
		if mgo.IsDup(err) {
			return StatusRegisteredYet, err
		}
		return http.StatusInternalServerError, err
	}

	return id.String(), nil
}

// 修改管理员信息：PUT:/manager
// 错误码：400,401,403,429,500,800
// 输入：PutMngForm
// 返回值：<nil>
func ModifyManager(db *mgo.Database, pf *PutMngForm) (int, error) {
	pd := putMngDb{
		Usr:    pf.Usr,
		Grp:    pf.Grp,
		AreaId: pf.AreaId,
	}
	// 密码不为空则计算新密码，并更新OpenId
	if pf.Pwd != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(pf.Pwd), 0)
		if err != nil {
			return http.StatusInternalServerError, err
		}
		pd.Pwd = string(hashed)
		pd.OpenId = bson.NewObjectId().Hex()
	}

	if err := db.C(Managers).UpdateId(bson.ObjectIdHex(pf.Id), bson.M{"$set": &pd}); err != nil {
		if err == mgo.ErrNotFound {
			return StatusNotRegistered, err
		}
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

// 获取管理员列表：GET:/managers
// 错误码：400,401,403,429,500
// 输入：无
// 返回值：[]ManagerDb
// type ManagerDb struct {
//	 Id      bson.ObjectId `bson:"_id"`
//	 Usr     string
//	 RegTime time.Time
//	 Grp    string // 用户组，如系统管理员，普通管理员，操作员
//	 AreaId string // 权限区域，为node.RootId表示可管理全地区
// }
func GetManagers(db *mgo.Database, gf *GetMngForm) (interface{}, error) {
	// 构建正则表达式，类似于 like %Usr%
	if gf.UsrName != "" {
		gf.Usr = &bson.RegEx{gf.UsrName, "i"}
	}

	var mngs []ManagerDb
	if err := db.C(Managers).Find(gf).Sort("-regtime").All(&mngs); err != nil {
		if err == mgo.ErrNotFound {
			return http.StatusOK, nil
		}
		return http.StatusInternalServerError, err
	}

	return mngs, nil
}

// 修改管理员信息：DELETE:/manager/:id
// 错误码：400,401,403,429,500,800
// 输入：无
// 返回值：<nil>
func RemoveManager(db *mgo.Database, ctx *api.Context) (int, error) {
	id := ctx.Params.Get("id")
	if !bson.IsObjectIdHex(id) {
		return http.StatusBadRequest, errors.New("id of manager is not valid")
	}
	if err := db.C(Managers).RemoveId(bson.ObjectIdHex(id)); err != nil {
		if err == mgo.ErrNotFound {
			return StatusNotRegistered, err
		}
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

// 通过uid获取managerDb,获取到的managerDb为缓存的指针，只读
func GetManagerDb(db *mgo.Database, uid jwt.UID) (mb *ManagerDb, code int, err error) {
	v := LruMemCache.Getsert(uid, func() interface{} {
		// 读取数据库用户信息
		var mng ManagerDb
		if err = db.C(Managers).Find(bson.M{"openid": uid}).One(&mng); err != nil {
			if err == mgo.ErrNotFound {
				code = http.StatusUnauthorized
			} else {
				code = http.StatusInternalServerError
			}
			return nil
		}
		return &mng
	})

	if v != nil {
		mb = v.(*ManagerDb)
		code = http.StatusOK
	}
	return
}
