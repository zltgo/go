package jwt

import (
	"net/http"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type sub struct {
	Foo []string
}

type usr struct {
	Id   int
	Name string
	Sub  sub
}

func TestAuth(t *testing.T) {
	TimeNow = timeFunc(1000, 1000)
	au := NewAuth(AuthOpts{
		HashKey:  "1111111111111111",
		BlockKey: "1111111111111111",
	})

	Convey("Test Auth", t, func() {
		Convey("encode and decode token successfully\n", func() {

			r, _ := http.NewRequest("GET", "/auth/myuid", nil)
			r.Header.Set("User-Agent", "agent")

			input := usr{1, "zyx", sub{[]string{"foo", "bar"}}}
			tk, err := au.NewAuthToken(r, input)
			So(err, ShouldBeNil)

			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)

			output := usr{}
			err = au.GetAccessInfo(r, &output)
			So(err, ShouldBeNil)
			So(output, ShouldResemble, input)
		})

		Convey("with ErrNoTokenInRequest\n", func() {
			r, _ := http.NewRequest("GET", "/auth/myuid", nil)
			r.Header.Set("User-Agent", "agent")
			output := usr{}
			err := au.GetAccessInfo(r, &output)
			So(err, ShouldEqual, ErrNoToken)

		})

		Convey("with  ErrMacInvalid\n", func() {
			r, _ := http.NewRequest("GET", "/auth/myuid", nil)

			r.Header.Set("ACCESS-TOKEN", "INVALIDTOKEN")
			output := usr{}
			err := au.GetAccessInfo(r, &output)
			So(err, ShouldEqual, ErrMacInvalid)
		})

		Convey("with wrong agent\n", func() {
			r, _ := http.NewRequest("GET", "/auth/myuid", nil)
			r.Header.Set("User-Agent", "agent")
			tk, err := au.NewAuthToken(r, nil)
			So(err, ShouldBeNil)

			r.Header.Set("ACCESS-TOKEN", tk.AccessToken)
			r.Header.Set("User-Agent", "wrong agent")
			output := usr{}
			err = au.GetAccessInfo(r, &output)
			So(err.Error(), ShouldEqual, "jwt: user agent mismatched: wrong agent")
		})

		Convey("with wrong grant type\n", func() {
			r, _ := http.NewRequest("GET", "/auth/myuid", nil)
			r.Header.Set("User-Agent", "agent")
			tk, err := au.NewAuthToken(r, nil)
			So(err, ShouldBeNil)

			r.Header.Set("REFRESH-TOKEN", tk.AccessToken)
			output := usr{}
			err = au.GetRefreshInfo(r, &output)
			So(err.Error(), ShouldEqual, "jwt: grant type mismatched: expected refresh, got access")
		})
	})
}
