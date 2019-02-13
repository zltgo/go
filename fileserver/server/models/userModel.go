package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/zltgo/reflectx/values"

	"github.com/zltgo/api"
	"github.com/zltgo/api/session"
	"github.com/zltgo/fileserver/captcha"
	"github.com/zltgo/fileserver/server/global"
	. "github.com/zltgo/fileserver/utils"
)

//自增ID在插入时必须指定NULL
const createUserTablesIfNotExists = `create table if not exists users(
	name char(32) NOT NULL UNIQUE,
	password char(32) NOT NULL,
	salt char(32) NOT NULL,
	real_name char(16) NOT NULL,
	department varchar(255) NOT NULL,
	class char(32)  NOT NULL,
	reg_ip bigint NOT NULL,
	last_ip bigint NOT NULL,
	reg_time bigint   NOT NULL,
	last_login_time bigint NOT NULL);`

type uidDb struct {
	Uid int64 `PK:"rowid"`
}

type regDb struct {
	Name          string `name`
	Password      string `password`
	Salt          string `salt`
	RealName      string `real_name`
	Department    string `department`
	Class         string `class`
	RegIp         int64  `reg_ip`
	LastIp        int64  `last_ip`
	RegTime       int64  `reg_time`
	LastLoginTime int64  `last_login_time`
}

type modifyUsrDb struct {
	Uid        int64  `PK:"rowid"`
	Name       string `name`
	Password   string `password`
	Salt       string `salt`
	RealName   string `real_name`
	Department string `department`
	Class      string `class`
}

const (
	c_maxLoginTimes = 3 //最大登录错误次数，超过则需要验证码
	c_maxRegTimes   = 3 //最大注册错误次数，超过则需要验证码
)

func init() {
	_, err := global.Ds.Exec(createUserTablesIfNotExists)
	CheckErr(err)

	//绑定结构体与表格
	global.Ds.Register("users", usrDb{})
	global.Ds.Register("users", uidDb{})
	global.Ds.Register("users", regDb{})
	global.Ds.Register("users", modifyUsrDb{})

	//增加系统管理员用户
	var tmp regDb
	tmp.Department = "开发部"
	tmp.LastIp, _ = IPv4("127.0.0.1:8080")
	tmp.LastLoginTime = time.Now().Unix()
	tmp.Name = "admin"
	tmp.RealName = "无名"
	tmp.RegIp = tmp.LastIp
	tmp.RegTime = tmp.LastLoginTime
	tmp.Salt = RandStr(32)
	tmp.Password = MD5("admin" + tmp.Salt)
	tmp.Class = "系统管理员"
	global.Ds.Insert(&tmp)

	//增加guest用户
	tmp.Name = "guest"
	tmp.RealName = "匿名"
	tmp.Salt = RandStr(32)
	tmp.Password = MD5("guest" + tmp.Salt)
	tmp.Class = "游客"
	global.Ds.Insert(&tmp)

	return
}

//get: /
//默认访问登录页面
func Index(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	http.Redirect(w, r, "/static/sign-in.html", http.StatusFound)
	return nil, nil
}

//put:api/usr
//400：继续刷修改密码页面，错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//600：验证码不正确
//601：用户名或密码不正确
//603：用户名或密码不正确且错误次数过多，需转到有验证码的页面
func ResetPwd(se *session.Session, ub *usrDb, r *http.Request, rf *ResetPwdForm) (interface{}, error) {
	u := *ub
	//u := ctx.Value("UsrDb").(usrDb)
	if u.Name == "guest" {
		return 403, errors.New("不能修改guest用户密码")
	}

	//判断是否需要核对验证码
	num := values.AddInt(se, "PwdErrorTimes", 1)

	if num > c_maxLoginTimes {
		id := se.ValueOf("CaptchaId").String()
		se.Remove("CaptchaId")
		if captcha.Verify(id, rf.Captcha) == false {
			return 600, errors.New("验证码不正确")
		}
	}

	//判断旧密码是否正确
	if u.Password != MD5(rf.OldPassword+u.Salt) {
		if num >= c_maxLoginTimes {
			return 603, fmt.Errorf("用户%s修改密码失败，输入原密码错误", u.Name)
		}
		return 601, fmt.Errorf("用户%s修改密码失败，输入原密码错误", u.Name)
	}

	//更新登录时间和密码
	u.Password = MD5(rf.NewPassword + u.Salt)
	u.LastIp, _ = IPv4(r.RemoteAddr)
	u.LastLoginTime = time.Now().Unix()
	err := global.Ds.Update(&u)
	if err != nil {
		return 500, fmt.Errorf("数据库更新用户密码失败：%s，详细信息为：%v", err.Error(), u)
	}

	se.Remove("PwdErrorTimes")
	return 200, nil
}

