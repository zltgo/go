package jwt

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zltgo/reflectx/values"
)

func timeFunc(sec int64, nsec int64) func() time.Time {
	return func() time.Time {
		return time.Unix(sec, nsec)
	}
}

func TestTokenProvider(t *testing.T) {
	key := []byte("1111111111111111")
	Convey("create and parse token without maxAge and encpyt", t, func() {
		tp := NewParser(0, key, nil)
		mp := map[string]interface{}{"foo": "bar"}

		Convey("create a token", func() {
			tk, err := tp.CreateToken(mp)
			So(err, ShouldBeNil)
			So(tk, ShouldEqual, "eyJmb28iOiJiYXIifVCgi6qzwZyo7IRYlTOdafty7b6y")
		})
		Convey("parse a token", func() {
			mp2, err := tp.ParseToken("eyJmb28iOiJiYXIifVCgi6qzwZyo7IRYlTOdafty7b6y")
			So(err, ShouldBeNil)
			So(mp2["foo"], ShouldEqual, "bar")
			So(len(mp2), ShouldEqual, 1)
		})
	})

	var tk string
	var err error
	Convey("create and parse token with maxAge and encpyt", t, func() {
		tp := NewParser(1, key, key)
		vs := values.JsonMap{"foo": "bar"}
		Convey("create a token", func() {
			TimeNow = timeFunc(1000, 1000)
			tk, err = tp.CreateToken(vs)
			So(err, ShouldBeNil)
			timestamp := vs.ValueOf(CreateTimeKey).Int64()
			So(timestamp, ShouldEqual, time.Unix(1000, 1000).Unix())
		})
		Convey("parse a token on success", func() {
			mp2, err := tp.ParseToken(tk)
			So(err, ShouldBeNil)
			So(mp2["foo"], ShouldEqual, "bar")

			timestamp := values.JsonMap(mp2).ValueOf(CreateTimeKey).Int64()
			So(timestamp, ShouldEqual, time.Unix(1000, 1000).Unix())
			So(len(mp2), ShouldEqual, 2)
		})
		Convey("parse a token on failure", func() {
			TimeNow = timeFunc(999, 999)
			_, err := tp.ParseToken(tk)
			So(err, ShouldEqual, ErrTimestamp)

			TimeNow = timeFunc(1002, 1001)
			_, err = tp.ParseToken(tk)
			So(err, ShouldEqual, ErrExpired)
		})
	})
}
