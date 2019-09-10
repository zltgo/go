package model

import (
	"errors"
	. "fisheep/conf"
	"net/http"
	"time"

	"github.com/zltgo/api/jwt"
	"github.com/zltgo/api/session"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Addr
type Addr struct {
	AreaId string
	//联系名，快递用
	Name   string
	Phone  string
	Detail string //"某小区A栋2201"
}

// collection: users
type UsrDb struct {
	Id      bson.ObjectId `bson:"_id"`
	OpenId  string        //用于登录验证，修改密码后会重新生成
	Phone   string
	HashPwd string
	RegTime time.Time
	Addrs   []Addr
}

type garbageUsrDb struct {
	Ub      UsrDb `bson:",inline"`
	Garbage string
}

type smsDb struct {
	Phone string `bson:"_id"`
	//手机短信验证码及时间(秒)
	Sms   string
	SmsAt int64
	Daily int
}

// 登录：POST:/login/usr
// 错误码：400,429,500,800,802,803,804
// 输入：UsrForm
// Pwd不为空时，使用Phone、Pwd验证, 否则使用Phone和Sms
// 短信验证方式下，如果用户不存在，直接注册一个新用户
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
func UsrLogin(db *mgo.Database, uf *UsrForm) (int, string) {
	if uf.Pwd == "" && uf.Sms == "" {
		return http.StatusBadRequest, "密码和短信验证码不能都为空"
	}

	var ub UsrDb
	// 密码验证
	if uf.Pwd != "" {
		// 读取 users 集合
		if err := db.C(Users).Find(bson.M{"phone": uf.Phone}).One(&ub); err != nil {
			if err == mgo.ErrNotFound {
				return StatusNotRegistered, uf.Phone + "：尚未注册"
			} else {
				return http.StatusInternalServerError, err.Error()
			}
		}

		if err := bcrypt.CompareHashAndPassword([]byte(ub.HashPwd), []byte(uf.Pwd)); err != nil {
			return StatusPwdError, uf.Phone + "：密码不正确"
		}
	} else {
		// 手机短信验证
		if code, err := checkSms(uf.Phone, uf.Sms, db); err != nil {
			return code, err.Error()
		}

		// 读取 users 集合
		if err := getsertUsrDb(db, uf.Phone, &ub); err != nil {
			return http.StatusInternalServerError, err.Error()
		}
	}

	// 写入缓存并返回OpenId
	LruMemCache.SetObj(jwt.UID(ub.OpenId), &ub)
	return http.StatusOK, ub.OpenId
}

// 设置配送地址：PUT:/usr/addrs
// 错误码：400,401,429,500
// 输入：AddrsForm
// 返回值：<nil>
// 注意事项：需给出编辑后的所有地址（最多8个），会替换而非添加
func SetAddrs(uid jwt.UID, db *mgo.Database, af *AddrsForm) (int, error) {
	var ub UsrDb
	if _, err := db.C(Users).Find(bson.M{"openid": uid}).Apply(mgo.Change{
		ReturnNew: true,
		Update: bson.M{
			"$set": bson.M{"addrs": af.Addrs},
			//删除填充字段
			"$unset": bson.M{"garbage": true},
		}}, &ub); err != nil {
		// 有可能用户在其他设备更好密码或手机，会导致openid失效
		if err == mgo.ErrNotFound {
			return http.StatusUnauthorized, err
		}
		return http.StatusInternalServerError, err
	}

	// 更新缓存
	LruMemCache.SetObj(jwt.UID(ub.OpenId), &ub)
	return http.StatusOK, nil
}

// 重置手机号：PUT:/usr/phone
// 错误码：400,401,429,500,801,802,803,804
// 输入：ResetPhoneForm
// 返回值：同"POST:/login"
// 注意事项：前提条件是已登录
func ResetPhone(uid jwt.UID, db *mgo.Database, se session.Session, rf *ResetPhoneForm) (int, string) {
	// 先核对验证码
	captcha := se.GetString(KeyCaptcha)
	se.Del(KeyCaptcha)
	if captcha != rf.Captcha {
		return StatusCaptchaError, "图形验证码不正确"
	}

	// 获取用户信息
	ub, code, err := getUsrDb(db, uid)
	if err != nil {
		return code, err.Error()
	}

	// 核对密码
	if err = bcrypt.CompareHashAndPassword([]byte(ub.HashPwd), []byte(rf.Pwd)); err != nil {
		return StatusPwdError, "密码不正确"
	}

	// 验证新手机短信
	if code, err := checkSms(rf.Phone, rf.Sms, db); err != nil {
		return code, err.Error()
	}

	// 更新到数据库，重新生成OpenId，使所有之前生成的AuthToken失效
	var newUb UsrDb
	if _, err = db.C(Users).FindId(ub.Id).Apply(mgo.Change{
		ReturnNew: true,
		Update: bson.M{
			"$set": bson.M{
				"phone":  rf.Phone,
				"openid": bson.NewObjectId().Hex(),
			}}}, &newUb); err != nil {
		// 有可能用户在函数执行期间被删除
		if err == mgo.ErrNotFound {
			return http.StatusUnauthorized, err.Error()
		}
		// 新的手机号码已经注册了
		if mgo.IsDup(err) {
			return StatusRegisteredYet, err.Error()
		}
		return http.StatusInternalServerError, err.Error()
	}

	LruMemCache.SetObj(jwt.UID(newUb.OpenId), &newUb)
	return http.StatusOK, newUb.OpenId
}

// 修改密码：PUT:/usr/pwd
// 错误码：400,401,429,500,802,804
// 输入：ResetPwdForm
// 返回值：同"POST:/login"
// 注意事项：前提条件是已登录
func ResetPwd(uid jwt.UID, db *mgo.Database, se session.Session, rf *ResetPwdForm) (int, string) {
	// 核对验证码
	captcha := se.GetString(KeyCaptcha)
	se.Del(KeyCaptcha)
	if captcha != rf.Captcha {
		return StatusCaptchaError, "图形验证码不正确"
	}

	// 获取用户信息
	ub, code, err := getUsrDb(db, uid)
	if err != nil {
		return code, err.Error()
	}

	// 手机短信验证
	if code, err := checkSms(ub.Phone, rf.Sms, db); err != nil {
		return code, err.Error()
	}

	// 计算新密码
	hashed, err := bcrypt.GenerateFromPassword([]byte(rf.Pwd), 0)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}
	// 更新到数据库，重新生成OpenId，使所有之前生成的AuthToken失效
	var newUb UsrDb
	if _, err = db.C(Users).FindId(ub.Id).Apply(mgo.Change{
		ReturnNew: true,
		Update: bson.M{
			"$set": bson.M{
				"hashpwd": string(hashed),
				"openid":  bson.NewObjectId().Hex(),
			}}}, &newUb); err != nil {
		// 有可能用户在函数执行期间被删除
		if err == mgo.ErrNotFound {
			return http.StatusUnauthorized, err.Error()
		}
		return http.StatusInternalServerError, err.Error()
	}

	LruMemCache.SetObj(jwt.UID(newUb.OpenId), &newUb)
	return http.StatusOK, newUb.OpenId
}

