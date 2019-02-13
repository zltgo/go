package wepay

import (
	"net/url"

	"github.com/pkg/errors"
	"github.com/zltgo/api/bind"
	"github.com/zltgo/api/client"
)

const (
	//支付API
	MicroPayUrl         = "https://api.mch.weixin.qq.com/pay/micropay"
	UnifiedOrderUrl     = "https://api.mch.weixin.qq.com/pay/unifiedorder"
	OrderQueryUrl       = "https://api.mch.weixin.qq.com/pay/orderquery"
	ReverseUrl          = "https://api.mch.weixin.qq.com/secapi/pay/reverse" //TODO
	CloseOrderUrl       = "https://api.mch.weixin.qq.com/pay/closeorder"
	RefundUrl           = "https://api.mch.weixin.qq.com/secapi/pay/refund"
	RefundQueryUrl      = "https://api.mch.weixin.qq.com/pay/refundquery"
	DownloadBillUrl     = "https://api.mch.weixin.qq.com/pay/downloadbill"
	ReportUrl           = "https://api.mch.weixin.qq.com/payitil/report"
	ShortUrl            = "https://api.mch.weixin.qq.com/tools/shorturl"
	AuthCodeToOpenidUrl = "https://api.mch.weixin.qq.com/tools/authcodetoopenid"
)

//共用参数
type Base struct {
	Appid    string `xml:"appid" validate:"required,max=32"`                     //微信分配的公众账号ID
	MchId    string `xml:"mch_id" validate:"required,max=32"`                    //微信支付分配的商户号
	NonceStr string `xml:"nonce_str" validate:"required,max=32"`                 //随机字符串
	Sign     string `xml:"sign" validate:"len=32|len=64"`                        //签名
	SignType string `xml:"sign_type,omitempty" validate:"eq=MD5|eq=HMAC-SHA256"` //签名算法
}

//统一下单请求参数
type UnifiedOrderReq struct {
	Base
	Attach         string `xml:"attach,omitempty" validate:"max=127"`               //附加数据，原样返回
	Body           string `xml:"body" validate:"required,max=127"`                  //商品描述
	Detail         string `xml:"detail,omitempty"`                                  //TODO
	DeviceInfo     string `xml:"device_info,omitempty" validate:"max=32"`           //微信支付分配的终端设备号
	GoodsTag       string `xml:"goods_tag,omitempty" validate:"max=127"`            //商品标记，与代金券有关，不使用请填空
	NotifyUrl      string `xml:"notify_url" validate:"omitempty,max=256"`           //接收微信支付成功通知
	Openid         string `xml:"openid" validate:"max=128"`                         //用户在商户appid 下的唯一标识，trade_type 为JSAPI时，此参数必传
	OutTradeNo     string `xml:"out_trade_no" validate:"required,max=32"`           //商户系统内部的订单号,可包含字母,确保在商户系统唯一
	ProductId      string `xml:"product_id,omitempty" validate:"max=32"`            //只在trade_type 为NATIVE时需要填写。此id 为二维码中包含的商品ID。
	Referer        string `xml:"referer"`                                           //TODO
	SpbillCreateIp string `xml:"spbill_create_ip" validate:"required,max=16"`       //订单生成的机器IP
	TimeStart      string `xml:"time_start,omitempty" validate:"omitempty,len=14"`  //订单生成时间， 格式为20190101080101，北京时。
	TimeExpire     string `xml:"time_expire,omitempty" validate:"omitempty,len=14"` //订单失效时间， 格式为20190101080101，北京时。
	TotalFee       int    `xml:"total_fee" validate:"min=1"`                        //订单总金额，单位为分
	TradeType      string `xml:"trade_type" validate:"eq=JSAPI|eq=NATIVE|eq=APP"`   //易类型，JSAPI、NATIVE、APP
}