//判断部门名称是否在配置的合法名称之内
func checkDepartment(d string) bool {
	for _, v := range global.Conf.Departments {
		if v == d {
			return true
		}
	}
	return false
}

//Post:api/usr
//400：继续刷注册页面，错误原因为输入参数解析失败或请求次数过多等
//500：服务器错误
//600：验证码不正确
//601：用户名已存在
//603：用户名或密码不正确且错误次数过多，需转到有验证码的页面
func Register(se *session.Session, w http.ResponseWriter, r *http.Request, rf *RegForm) (interface{}, error) {
	//判断部门名称是否在配置的合法名称之内
	if !checkDepartment(rf.Department) {
		return 400, errors.New("提交错误表单，部门名称非法：" + rf.Department)
	}

	//判断是否需要核对验证码
	num := values.AddInt(se, "RegErrorTimes", 1)
	if num > c_maxRegTimes {
		id := se.ValueOf("CaptchaId").String()
		se.Remove("CaptchaId")
		if captcha.Verify(id, rf.Captcha) == false {
			return 600, errors.New("验证码不正确")
		}
	}

	//检查用户名与密码
	var tmp regDb
	tmp.Department = rf.Department
	tmp.LastIp, _ = IPv4(r.RemoteAddr)
	tmp.LastLoginTime = time.Now().Unix()
	tmp.Name = rf.Name
	tmp.RealName = rf.RealName
	tmp.RegIp = tmp.LastIp
	tmp.RegTime = tmp.LastLoginTime
	tmp.Salt = RandStr(32)
	tmp.Password = MD5(rf.Password + tmp.Salt)
	tmp.Class = "游客"

	//插入数据库
	err := global.Ds.Insert(&tmp)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.name" {
			if num >= c_maxLoginTimes {
				return 603, errors.New("用户注册失败，" + err.Error())
			}
			return 601, errors.New("用户注册失败，" + err.Error())
		}
		return 500, fmt.Errorf("数据库Insert失败，%s，，数据详细信息：%v", err.Error(), tmp)
	}

	//需要自行转登录页面
	se.Remove("RegErrorTimes")
	return 200, nil
}

//Get: api/usr
//401,403,500
func GetUsrInfo(tmp *usrDb, r *http.Request) (interface{}, error) {
	var rv UsrInfo
	rv.CurrentIp, _ = IPv4(r.RemoteAddr)
	rv.Department = tmp.Department
	rv.Class = tmp.Class
	rv.LastIp = tmp.LastIp
	rv.LastLoginTime = tmp.LastLoginTime
	rv.Name = tmp.Name
	rv.RealName = tmp.RealName

	return rv, nil
}

//Get: api/logout
func Logout(se *session.Session, w http.ResponseWriter, r *http.Request) (interface{}, error) {
	//删除session
	se.RemoveAll()

	//重定向到登录
	http.Redirect(w, r, "/static/sign-in.html", http.StatusFound)
	return nil, nil
}

