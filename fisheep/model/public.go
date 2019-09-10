package model

var (
	// 使用到的通用错误码有五个
	// 400 输入参数不正确
	// 401 未登录，转到登录页面
	// 404 资源不存在或已删除
	// 429 请求过于频繁，请稍后再试
	// 500 服务器内部错误，请重试

	//自定义错误码
	StatusNotRegistered = 800 // 用户尚未注册
	StatusRegisteredYet = 801 // 用户已注册
	StatusCaptchaError  = 802 // 图形验证码不正确
	StatusPwdError      = 803 // 密码不正确
	StatusSmsError      = 804 // 短信验证码不正确或已过期
)

//***************************************************************************
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
type UsrForm struct {
	Phone string `validate:"moblie,len=11"`
	Pwd   string `validate:"omitempty,alphanum,min=8,max=32"`
	Sms   string `validate:"omitempty,alphanum,min=4,max=6"`
}

//***************************************************************************
// 刷新AuthToken：GET:/token
// 错误码：401,500
// 输入：
// 将RefreshToken放入http头部中：http.Header.Set("REFRESH-TOKEN", RefreshToken)
// 返回值：
// type AuthToken struct {
//	 AccessToken  string
//	 MaxAge       int
//	 RefreshToken string
// }
// 其中AccessToken为新的，RefreshToken不变
// RefreshToken的有效时间较长，一般为30天或永久有效，失效后需重新登录获取

//***************************************************************************
// 重置密码：PUT:/usr/pwd
// 错误码：400,401,429,500,802,804
// 输入：ResetPwdForm
// 返回值：同"POST:/login"
// 注意事项：前提条件是已登录
type ResetPwdForm struct {
	Pwd     string `validate:"alphanum,min=8,max=32"`
	Sms     string `validate:"alphanum,min=4,max=6"`
	Captcha string `validate:"alphanum,min=4,max=6"`
}

//***************************************************************************
// 重置手机号：PUT:/usr/phone
// 错误码：400,401,429,500,801,802,803,804
// 输入：ResetPhoneForm
// 返回值：同"POST:/login"
// 注意事项：前提条件是已登录
type ResetPhoneForm struct {
	Phone   string `validate:"moblie,len=11"`
	Pwd     string `validate:"alphanum,min=8,max=32"`
	Sms     string `validate:"alphanum,min=4,max=6"`
	Captcha string `validate:"alphanum,min=4,max=6"`
}

//***************************************************************************
// 设置配送地址：PUT:/usr/addrs
// 错误码：400,401,429,500
// 输入：AddrsForm
// 返回值：<nil>
// 注意事项：需给出编辑后的所有地址（最多8个），会替换而非添加
type AddrsForm struct {
	Addrs []Address `validate:"omitempty,max=8,dive"`
}

type Address struct {
	AreaId string `validate:"hexadecimal,len=24"`
	//联系名和联系手机
	Name  string `validate:"chinese,max=32"`
	Phone string `validate:"moblie,len=11"`
	//详细地址，例如某小区A栋2201
	Detail string `validate:"name,max=150"`
}

//***************************************************************************
// 获取图形验证码：GET:/captcha
// 错误码：429,500
// 输入：无
// 返回值：png（4-6个字符组成的验证码图片）

//***************************************************************************
// 测试图形验证码：GET:/captcha
// 错误码：429,500
// 输入：无
// 返回值：string，用于测试
// 注意事项：在配置文件中的GinMode为"test"时才有效

//***************************************************************************
// 测试短信验证码：GET:/test/sms
// 错误码：400,429,500,800
// 输入：UsrForm(只需提交手机号即可)
// 返回值：string，用于测试
// 注意事项：同一个手机号请求短信验证码有最小时间间隔限制和每日请求次数限制