//通信信息
type ReturnInfo struct {
	ReturnCode string `xml:"return_code" validate:"eq=SUCCESS|eq=FAIL"` // 通信标识，非交易标识，SUCCESS/FAIL
	ReturnMsg  string `xml:"return_msg"  validate:"max=128"`            //错误原因:签名失败，参数格式校验错误等
}

//基础返回数据
type Result struct {
	ReturnInfo
	ResultCode string `xml:"result_code" validate:"eq=SUCCESS|eq=FAIL"` // 业务结果，SUCCESS/FAIL
	ErrCode    string `xml:"err_code" validate:"max=32"`                // 业务错误描述
	ErrCodeDes string `xml:"err_code_des" validate:"max=128"`           // 业务错误描述
}

//统一下单返回参数
type UnifiedOrderResult struct {
	Base
	Result
	Attach     string `xml:"attach" validate:"max=127"`                       //附加数据，原样返回
	CodeUrl    string `xml:"code_url"  validate:"max=64"`                     //二维码链接,trade_type 为NATIVE 是有返回，此参数可直接生成二维码展示出来进行扫码支付
	DeviceInfo string `xml:"device_info" validate:"max=32"`                   //微信支付分配的终端设备号
	PrepayId   string `xml:"prepay_id" validate:"max=64"`                     //微信生成的预支付ID，用于后续接口调用中使用
	TradeType  string `xml:"trade_type" validate:"eq=JSAPI|eq=NATIVE|eq=APP"` //易类型，JSAPI、NATIVE、APP
	MwebUrl    string `xml:"mweb_url" validate:"max=64"`                      //TODO
}

//订单查询请求参数
type OrderQueryReq struct {
	Base
	OutTradeNo    string `xml:"out_trade_no" validate:"required,max=32"` //商户系统内部的订单号,可包含字母,确保在商户系统唯一
	TransactionId string `xml:"transaction_id" validate:"max=32"`        //商户系统内部的订单号,如果同时存在优先级：transaction_id> out_trade_no
}

//订单支付结果，由微信服务器主动发送到， 也可以调用订单查询接口主动查询
type OrderResult struct {
	Base
	Result
	BankType    string `xml:"bank_type" validate:"required,max=16"`    //银行类型，采用字符串类型的银行标识
	CashFee     int    `xml:"cash_fee"`                                //TODO
	DeviceInfo  string `xml:"device_info" validate:"max=32"`           //微信支付分配的终端设备号
	FeeType     string `xml:"fee_type" validate:"max=8"`               //货币类型，符合ISO 4217 标准的三位字母代码，默认人民币：CNY
	IsSubscribe string `xml:"is_subscribe" validate:"eq=Y|eq=N"`       //用户是否关注公众账号，Y-关注，N-未关注，仅在公众账号类型支付有效
	Openid      string `xml:"openid" validate:"max=128"`               //用户在商户appid 下的唯一标识，trade_type 为JSAPI时，此参数必传
	OutTradeNo  string `xml:"out_trade_no" validate:"required,max=32"` //商户系统内部的订单号,可包含字母,确保在商户系统唯一

	TimeEnd  string `xml:"time_end" validate:"len=14"` //订单完成时间， 格式为20190101080101，北京时。
	TotalFee int    `xml:"total_fee" validate:"min=1"` //订单总金额，单位为分
	// SUCCESS—支付成功
	// REFUND—转入退款
	// NOTPAY—未支付
	// CLOSED—已关闭
	// REVOKED—已撤销
	// USERPAYING--用户支付中
	// NOPAY--未支付(输入密码或确认支付超时)
	// PAYERROR--支付失败(其他原因，如银行返回失败
	TradeState     string `xml:"trade_state" validate:"eq=SUCCESS|eq=REFUND|eq=NOTPAY|eq=CLOSED|eq=REVOKED|eq=USERPAYING|eq=NOPAY|eq=PAYERROR"`
	TradeStateDesc string `xml:"trade_state_desc"`                                //TODO
	TradeType      string `xml:"trade_type" validate:"eq=JSAPI|eq=NATIVE|eq=APP"` //易类型，JSAPI、NATIVE、APP
	TransactionId  string `xml:"transaction_id" validate:"max=32"`                //商户系统内部的订单号,如果同时存在优先级：transaction_id> out_trade_no
}

