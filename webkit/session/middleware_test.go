package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/zltgo/api"
)

func ginServer() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	DefaultCookiePd.SetUrlRatelimit(map[string][]int{
		"POST:/login": []int{1000, 10},
		"GET:/login":  []int{3000, 10},
		"ANY:/login":  []int{1500, 10},
		"GET:/auth":   []int{1000, 10},
		"ANY:/":       []int{3000, 10},
		"GET:/hello":  []int{3000, 10},
	})
	rl := DefaultCookiePd.SessionMware
	r.POST("/login", api.Gin(rl, func() string {
		return "post_login"
	}))
	r.GET("/login", api.Gin(rl, func() string {
		return "get_login"
	}))
	r.GET("/auth/:name", api.Gin(rl, func(ctx *api.Context) {
		ctx.Render(ctx.Params.Get("name"))
	}))
	r.GET("/hello", api.Gin(rl, func() string {
		return "hello"
	}))
	r.GET("/world", api.Gin(rl, func() string {
		return "world"
	}))

	return r
}

func TestUrlRatelimiter(t *testing.T) {
	cks := make([]*http.Cookie, 2)
	g := ginServer()
	Convey("should accurately rate-limit POST:/login", t, func() {
		r, _ := http.NewRequest("POST", "/login", nil)
		r.RemoteAddr = "ip:9527"
		w := httptest.NewRecorder()
		g.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)
		So(w.Body.String(), ShouldEqual, "post_login")

		cks = w.Result().Cookies()
		So(len(cks), ShouldEqual, 2)

		r.AddCookie(cks[0])

		var cnt int
		for cnt = 1; cnt < 2000; cnt++ {
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			if w.Code != 200 {
				So(w.Code, ShouldEqual, http.StatusTooManyRequests)
				break
			}
		}

		So(cnt, ShouldEqual, 1000)
	})

	Convey("should accurately rate-limit ANY:/login", t, func() {
		r, _ := http.NewRequest("GET", "/login", nil)
		r.RemoteAddr = "ip:9527"
		r.AddCookie(cks[0])

		var cnt int
		for cnt = 1; cnt < 2000; cnt++ {
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			if w.Code != 200 {
				So(w.Code, ShouldEqual, http.StatusTooManyRequests)
				break
			}
		}
		So(cnt, ShouldEqual, 500)
	})

	Convey("should accurately rate-limit GET:/auth", t, func() {
		r, _ := http.NewRequest("GET", "/auth/zyx", nil)
		r.RemoteAddr = "ip:9527"
		r.AddCookie(cks[0])

		var cnt int
		for cnt = 0; cnt < 2000; cnt++ {
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			if w.Code != 200 {
				So(w.Code, ShouldEqual, http.StatusTooManyRequests)
				break
			}
		}
		So(cnt, ShouldEqual, 1000)
	})

	Convey("should accurately rate-limit ANY:/", t, func() {
		r, _ := http.NewRequest("GET", "/hello", nil)
		r.RemoteAddr = "ip:9527"
		r.AddCookie(cks[0])

		var cnt int
		for cnt = 0; cnt < 2000; cnt++ {
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			if w.Code != 200 {
				So(w.Code, ShouldEqual, http.StatusTooManyRequests)
				break
			}
		}
		So(cnt, ShouldEqual, 498)
	})

	Convey("should accurately rate-limit url which not cfg", t, func() {
		r, _ := http.NewRequest("GET", "/world", nil)
		r.RemoteAddr = "ip:9527"
		r.AddCookie(cks[0])

		var cnt int
		for cnt = 0; cnt < 2000; cnt++ {
			w := httptest.NewRecorder()
			g.ServeHTTP(w, r)
			if w.Code != 200 {
				So(w.Code, ShouldEqual, http.StatusTooManyRequests)
				break
			}
		}
		So(cnt, ShouldEqual, 0)
	})
}