//Post: api/login
//400：继续刷登录页面，错误原因为输入参数解析失败或请求次数过多等
//500：服务器错误
//600：验证码不正确
//601：用户名或密码不正确
//603：用户名或密码不正确且错误次数过多，需转到有验证码的页面
func Login(se *session.Session, w http.ResponseWriter, r *http.Request, gf *LoginForm) (interface{}, error) {
	//判断是否需要核对验证码
	num := values.AddInt(se, "LoginErrorTimes", 1)
	if num > c_maxLoginTimes {
		id := se.ValueOf("CaptchaId").String()
		se.Remove("CaptchaId")
		if captcha.Verify(id, gf.Captcha) == false {
			return 600, errors.New("验证码不正确")
		}
	}

	//检查用户名与密码
	var tmp usrDb
	err := global.Ds.LoadRow(&tmp, "where name = ?", gf.Name)
	if err != nil && err != sql.ErrNoRows {
		return 500, errors.New("以Name=" + gf.Name + "查询数据库失败：" + err.Error())
	}
	if err == sql.ErrNoRows || tmp.Password != MD5(gf.Password+tmp.Salt) {
		if num >= c_maxLoginTimes {
			return 603, errors.New("用户名或密码不正确")
		}
		return 601, errors.New("用户名或密码不正确")
	}

	//更新登录时间
	tmp.LastIp, _ = IPv4(r.RemoteAddr)
	tmp.LastLoginTime = time.Now().Unix()
	err = global.Ds.Update(&tmp)
	if err != nil {
		err = fmt.Errorf("数据库Update失败，%s，数据详细信息：%v", err.Error(), tmp)
	}

	se.Remove("LoginErrorTimes")
	se.Remove("PwdErrorTimes")
	se.Set("Uid", tmp.Uid)
	return 200, err
}

//Get: api/captcha
//401, 500, 603
func Captcha(se *session.Session, w http.ResponseWriter, r *http.Request) (interface{}, error) {
	//读取session数据，为了设置CaptchaId
	id := captcha.New(4)
	se.Set("CaptchaId", id)

	png, err := captcha.GetImage(id, 200, 70)
	if err != nil {
		return 500, errors.New("创建验证码图片失败，" + err.Error())
	}

	w.Header().Set("Content-Type", "image/png")
	return png, nil
}

//Get: api/usrs
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
func GetUsrs(ub *usrDb, gf GetUsrsForm) (interface{}, error) {
	//ub := ctx.Value("UsrDb").(usrDb)

	var query string = "where "
	if gf.Department != "" {
		query += "department = ? "
	} else {
		query += `"" = ? `
	}
	if gf.Class != "" {
		query += "and class = ? "
	} else {
		query += `and "" = ? `
	}
	if gf.RealName != "" {
		query += "and real_name like ? "
	} else {
		query += `and "" = ? `
	}

	//第1页查询时给出总共有多少条记录
	var ufos UsrInfos
	ufos.UsrList = make([]usrDb, 0)
	var err error

	ufos.Sum, err = global.Ds.Numerical("select count(*) from users "+query, gf.Department, gf.Class, gf.RealName)
	if err != nil {
		goto ERROR500
	}

	err = global.Ds.LoadRow(&ufos.UsrList, query+" ORDER BY department, class, real_name limit ? offset ? ",
		gf.Department, gf.Class, gf.RealName, gf.OnePageCount, gf.OnePageCount*(gf.Page-1))
	if err != nil {
		goto ERROR500
	}
	return ufos, nil

ERROR500:
	return 500, fmt.Errorf("uid为%d的管理员查询用户信息失败，查询语句为%s，错误信息为%s", ub.Uid, query, err.Error())
}

//Post: api/usrs
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
//601：用户名已存在
func AddUsr(r *http.Request, rf RegForm) (interface{}, error) {
	//判断部门名称是否在配置的合法名称之内
	if !checkDepartment(rf.Department) {
		return 400, errors.New("提交错误表单，部门名称非法：" + rf.Department)
	}

	//检查用户名与密码
	var tmp regDb
	tmp.Department = rf.Department
	tmp.LastIp, _ = IPv4(r.RemoteAddr)
	tmp.LastLoginTime = time.Now().Unix()
	tmp.Name = rf.Name
	tmp.RealName = rf.RealName
	tmp.RegIp = tmp.LastIp
	tmp.RegTime = tmp.LastLoginTime
	tmp.Salt = RandStr(32)
	tmp.Password = MD5(rf.Password + tmp.Salt)
	tmp.Class = rf.Class

	//插入数据库
	err := global.Ds.Insert(&tmp)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.name" {
			return 601, errors.New(err.Error() + " " + tmp.Name)
		}
		return 500, fmt.Errorf("数据库Insert失败，%s，数据详细信息：%v", err.Error(), tmp)
	}

	return 200, nil
}

