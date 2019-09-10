package session

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/jwt"
)

func TestCookieStore(t *testing.T) {
	cp := cache.NewLruMemCache(1)
	cs := NewCookieProvider(CookieOpts{}, cp)
	cks := make([]*http.Cookie, 2)
	Convey("get session without session id", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		c, err := cs.GetSession(r)
		So(err, ShouldEqual, http.ErrNoCookie)

		c.Set("ping", "pang")
		err = cs.SaveSession(w, c)
		So(err, ShouldBeNil)

		cks = w.Result().Cookies()
		So(len(cks), ShouldEqual, 2)
		t.Log(cks[0])
		t.Log(cks[1])
	})

	Convey("get session in lru mem cache", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		c, err := cs.GetSession(r)
		So(err, ShouldBeNil)
		So(c.Get("ping"), ShouldEqual, "pang")
	})

	Convey("get session without cookie token", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		cp.Get("delete cookie cache")
		_, err := cs.GetSession(r)
		So(err, ShouldEqual, http.ErrNoCookie)
	})

	Convey("get session with ErrMacInvalid error", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])
		r.AddCookie(cks[1])

		tmp := NewCookieProvider(CookieOpts{}, cp)
		_, err := tmp.GetSession(r)
		So(err, ShouldEqual, jwt.ErrMacInvalid)
	})

	Convey("get session with ErrIdMismatch error", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)

		c, _ := cs.GetSession(r)
		c.Set("some value", 1)
		w := httptest.NewRecorder()
		cs.SaveSession(w, c)

		r.AddCookie(cks[0])
		r.AddCookie(w.Result().Cookies()[1])

		_, err := cs.GetSession(r)
		So(err, ShouldEqual, ErrIdMismatch)
	})

	Convey("get session from cookie successfully", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])
		r.AddCookie(cks[1])

		c, err := cs.GetSession(r)
		So(err, ShouldBeNil)
		So(c.Get("ping"), ShouldEqual, "pang")
	})

	Convey("remove session, should return ErrIdMismatch", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])
		r.AddCookie(cks[1])

		c, err := cs.GetSession(r)
		c.Assign(nil)

		w := httptest.NewRecorder()
		err = cs.SaveSession(w, c)
		So(err, ShouldBeNil)

		cks := w.Result().Cookies()
		So(len(cks), ShouldEqual, 2)
		So(cks[0].MaxAge, ShouldEqual, -1)
		So(cks[1].MaxAge, ShouldEqual, -1)

		c, err = cs.GetSession(r)
		So(err, ShouldEqual, ErrIdMismatch)
	})

	Convey("clear session, should return ErrIdMismatch", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])
		r.AddCookie(cks[1])

		c, err := cs.GetSession(r)
		c.Clear()

		w := httptest.NewRecorder()
		err = cs.SaveSession(w, c)
		So(err, ShouldBeNil)

		cks := w.Result().Cookies()
		So(len(cks), ShouldEqual, 2)
		So(cks[0].MaxAge, ShouldEqual, -1)
		So(cks[1].MaxAge, ShouldEqual, -1)

		c, err = cs.GetSession(r)
		So(err, ShouldEqual, ErrIdMismatch)
		So(c.Items(), ShouldEqual, 1)
	})
}

func TestCookieRatelimit(t *testing.T) {
	Convey("shoulld be rate limited", t, func() {
		cs := NewCookieProvider(CookieOpts{
			RateLimit: []int{1000, 100},
		}, cache.DefaultLruMemCache)

		r, _ := http.NewRequest("GET", "/test", nil)
		r.RemoteAddr = "ip1:9527"
		cnt := 0
		for ; cnt < 20000; cnt++ {
			if se, _ := cs.GetSession(r); se == nil {
				break
			}
		}

		So(cnt, ShouldEqual, 1000)
	})

	Convey("should be thread-safe", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.RemoteAddr = "ip2:9527"
		c := 1000
		n := 1000
		wg := sync.WaitGroup{}
		cs := NewCookieProvider(CookieOpts{
			RateLimit: []int{c * n, 100},
		}, cache.DefaultLruMemCache)

		for i := 0; i < c; i++ {
			wg.Add(1)

			go func(thread int) {
				defer wg.Done()
				for j := 0; j < n; j++ {
					if se, err := cs.GetSession(r); se == nil {
						t.Error(fmt.Sprintf("thread %d, cycl %d, %v", thread, j, err))
						break
					}
				}
			}(i)
		}
		wg.Wait()

		se, err := cs.GetSession(r)
		So(se, ShouldBeNil)
		So(err, ShouldEqual, ErrOverrun)
	})
}

func Benchmark_NewSession(b *testing.B) {
	r, _ := http.NewRequest("GET", "/test", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if s, err := DefaultCookiePd.GetSession(r); s == nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_GetSession(b *testing.B) {
	r, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	c, _ := DefaultCookiePd.GetSession(r)
	c.Set("1", 1)
	if err := DefaultCookiePd.SaveSession(w, c); err != nil {
		b.Error(err)
		return
	}
	cks := w.Result().Cookies()
	if len(cks) != 2 {
		b.Error("no cookie")
		return
	}
	r.AddCookie(cks[0])
	r.AddCookie(cks[1])
	c, _ = DefaultCookiePd.GetSession(r)
	if c == nil || c.GetInt("1") != 1 {
		b.Error("unknown error")
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DefaultCookiePd.GetSession(r); err != nil {
			b.Error(err)
			return
		}
	}
}
