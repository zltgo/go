package wepay

import (
	"time"

	"github.com/zltgo/api/bind"

	"github.com/pkg/errors"
	"github.com/zltgo/api/client"
	"github.com/zltgo/reflectx"
)

const (
	OK          = "OK"
	FAIL        = "FAIL"
	SUCCESS     = "SUCCESS"
	NonceStrLen = 32
)

type ActOpts struct {
	Appid    string        // 公众账号ID
	MchId    string        // 商户号ID
	Secret   string        // JSAPI 接口中获取openid
	CertPath string        // cert文件路径
	KeyPath  string        // key文件路径
	TimeOut  time.Duration //请求超时时间
	SignOpts
}

//微信支付账户
type Account struct {
	appid       string        // 公众账号ID
	apiKey      string        // 支付密钥
	mchId       string        // 商户号ID
	secret      string        // JSAPI 接口中获取openid
	httpsClient client.Client // 双向证书链接
	*Signer                   //签名
	signType    string
	fm          reflectx.FormMapper
}

func NewAccount(opts ActOpts) (*Account, error) {
	//create https 双向请求客户端
	c, err := NewHttpsClient(opts.CertPath, opts.KeyPath, opts.TimeOut)
	if err != nil {
		return nil, errors.Wrap(err, "NewHttpsClient")
	}

	return &Account{
		appid:       opts.Appid,
		apiKey:      opts.ApiKey,
		mchId:       opts.MchId,
		secret:      opts.Secret,
		httpsClient: client.Client{c},
		Signer:      NewSigner(opts.SignOpts),
		signType:    opts.SignType,
		fm:          reflectx.NewFormMapper(opts.TagName, reflectx.FieldNameToUnderscore),
	}, nil
}

//FillBaseParams writes the Appid,MchId, Sign, SignType,NonceStr of the struct.
func (a *Account) FillBaseParams(ptr interface{}) {
	mp := map[string][]string{
		"appid":     []string{a.appid},
		"mch_id":    []string{a.mchId},
		"nonce_str": []string{RandString(NonceStrLen)},
		"sign_type": []string{a.signType},
	}
	if err := a.fm.FormToStruct(mp, ptr); err != nil {
		panic(err)
	}
	//write sign field.
	a.Sign(ptr)
	return
}

//https no cert post
func (a *Account) PostWithoutCert(urlStr string, inPtr, outPtr interface{}) error {
	//fill base parameters
	a.FillBaseParams(inPtr)

	//validate params
	if err := bind.Validate(inPtr); err != nil {
		return errors.Wrap(err, "input parameters")
	}

	//create request
	req, err := client.NewXmlRequest("POST", urlStr, inPtr)
	if err != nil {
		return errors.Wrap(err, "NewXmlRequest")
	}
	req.Header.Set("Accept", "application/xml")

	//post and get restult
	status, err := client.Default.Exec(req, outPtr)
	if err != nil {
		return errors.Wrapf(err, "https post error, status=%v", status)
	}

	//validate result
	if err := bind.Validate(outPtr); err != nil {
		return errors.Wrap(err, "output parameters")
	}

	//verify return info
	var rv ReturnInfo
	reflectx.CopyStruct(&rv, outPtr)
	if rv.ReturnCode != SUCCESS {
		return errors.Errorf("return_code is not success: %v", rv.ReturnMsg)
	}
	//verify signature
	if !a.Verify(outPtr) {
		return errors.Errorf("failed to verify sign")
	}
	//verify result info?
	// if rv.ResultCode != SUCCESS {
	// 	return outPtr, errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", rv.ErrCode, rv.ErrCodeDes)
	// }

	return nil
}

//https no cert post
func (a *Account) PostWithCert(urlStr string, inPtr, outPtr interface{}) error {
	//validate params
	if err := bind.Validate(inPtr); err != nil {
		return err
	}

	//create request
	req, err := client.NewXmlRequest("POST", urlStr, inPtr)
	if err != nil {
		return errors.Wrap(err, "NewXmlRequest")
	}
	req.Header.Set("Accept", "application/xml")

	//post and get restult
	status, err := a.httpsClient.Exec(req, outPtr)
	if err != nil {
		return errors.Wrapf(err, "https post error, status=%v", status)
	}

	//verify return info
	var rv ReturnInfo
	reflectx.CopyStruct(&rv, outPtr)
	if rv.ReturnCode != SUCCESS {
		return errors.Errorf("return_code is not success: %v", rv.ReturnMsg)
	}
	//verify signature
	if !a.Verify(outPtr) {
		return errors.Errorf("failed to verify sign")
	}
	//verify result info?
	// if rv.ResultCode != SUCCESS {
	// 	return &rv, errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", rv.ErrCode, rv.ErrCodeDes)
	// }

	return nil
}
