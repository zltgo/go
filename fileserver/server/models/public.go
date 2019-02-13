/*
200, "成功"
400, "错误的请求"
401, "未登录"
403, "未授权"
404, "服务器找不到请求的网页"
500, "服务器内部错误"
503, "服务不可用，超载或停机维护"
600, "验证码不正确"
601, "用户名或密码不正确"
603, "多次操作失败，需输入验证码"
604, "文件扩展名非法"
605, "文件或文件夹路径错误"
*/
package models

//删除用户时 提交的表单
type RemoveUsrsForm struct {
	UidList []int64 `validate:"required,min=1,max=100"`
}

//获取图表时的表单
type GetGraphForm struct {
	Flag  string `validate:"eq=Year|eq=Month|eq=Day|eq=Weekday"`
	Year  int    `validate:"omitempty"`
	Day   int    `validate:"omitempty,min=1,max=365" default:"1"`
	Limit int    `validate:"omitempty,min=1,max=100"  default:"10"`
}

//获取下载记录时的表单
type GetCntForm struct {
	Day          int    `validate:"omitempty,min=0,max=365"`
	RealName     string `validate:"omitempty,chinese,min=1,max=16"`
	Department   string `validate:"omitempty,name,min=3,max=255"`
	Class        string `validate:"omitempty,chinese,min=1,max=32"`
	Page         int    `validate:"omitempty,min=1" default:"1"`
	OnePageCount int    `validate:"omitempty,min=1,max=200" default:"10"`
}

//获取用户信息的表单
type GetUsrsForm struct {
	RealName     string `validate:"omitempty,chinese,min=1,max=16"`
	Department   string `validate:"omitempty,name,min=3,max=255"`
	Class        string `validate:"omitempty,chinese,min=1,max=32"`
	Page         int    `validate:"omitempty,min=1" default:"1"`
	OnePageCount int    `validate:"omitempty,min=1,max=200" default:"10"`
}

//重置密码表单
type ResetPwdForm struct {
	OldPassword string `validate:"alphanum,min=5,max=32"`
	NewPassword string `validate:"alphanum,min=5,max=32"`
	Captcha     string `validate:"omitempty,alphanum,min=4,max=6"`
}

//登录表单
type LoginForm struct {
	Name     string `validate:"alphanum,min=5,max=32"`
	Password string `validate:"alphanum,min=5,max=32"`
	Captcha  string `validate:"omitempty,alphanum,min=4,max=6"`
}

//更改用户信息表单
type ModifyUsrForm struct {
	Uid         int64  `validate:"min=1"`
	Name        string `validate:"alphanum,min=5,max=32"`
	Password    string `validate:"alphanum,min=5,max=32"`
	Salt        string `validate:"alpha,min=6,max=32"`
	RealName    string `validate:"chinese,min=1,max=16"`
	Department  string `validate:"name,min=3,max=255"`
	Class       string `validate:"chinese,min=1,max=32"`
	NewPassword string `validate:"omitempty,alphanum,min=5,max=32"`
}

//注册用户表单
type RegForm struct {
	Name       string `validate:"alphanum,min=5,max=32"`
	Password   string `validate:"alphanum,min=5,max=32"`
	RealName   string `validate:"chinese,min=1,max=16"`
	Department string `validate:"name,min=3,max=255"`
	Class      string `validate:"omitempty,chinese,min=1,max=32"`
	Captcha    string `validate:"omitempty,alphanum,min=4,max=6"`
}

//用户信息结构
type UsrInfo struct {
	Name          string
	RealName      string
	Department    string
	LastIp        int64
	LastLoginTime int64
	Class         string
	CurrentIp     int64
}

//多个用户信息回答
type UsrInfos struct {
	Sum     int64
	UsrList []usrDb
}

//下载信息回答
type DownloadInfos struct {
	Sum          int64
	DownloadList []downloadDb
}

//统计信息结构
type CntInfo struct {
	IsDir    bool
	FileSize int64 //为负说明不存在
	Path     string
	Cnt      int   //下载次数
	Time     int64 //下载时间
}

//统计信息回答
type CntInfos struct {
	Sum     int64
	CntList []CntInfo
}

//图表信息回答
type GraphInfos struct {
	Flag      string //横轴类型，Year|Month|Day|Weekday
	PointList []Point
}

//图表中点的信息结构
type Point struct {
	Year    int
	Month   int
	Day     int
	Weekday int
	Cnt     int //下载次数
}

type usrDb struct {
	Uid           int64  `PK:"rowid"`
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

type downloadDb struct {
	IsDir      bool
	FileSize   int64  //为负说明不存在
	Path       string `path`
	RealName   string `real_name`
	Department string `department`
	Class      string `class`
	Ip         int64  `ip`
	Time       int64  `time`
}
