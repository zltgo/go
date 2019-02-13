package wepay

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/zltgo/api/bind"
	"github.com/zltgo/api/client"
)

const (
	//参数顺序不能错
	OauthUrl        = "https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect"
	AccessTokenUrl  = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"
	RefreshTokenUrl = "https://api.weixin.qq.com/sns/oauth2/refresh_token?appid=%s&grant_type=refresh_token&refresh_token=%s"
	CheckTokenUrl   = "https://api.weixin.qq.com/sns/auth?access_token=%s&openid=%s"
	UserInfoUrl     = "https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN"
)

type OauthReq struct {
	Appid string `validate:"required,max=32"` //微信分配的公众账号ID

	//授权后重定向的回调链接地址，请使用urlencode对链接进行处理
	RedirectUrl string `validate:"required,url,max=256"`

	//应用授权作用域，snsapi_base ，不弹出授权页面，直接跳转，只能获取用户openid.
	//snsapi_userinfo ，弹出授权页面，可通过openid拿到昵称、性别、所在地。
	//并且，即使在未关注的情况下，只要用户授权，也能获取其信息。
	Scope string `validate:"eq=snsapi_base|eq=snsapi_userinfo"`

	//重定向后会带上state参数，开发者可以填写a-zA-Z0-9的参数值，最多128字节
	State string `validate:"omitempty,alphanum,max=128"`
}

type OauthResult struct {
	Code string `validate:"required"` //TODO
	//重定向后会带上state参数，开发者可以填写a-zA-Z0-9的参数值，最多128字节
	State string `validate:"omitempty,alphanum,max=128"`
}

//获取到网页授权access_token请求参数
type AccessTokenReq struct {
	Appid  string //微信分配的公众账号ID
	Code   string // Oauth获取的code参数
	Secret string //公众号的appsecret
}

// 微信返回的通用错误json
type ErrorInfo struct {
	ErrCode int64  `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

//获取用户授权access_token的返回结果
type AccessTokenResult struct {
	ErrorInfo
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`                         //access_token超时时间，单位（秒）
	OpenID       string `json:"openid" validate:"required,max=128"` //用户唯一标识，在未关注公众号时，也会产生一个用户和公众号唯一的OpenID
	Scope        string `json:"scope"`                              //用户授权的作用域，使用逗号（,）分隔
}

//UserInfo 用户授权获取到用户信息
type UserInfo struct {
	ErrorInfo
	OpenID   string `json:"openid" validate:"required,max=128"`
	Nickname string `json:"nickname"`                    //用户昵称
	Sex      int32  `json:"sex"  validate:"min=0,max=2"` //用户的性别，值为1时是男性，值为2时是女性，值为0时是未知
	Province string `json:"province"`                    //用户个人资料填写的省份
	City     string `json:"city"`                        //普通用户个人资料填写的城市
	Country  string `json:"country"`                     //国家，如中国为CN

	//用户头像，最后一个数值代表正方形头像大小（有0、46、64、96、132数值可选，0代表640*640正方形头像），
	//用户没有头像时该项为空。若用户更换头像，原有头像URL将失效。
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"` //用户特权信息，json 数组，如微信沃卡用户为（chinaunicom）
	Unionid    string   `json:"unionid"`   //只有在用户将公众号绑定到微信开放平台帐号后，才会出现该字段。
}

//跳转到wechat网页授权。
//如果用户同意授权，页面将跳转至 redirect_uri/?code=CODE&state=STATE。
//code说明 ： code作为换取access_token的票据，每次用户授权带上的code将不一样，
//code只能使用一次，5分钟未被使用自动过期。
func Oauth(w http.ResponseWriter, r *http.Request, oh *OauthReq) error {
	if err := bind.Validate(oh); err != nil {
		return err
	}
	redirectUrl := url.QueryEscape(oh.RedirectUrl)
	oauthUrl := fmt.Sprintf(OauthUrl, oh.Appid, redirectUrl, oh.Scope, oh.State)

	http.Redirect(w, r, oauthUrl, http.StatusFound)
	return nil
}

// 通过网页授权的code 换取access_token
func GetAccessToken(ar *AccessTokenReq) (*AccessTokenResult, error) {
	urlStr := fmt.Sprintf(AccessTokenUrl, ar.Appid, ar.Secret, ar.Code)
	rv := &AccessTokenResult{}

	// https get
	if status, err := client.Default.Get(urlStr, nil, rv); err != nil {
		return nil, errors.Errorf("http get error, url=%v, status=%v, error=%v", urlStr, status, err)
	}

	if err := bind.Validate(rv); err != nil {
		return nil, err
	}

	if rv.ErrCode != 0 {
		return nil, errors.Errorf("returned from wechat: errcode=%v , errmsg=%v", rv.ErrCode, rv.ErrMsg)
	}
	return rv, nil
}

// 刷新access_token
func RefreshToken(appid, refreshToken string) (*AccessTokenResult, error) {
	urlStr := fmt.Sprintf(RefreshTokenUrl, appid, refreshToken)
	rv := &AccessTokenResult{}

	// https get
	if status, err := client.Default.Get(urlStr, nil, rv); err != nil {
		return nil, errors.Errorf("http get error, url=%v, status=%v, error=%v", urlStr, status, err)
	}

	if err := bind.Validate(rv); err != nil {
		return nil, err
	}

	if rv.ErrCode != 0 {
		return nil, errors.Errorf("returned from wechat: errcode=%v , errmsg=%v", rv.ErrCode, rv.ErrMsg)
	}
	return rv, nil
}

//检验access_token是否有效
func CheckAccessToken(accessToken, openid string) (bool, error) {
	urlStr := fmt.Sprintf(CheckTokenUrl, accessToken, openid)

	var result ErrorInfo
	// https get
	if status, err := client.Default.Get(urlStr, nil, &result); err != nil {
		return false, errors.Errorf("http get error, url=%v, status=%v, error=%v", urlStr, status, err)
	}

	return result.ErrCode == 0, nil
}

//如果网页授权作用域为snsapi_userinfo，则此时开发者可以通过access_token和openid拉取用户信息。
func GetUserInfo(accessToken, openid string) (result *UserInfo, err error) {
	urlStr := fmt.Sprintf(UserInfoUrl, accessToken, openid)
	rv := &UserInfo{}

	// https get
	if status, err := client.Default.Get(urlStr, nil, rv); err != nil {
		return nil, errors.Errorf("http get error, url=%v, status=%v, error=%v", urlStr, status, err)
	}

	if err := bind.Validate(rv); err != nil {
		return nil, err
	}

	if rv.ErrCode != 0 {
		return nil, errors.Errorf("returned from wechat: errcode=%v , errmsg=%v", rv.ErrCode, rv.ErrMsg)
	}
	return rv, nil
}