//退款请求参数
type RefundReq struct {
	Base
	OpUserId      string `xml:"op_user_id" validate:"required,max=32"`    //操作员帐号, 默认为商户号
	OutTradeNo    string `xml:"out_trade_no" validate:"required,max=32"`  //商户系统内部的订单号,可包含字母,确保在商户系统唯一
	OutRefundNo   string `xml:"out_refund_no" validate:"required,max=32"` //商户系统内部的退款单号，商户系统内部唯一，同一退款单号多次请求只退一笔
	RefundFee     int    `xml:"refund_fee" validate:"min=1"`              //退款总金额，单位为分,可以做部分退款
	TotalFee      int    `xml:"total_fee" validate:"min=1"`               //订单总金额，单位为分
	TransactionId string `xml:"transaction_id" validate:"max=32"`         //商户系统内部的订单号,如果同时存在优先级：transaction_id> out_trade_no
}

//订单退款结果
type RefundResult struct {
	Base
	Result
	CouponRefundFee int    `xml:"coupon_refund_fee" validate:"min=0"`                         //现金券退款金额<= 退款金额，退款金额-现金券退款金额为现金
	DeviceInfo      string `xml:"device_info" validate:"max=32"`                              //微信支付分配的终端设备号
	OutTradeNo      string `xml:"out_trade_no" validate:"required,max=32"`                    //商户系统内部的订单号,可包含字母,确保在商户系统唯一
	OutRefundNo     string `xml:"out_refund_no" validate:"required,max=32"`                   //商户系统内部的退款单号，商户系统内部唯一，同一退款单号多次请求只退一笔
	RefundChannel   string `xml:"refund_channel" validate:"omitempty,eq=ORIGINAL|eq=BALANCE"` //退回到余额
	RefundFee       int    `xml:"refund_fee" validate:"min=1"`                                //退款总金额，单位为分,可以做部分退款
	RefundId        string `xml:"refund_id" validate:"required,max=32"`                       //微信退款单号
	TotalFee        int    `xml:"total_fee" validate:"min=1"`                                 //订单总金额，单位为分
	TransactionId   string `xml:"transaction_id" validate:"required,max=32"`                  //商户系统内部的订单号,如果同时存在优先级：transaction_id> out_trade_no
}

//退款查询
type RefundQueryReq struct {
	Base
	//四个参数必填一个，如果同事存在优先级为：
	//refund_id>out_refund_no>transaction_id>out_trade_no
	OutTradeNo    string `xml:"out_trade_no" validate:"max=32"`   //商户系统内部的订单号,可包含字母,确保在商户系统唯一
	OutRefundNo   string `xml:"out_refund_no" validate:"max=32"`  //商户系统内部的退款单号，商户系统内部唯一，同一退款单号多次请求只退一笔
	RefundId      string `xml:"refund_id" validate:"max=32"`      //微信退款单号
	TransactionId string `xml:"transaction_id" validate:"max=32"` //商户系统内部的订单号,如果同时存在优先级：transaction_id> out_trade_no
}

//退款查询结果
//TODO
type RefundQueryResult struct {
	Base
	Result
	OutTradeNo            string `xml:"out_trade_no"`
	RefundStatus_0        string `xml:"refund_status_0"`
	SettlementRefundFee_0 string `xml:"settlement_refund_fee_0"`
}

//关闭订单请求参数
type CloseOrderReq struct {
	Base
	OutTradeNo string `xml:"out_trade_no" validate:"required,max=32"` //商户系统内部的订单号
}

