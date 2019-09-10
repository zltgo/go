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

func TestTokenParser(t *testing.T) {
	key := []byte("1111111111111111")
	Convey("create and parse token without maxAge and encpyt", t, func() {
		tp := NewParser(0, key, nil)
		mp := map[string]interface{}{"foo": "bar"}
		TimeNow = timeFunc(1000, 1000)
		Convey("create a token", func() {
			tk, err := tp.CreateToken(mp)
			So(err, ShouldBeNil)
			So(tk, ShouldEqual, "eyJUb2tlblZhbHVlcyI6eyJmb28iOiJiYXIifSwiSXNzdWVkQXQiOjEwMDB9VgdhPM3G67TFV6P-amk4BofAyDE=")
		})
		Convey("parse a token with map", func() {
			mp2 := make(map[string]interface{})
			err := tp.ParseToken("eyJUb2tlblZhbHVlcyI6eyJmb28iOiJiYXIifSwiSXNzdWVkQXQiOjEwMDB9VgdhPM3G67TFV6P-amk4BofAyDE=", &mp2)
			So(err, ShouldBeNil)
			So(len(mp2), ShouldEqual, 1)
			So(mp2["foo"], ShouldEqual, "bar")
		})
		Convey("parse a token with struct", func() {
			struc := struct {
				Foo string
			}{}
			err := tp.ParseToken("eyJUb2tlblZhbHVlcyI6eyJmb28iOiJiYXIifSwiSXNzdWVkQXQiOjEwMDB9VgdhPM3G67TFV6P-amk4BofAyDE=", &struc)
			So(err, ShouldBeNil)
			So(struc.Foo, ShouldEqual, "bar")
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
		})

		Convey("parse a token on failure", func() {
			TimeNow = timeFunc(999, 999)
			mp := make(map[string]interface{})
			err := tp.ParseToken(tk, &mp)
			So(err, ShouldEqual, ErrIssued)

			TimeNow = timeFunc(1002, 1001)
			err = tp.ParseToken(tk, &mp)
			So(err, ShouldEqual, ErrExpired)
		})
	})
}
