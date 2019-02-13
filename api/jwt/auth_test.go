package jwt

import (
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zltgo/api"
	"github.com/zltgo/api/client"
)

type LoginForm struct {
	Name     string `validate:"alphanum,min=5,max=32"`
	Password string `validate:"alphanum,min=5,max=32"`
}

func login(lf LoginForm) (int, string) {
	if lf.Name == "admin" && lf.Password == "admin" {
		return 200, "000001"
	} else {
		return 401, "name or password error"
	}
}

func MyUid(uid UID) string {
	return string(uid)
}

func apiServer() *api.Server {
	api.SetMode(api.TestMode)
	r := api.New(api.Recovery())
	tp1 := NewParser(10, []byte("1111111111111111"), nil)
	tp2 := NewParser(100, []byte("1111111111111111"), nil)

	auth := NewAuth(tp1, tp2, nil)

	r.POST("/login", auth.LoginHandler(login))
	r.GET("/refresh_token", auth.RefreshHandler)
	r.GET("/auth/myuid", auth.AuthHandler, api.H(MyUid))

	return r
}

var (
	router       = apiServer()
	accessToken  = "eyJfYWdoIjoiY2JmMjljZTQ4NDIyMjMyNSIsIl9jdCI6MTAwMCwiX2dydCI6ImFjY2VzcyIsIl91aWQiOiIwMDAwMDEifU7ipBKGlfRjdkmBynUndABApXZp"
	refreshToken = "eyJfYWdoIjoiY2JmMjljZTQ4NDIyMjMyNSIsIl9jdCI6MTAwMCwiX2dydCI6InJlZnJlc2giLCJfdWlkIjoiMDAwMDAxIn2savtHeGDugt20HwriMXuInEHGOg=="
)

func TestLoginFunc(t *testing.T) {
	Convey("Create LoginHandler  successfully", t, func() {
		auth := NewAuth(nil, nil, nil)
		So(func() { auth.LoginHandler(login) }, ShouldNotPanic)
	})

	Convey("Create LoginHandler with bad return value", t, func() {
		f := func(lf LoginForm) (LoginForm, error) { return lf, nil }
		auth := NewAuth(nil, nil, nil)
		So(func() { auth.LoginHandler(f) }, ShouldPanicWith, "loginFunc must return (int, string)")
	})

}

func TestLoginHandler(t *testing.T) {
	TimeNow = timeFunc(1000, 1000)
	Convey("test client login and get access-token", t, func() {
		Convey("Login successfully", func() {
			form := url.Values{}
			form.Set("Name", "admin")
			form.Set("Password", "admin")
			r, err := client.NewRequest("POST", "/login", form)
			So(err, ShouldBeNil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			token := AuthToken{}
			err = json.NewDecoder(w.Body).Decode(&token)
			So(err, ShouldBeNil)
			So(token.AccessToken, ShouldEqual, accessToken)
			So(token.MaxAge, ShouldEqual, 10)
			So(token.RefreshToken, ShouldEqual, refreshToken)
		})

		Convey("Login with name or password error", func() {
			form := url.Values{}
			form.Set("Name", "admin")
			form.Set("Password", "pingpong")
			r, err := client.NewRequest("POST", "/login", form)
			So(err, ShouldBeNil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})

		Convey("Login with bad request", func() {
			form := url.Values{}
			form.Set("Name", "a")
			form.Set("Password", "b")
			r, err := client.NewRequest("POST", "/login", form)
			So(err, ShouldBeNil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 400)
		})
	})
}

func TestAuthHandler(t *testing.T) {
	TimeNow = timeFunc(1000, 1000)
	Convey("Test AuthHandler", t, func() {
		Convey("Pass  AuthHandler and get uid successfully\n", func() {
			r, _ := client.NewRequest("GET", "/auth/myuid", nil)
			r.Header.Set("ACCESS-TOKEN", accessToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)
			So(w.Body.String(), ShouldEqual, "000001")
		})

		Convey("Pass  AuthMware with  ErrNoTokenInRequest\n", func() {
			r, _ := client.NewRequest("GET", "/auth/myuid", nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})

		Convey("Pass  AuthMware with  ErrMacInvalid\n", func() {
			r, _ := client.NewRequest("GET", "/auth/myuid", nil)

			r.Header.Set("ACCESS-TOKEN", "INVALIDTOKEN")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})

		Convey("Pass  AuthMware with wrong agent\n", func() {
			r, _ := client.NewRequest("GET", "/auth/myuid", nil)
			r.Header.Set("User-Agent", "wrong agent")

			r.Header.Set("ACCESS-TOKEN", accessToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})

		Convey("Pass  AuthMware with wrong grant type\n", func() {
			r, _ := client.NewRequest("GET", "/auth/myuid", nil)
			r.Header.Set("ACCESS-TOKEN", refreshToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})
	})
}

func TestRefreshHandler(t *testing.T) {
	TimeNow = timeFunc(1000, 1000)
	Convey("Test AuthHandler", t, func() {
		Convey("refresh token successfully\n", func() {
			r, _ := client.NewRequest("GET", "/refresh_token", nil)
			r.Header.Set("ACCESS-TOKEN", refreshToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 200)

			token := AuthToken{}
			err := json.NewDecoder(w.Body).Decode(&token)
			So(err, ShouldBeNil)
			So(token.AccessToken, ShouldEqual, accessToken)
			So(token.RefreshToken, ShouldEqual, refreshToken)
			So(token.MaxAge, ShouldEqual, 10)
		})

		Convey("refresh token with  ErrNoTokenInRequest\n", func() {
			r, _ := client.NewRequest("GET", "/refresh_token", nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})

		Convey("refresh token with  ErrMacInvalid\n", func() {
			r, _ := client.NewRequest("GET", "/refresh_token", nil)

			r.Header.Set("ACCESS-TOKEN", "INVALIDTOKEN")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})

		Convey("refresh token with wrong agent\n", func() {
			r, _ := client.NewRequest("GET", "/refresh_token", nil)
			r.Header.Set("User-Agent", "wrong agent")

			r.Header.Set("ACCESS-TOKEN", refreshToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})

		Convey("refresh token with wrong grant type\n", func() {
			r, _ := client.NewRequest("GET", "/refresh_token", nil)
			r.Header.Set("ACCESS-TOKEN", accessToken)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			So(w.Code, ShouldEqual, 401)
		})
	})
}