//统一下单
//URL 地址：https://api.mch.weixin.qq.com/pay/unifiedorder
//统一支付接口，可接受JSAPI/NATIVE/APP 下预支付订单，返回预支付订单号。
//NATIVE 支付返回二维码code_url。
//注意：JSAPI 下单前需要调用登录授权接口(详细调用说明请点击打开链接)获取到用户的Openid。
func (a *Account) UnifiedPay(param *UnifiedOrderReq) (rv *UnifiedOrderResult, err error) {
	rv = &UnifiedOrderResult{}
	if err = a.PostWithoutCert(UnifiedOrderUrl, param, rv); err != nil {
		return nil, err
	}
	//verify result info
	if rv.ResultCode != SUCCESS {
		err = errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", rv.ErrCode, rv.ErrCodeDes)
	}
	return rv, err
}

//微信支付通知处理
//通知URL 是UnifiedPay中提交的参数notify_url，支付完成后，微信会把相关支付和用户信
//息发送到该URL，商户需要接收处理信息。
// 对后台通知交互时，如果微信收到商户的应答不是成功或超时，微信认为通知失败，微
// 信会通过一定的策略（如30 分钟共8 次）定期重新发起通知，尽可能提高通知的成功率，
// 但微信不保证通知最终能成功。
// 由于存在重新发送后台通知的情况，因此同样的通知可能会多次发送给商户系统。商户
// 系统必须能够正确处理重复的通知。
// 推荐的做法是，当收到通知进行处理时，首先检查对应业务数据的状态，判断该通知是
// 否已经处理过，如果没有处理过再进行处理，如果处理过直接返回结果成功。在对业务数据
// 进行状态检查和处理之前，要采用数据锁进行并发控制，以避免函数重入造成的数据混乱。
func (a *Account) NotifyFromWechat(ur *UnifiedOrderResult) (ReturnInfo, error) {
	//verify return info
	if ur.ReturnCode != SUCCESS {
		return ReturnInfo{
			ReturnCode: SUCCESS,
			ReturnMsg:  OK,
		}, errors.Errorf("return_code is not success: %v", ur.ReturnMsg)
	}
	//verify signature
	if !a.Verify(ur) {
		return ReturnInfo{
			ReturnCode: FAIL,
			ReturnMsg:  "failed to verify sign, please retry!",
		}, errors.Errorf("failed to verify sign")
	}
	//verify result info
	if ur.ResultCode != SUCCESS {
		return ReturnInfo{
			ReturnCode: SUCCESS,
			ReturnMsg:  OK,
		}, errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", ur.ErrCode, ur.ErrCodeDes)
	}

	return ReturnInfo{
		ReturnCode: SUCCESS,
		ReturnMsg:  OK,
	}, nil
}

//订单查询
//接口链接：https://api.mch.weixin.qq.com/pay/orderquery
//该接口提供所有微信支付订单的查询，当支付通知处理异常或丢失的情况，商户可以通
//过该接口查询订单支付状态。
func (a *Account) OrderQuery(param *OrderQueryReq) (rv *UnifiedOrderResult, err error) {
	rv = &UnifiedOrderResult{}
	if err = a.PostWithoutCert(OrderQueryUrl, param, rv); err != nil {
		return nil, err
	}
	//verify result info
	if rv.ResultCode != SUCCESS {
		err = errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", rv.ErrCode, rv.ErrCodeDes)
	}
	return rv, err
}

//关闭订单
//接口链接：https://api.mch.weixin.qq.com/pay/closeorder
//当订单支付失败，调用关单接口后用新订单号重新发起支付，如果关单失败，返回已完
//成支付请按正常支付处理。如果出现银行掉单，调用关单成功后，微信后台会主动发起退款。
func (a *Account) CloseOrder(param *CloseOrderReq) (rv *Result, err error) {
	rv = &Result{}
	if err = a.PostWithoutCert(CloseOrderUrl, param, rv); err != nil {
		return nil, err
	}
	//verify result info
	if rv.ResultCode != SUCCESS {
		err = errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", rv.ErrCode, rv.ErrCodeDes)
	}
	return rv, err
}

