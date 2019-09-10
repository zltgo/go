package model

import (
	"gopkg.in/mgo.v2/bson"
)

var (
	// 通用错误码比public多一个
	// 403 没有权限访问

	//自定义错误码
	StatusDbNotFound = 900 // 数据库记录不存在或已被删除
	StatusDbDup      = 901 // 数据库插入或更新失败，唯一键冲突
	StatusAreaErr    = 902 // 区域信息错误，不存在或者超过管理范围
)

//***************************************************************************
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
type LoginMngForm struct {
	Usr     string `validate:"name,min=2,max=32"`
	Pwd     string `validate:"alphanum,min=8,max=32"`
	Captcha string `validate:"min=4,max=6"`
}

//***************************************************************************
// 登录：POST:/manager
// 错误码：400,401,403,429,500,801
// 输入：AddMngForm
// 返回值：新增的管理员ID
type AddMngForm struct {
	Usr    string `validate:"name,min=2,max=32"`
	Pwd    string `validate:"alphanum,min=8,max=32"`
	Grp    string `validate:"name,min=2,max=32"`
	AreaId string `validate:"hexadecimal,len=24"`
}

//***************************************************************************
// 修改管理员信息：PUT:/manager
// 错误码：400,401,403,429,500,800
// 输入：PutMngForm
// 返回值：<nil>
type PutMngForm struct {
	Id     string `validate:"hexadecimal,len=24"`
	Usr    string `validate:"omitempty,name,min=2,max=32"`
	Pwd    string `validate:"omitempty,alphanum,min=8,max=32"`
	Grp    string `validate:"omitempty,name,min=2,max=32"`
	AreaId string `validate:"omitempty,hexadecimal,len=24"`
}

//***************************************************************************
// 修改管理员信息：DELETE:/manager
// 错误码：400,401,403,429,500,800
// 输入：PutMngForm
// 返回值：<nil>

//***************************************************************************
// 获取管理员列表：GET:/managers
// 错误码：400,401,403,429,500
// 输入：无
// 返回值：[]managerDb
// type managerDb struct {
//	 Id      bson.ObjectId `bson:"_id"`
//	 Usr     string
//	 RegTime time.Time
//	 Grp    string  // 用户组，如系统管理员，普通管理员，操作员
//	 AreaId string // 权限区域，为node.RootId表示可管理全地区
// }
type GetMngForm struct {
	Usr *bson.RegEx `bson:",omitempty"`
	//用户名关键词，模糊查询，不区分大小写
	UsrName string `bson:"-" validate:"omitempty,name,min=1,max=32"`
	Grp     string `bson:",omitempty" validate:"omitempty,name,min=2,max=32"`
	AreaId  string `bson:",omitempty" validate:"omitempty,hexadecimal,len=24"`
}

// 对节点进行増删改查时的表单
type NodeForm struct {
	Id   string `validate:"omitempty,hexadecimal,len=24"`
	Pid  string `validate:"omitempty,hexadecimal,len=24"`
	Name string `validate:"omitempty,name,max=150"`
}