// 通过uid获取usrDb,获取到的usrDb为缓存的指针，只读
func getUsrDb(db *mgo.Database, uid jwt.UID) (ub *UsrDb, code int, err error) {
	v := LruMemCache.Getsert(uid, func() interface{} {
		// 读取数据库用户信息
		var usr UsrDb
		if err = db.C(Users).Find(bson.M{"openid": uid}).One(&usr); err != nil {
			if err == mgo.ErrNotFound {
				code = http.StatusUnauthorized
			} else {
				code = http.StatusInternalServerError
			}
			return nil
		}

		return &usr
	})

	if v != nil {
		ub = v.(*UsrDb)
		code = http.StatusOK
	}
	return
}

// 检查短信验证码是否正确
func checkSms(phone, sms string, db *mgo.Database) (int, error) {
	var sb smsDb
	if err := db.C(Smses).FindId(phone).One(&sb); err != nil {
		if err == mgo.ErrNotFound {
			return StatusSmsError, errors.New(phone + "：短信验证码不存在")
		} else {
			return http.StatusInternalServerError, err
		}
	}
	if sms != sb.Sms {
		return StatusSmsError, errors.New(phone + "：短信验证码不正确")
	}
	d := time.Now().Unix() - sb.SmsAt
	if d < 0 || d > int64(Pub.SmsMaxAge) {
		return StatusSmsError, errors.New(phone + "：短信验证码已过期")
	}
	return http.StatusOK, nil
}

// 通过手机号获取用户信息，未找到则注册一个新用户
func getsertUsrDb(db *mgo.Database, phone string, ub *UsrDb) error {
	// 读取 users 集合
	if err := db.C(Users).Find(bson.M{"phone": phone}).One(ub); err != nil {
		if err != mgo.ErrNotFound {
			return err
		}

		// 注册一个新用户, 初始密码随机
		hashed, err := bcrypt.GenerateFromPassword(jwt.RandBytes(32), 0)
		if err != nil {
			return err
		}
		ub.Id = bson.NewObjectId()
		ub.OpenId = bson.NewObjectId().Hex()
		ub.Phone = phone
		ub.HashPwd = string(hashed)
		ub.RegTime = time.Now()

		//插入users集合，最后一个字段为填充占位，添加地址时删除
		if _, err := db.C(Users).Find(bson.M{"phone": phone}).Apply(mgo.Change{
			Upsert:    true,
			ReturnNew: true,
			Update: garbageUsrDb{
				Ub: *ub,
				//填充占位，添加地址时删除
				Garbage: "11111111112222222222333333333344444444445555555555" +
					"11111111112222222222333333333344444444445555555555" +
					"11111111112222222222333333333344444444445555555555",
			}}, ub); err != nil {
			return err
		}
	}
	return nil
}

// 显示商品：GET:/goods
// 错误码：400,401,403,429,500
// 输入：GetGoodsForm
// 返回值：[]goodsDb
func ViewGoods(db *mgo.Database, gf *GetGoodsForm) (interface{}, error) {
	// 用户只能看上架的
	gf.Flag = 1

	// 构建正则表达式，类似于 like %Usr%
	if gf.GoodsName != "" {
		gf.Title = &bson.RegEx{gf.GoodsName, "i"}
	}

	var gs []GoodsDb
	if err := db.C(Goods).Find(gf).Limit(gf.Cnt).Skip(gf.Page * gf.Cnt).Sort(gf.Sort...).All(&gs); err != nil {
		if err == mgo.ErrNotFound {
			return http.StatusOK, nil
		}
		return http.StatusInternalServerError, err
	}

	return gs, nil
}