// 退款
// 接口链接：https://api.mch.weixin.qq.com/secapi/pay/refund
// 请求需要双向证书，需先在商户后台添加操作员。
// 注意：
// 1.交易时间超过1 年的订单无法提交退款；
// 2.支持部分退款，部分退需要设置相同的订单号和不同的out_refund_no。一笔退款失
// 败后重新提交，要采用原来的out_refund_no。总退款金额不能超过用户实际支付金额。
func (a *Account) Refund(param *RefundReq) (rv *RefundResult, err error) {
	rv = &RefundResult{}
	if err = a.PostWithCert(RefundUrl, param, rv); err != nil {
		return nil, err
	}
	//verify result info
	if rv.ResultCode != SUCCESS {
		err = errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", rv.ErrCode, rv.ErrCodeDes)
	}
	return rv, err
}

// 退款查询
// 接口链接：https://api.mch.weixin.qq.com/pay/refundquery
// 提交退款申请后，通过调用该接口查询退款状态。退款有一定延时，用零钱支付的退款
// 20 分钟内到账，银行卡支付的退款3 个工作日后重新查询退款状态。
func (a *Account) RefundQuery(param *RefundQueryReq) (rv *RefundQueryResult, err error) {
	rv = &RefundQueryResult{}
	if err = a.PostWithoutCert(RefundQueryUrl, param, rv); err != nil {
		return nil, err
	}
	//verify result info
	if rv.ResultCode != SUCCESS {
		err = errors.Errorf("result_code is not success: err_code=%v, err_code_des=%v", rv.ErrCode, rv.ErrCodeDes)
	}
	return rv, err
}

type ShortUrlReq struct {
	Base
	//需要转换的URL，签名用原串，传输需URL encode
	LongUrl string `xml:"long_url" validate:"required,max=512"`
}

type ShortUrlResult struct {
	Base
	//SYSTEMERROR—系统错误, URLFORMATERROR—URL 格式错误
	Result
	ShortUrl string `xml:"short_url" validate:"required,max=64"` //转换后的URL
}

// 转换短链接
// 接口链接： https://api.mch.weixin.qq.com/tools/shorturl
// 该接口主要用于Native 支付模式一中的二维码链接转成短链接
// (weixin://wxpay/s/XXXXXX)，减小二维码数据量，提升扫描速度。
func (a *Account) ShortUrl(param *ShortUrlReq) (rv *ShortUrlResult, err error) {
	rv = &ShortUrlResult{}
	//fill base parameters
	a.FillBaseParams(param)

	//需要转换的URL，签名用原串，传输需URL encode
	param.LongUrl = url.QueryEscape(param.LongUrl)

	//validate params
	if err := bind.Validate(param); err != nil {
		return nil, errors.Wrap(err, "input parameters")
	}

	//create request
	req, err := client.NewXmlRequest("POST", ShortUrl, param)
	if err != nil {
		return nil, errors.Wrap(err, "NewXmlRequest")
	}
	req.Header.Set("Accept", "application/xml")

	//post and get restult
	status, err := client.Default.Exec(req, rv)
	if err != nil {
		return nil, errors.Wrapf(err, "https post error, status=%v", status)
	}

	//validate result
	if err := bind.Validate(rv); err != nil {
		return nil, errors.Wrap(err, "output parameters")
	}
	//verify return info
	if rv.ReturnCode != SUCCESS {
		return nil, errors.Errorf("return_code is not success: %v", rv.ReturnMsg)
	}
	//verify signature
	if !a.Verify(rv) {
		return nil, errors.Errorf("failed to verify sign")
	}
	//verify result info
	if rv.ResultCode != SUCCESS {
		//SYSTEMERROR—系统错误, URLFORMATERROR—URL 格式错误
		err = errors.Errorf("result_code is not success: err_code=%v", rv.ErrCode)
	}

	return rv, err
}
