package wepay

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type Base struct {
	Appid      string
	AuthCode   string
	MchId      string
	DeviceInfo string
}

type UnifyOrderReq struct {
	Base
	Sign           string
	Body           string
	SpbillCreateIp string
	TotalFee       int
	OutTradeNo     string
	NonceStr       string
}

func TestSign(t *testing.T) {
	base := Base{
		Appid:      "wxd930ea5d5a258f4f",
		AuthCode:   "123456",
		MchId:      "1900000109",
		DeviceInfo: "123",
	}
	req := UnifyOrderReq{
		Base:           base,
		Sign:           "nothing",
		Body:           "test",
		SpbillCreateIp: "127.0.0.1",
		TotalFee:       1,
		OutTradeNo:     "1400755861",
		NonceStr:       "960f228109051b9969f76c82bde183ac",
	}

	Convey("should sign correctly with MD5", t, func() {
		kvString := `appid=wxd930ea5d5a258f4f&auth_code=123456&body=test&device_info=123&mch_id=1900000109&nonce_str=960f228109051b9969f76c82bde183ac&out_trade_no=1400755861&spbill_create_ip=127.0.0.1&total_fee=1&key=8934e7d15453e97507ef794cf7b0519d`
		sb := md5.Sum([]byte(kvString))
		signValue := strings.ToUpper(hex.EncodeToString(sb[:]))
		So(signValue, ShouldEqual, "729A68AC3DE268DBD9ADE442382E7B24")

		s := NewSigner(SignOpts{ApiKey: "8934e7d15453e97507ef794cf7b0519d"})
		sv := s.Sign(&req)
		So(sv, ShouldEqual, signValue)
	})

	Convey("should sign correctly with hmac-sha256", t, func() {
		s := NewSigner(SignOpts{
			ApiKey:   "8934e7d15453e97507ef794cf7b0519d",
			HashKey:  "8934e7d15453e97507ef794cf7b0519d",
			SignType: "HMAC-SHA256",
		})
		sv := s.Sign(&req)
		So(sv, ShouldEqual, "F0BC24E76CC164C44AD833618BE7964B3544CE4E5EA06F069A263EA9C433CC50")
	})
}