//Put: api/usrs
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
func ModifyUsr(u *usrDb, mf *ModifyUsrForm) (interface{}, error) {
	//u := ctx.Value("UsrDb").(usrDb)
	if u.Name == mf.Name {
		return 400, errors.New("不能修改自己的用户信息：" + mf.Name)
	}

	//判断部门名称是否在配置的合法名称之内
	if !checkDepartment(mf.Department) {
		return 400, errors.New("提交错误表单，部门名称非法：" + mf.Department)
	}

	//赋值
	var tmp modifyUsrDb
	tmp.Uid = mf.Uid
	tmp.Name = mf.Name
	if mf.NewPassword != "" {
		tmp.Password = MD5(mf.NewPassword + mf.Salt)
	} else {
		tmp.Password = mf.Password
	}
	tmp.RealName = mf.RealName
	tmp.Department = mf.Department
	tmp.Class = mf.Class
	tmp.Salt = mf.Salt

	//更新数据库
	err := global.Ds.Update(&tmp)
	if err == sql.ErrNoRows {
		return 400, fmt.Errorf("数据库Update失败，%s，数据详细信息：%v", err.Error(), tmp)
	}
	if err != nil {
		return 500, fmt.Errorf("数据库Update失败，%s，数据详细信息：%v", err.Error(), tmp)
	}
	return 200, nil
}

//Delete: api/usrs
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
func RemoveUsr(u *usrDb, rf *RemoveUsrsForm) (interface{}, error) {
	//u := ctx.Value("UsrDb").(usrDb)

	//赋值
	var tmp []uidDb = make([]uidDb, len(rf.UidList))
	for i, v := range rf.UidList {
		tmp[i].Uid = v
		if v == u.Uid {
			return 400, errors.New("删除的用户列表中不能包含自己：" + u.Name)
		}
	}

	//数据库删除操作
	err := global.Ds.Delete(&tmp)
	if err == sql.ErrNoRows {
		return 400, fmt.Errorf("数据库Delete失败，%s，数据详细信息：%v", err.Error(), tmp)
	}
	if err != nil {
		return 500, fmt.Errorf("数据库Delete失败，%s，数据详细信息：%v", err.Error(), tmp)
	}
	return 200, nil
}

//Get: api/conf
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
func GetConf(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	return global.Conf, nil
}

//Put: api/conf
//400：错误原因为输入参数解析失败或请求次数过多等
//401：未登录
//403：未授权
//500：服务器错误
func SetConf(r *http.Request, conf *global.Config) (interface{}, error) {
	//打开配置文件，没有则创建，权限参数待改进
	file, err := os.OpenFile(global.AppConfigPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return 500, errors.New("配置文件打开失败，" + err.Error())
	}

	defer file.Close()

	//写入配置文件
	b, err := json.Marshal(conf)
	if err != nil {
		return 500, err
	}
	_, err = file.Write(b)
	if err != nil {
		return 500, err
	}

	return 200, nil
}

var ErrNotPermitted = errors.New("userModel: request not permitted")

//获取session或uid, 失败为nil, 0
func AuthRequire(ctx *api.Context) {
	//读取session
	var se *session.Session
	ctx.MustGet(&se)

	uid, err := se.ValueOf("Uid").ToInt64()
	if err != nil {
		ctx.Render(401, errors.New("get uid failed!"))
		return
	}

	//检查权限
	ub := usrDb{}
	ub.Uid = uid
	err = global.Ds.Load(&ub)
	if err == sql.ErrNoRows {
		ctx.Render(401, fmt.Errorf("uid%d在数据库中无记录", ub.Uid))
		return
	}
	if err != nil {
		ctx.Render(500, fmt.Errorf("以Uid=%d查询数据库失败：", uid, err))
		return
	}

	ctx.Map(&ub)
	//判断是否有权限调用该请求
	urls, ok := global.GroupPattern[ub.Class]
	req := ctx.Request.Method + ":" + ctx.Request.URL.Path
	if ok {
		for _, v := range urls {
			if v.MatchString(req) {
				//ctx.Next()
				return
			}
		}
	}

	ctx.Render(403, ErrNotPermitted)
}
