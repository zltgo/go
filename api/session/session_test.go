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
		t.Log(cks[0])
		t.Log(cks[1])
		So(len(cks), ShouldEqual, 2)
	})

	Convey("get session in lruMemCache", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		c, err := cs.GetSession(r)
		So(err, ShouldBeNil)
		So(c.ValueOf("ping").String(), ShouldEqual, "pang")
	})

	Convey("get session without cookie token", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		cp.Set("any id", "delete cookie cache")
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
		So(c.ValueOf("ping").String(), ShouldEqual, "pang")
	})

	Convey("remove id, should return ErrIdMismatch", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])
		r.AddCookie(cks[1])

		c, err := cs.GetSession(r)
		c.Id = "" //RemoveAll is better

		w := httptest.NewRecorder()
		err = cs.SaveSession(w, c)
		So(err, ShouldBeNil)

		cookies := w.Result().Cookies()
		So(len(cookies), ShouldEqual, 1)
		So(cookies[0].MaxAge, ShouldEqual, -1)

		c, err = cs.GetSession(r)
		So(err, ShouldEqual, ErrIdMismatch)
	})

	Convey("clear session, should return ErrIdMismatch", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])
		r.AddCookie(cks[1])

		c, err := cs.GetSession(r)
		c.RemoveAll() //***

		w := httptest.NewRecorder()
		err = cs.SaveSession(w, c)
		So(err, ShouldBeNil)

		cookies := w.Result().Cookies()
		So(len(cookies), ShouldEqual, 2)
		So(cookies[0].MaxAge, ShouldEqual, -1)
		So(cookies[1].MaxAge, ShouldEqual, -1)

		c, err = cs.GetSession(r)
		So(err, ShouldEqual, ErrIdMismatch)
		So(c.Len(), ShouldEqual, 1)
	})
}

func TestCookieRatelimit(t *testing.T) {
	lmc := cache.NewLruMemCache(1024)

	Convey("shoulld be rate limited", t, func() {
		cs := NewCookieProvider(CookieOpts{
			RateSec: []int{1000, 100}, //1000 times per 100 sec
		}, lmc)

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
			RateSec: []int{c * n, 100},
		}, lmc)

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
	lmc := cache.NewLruMemCache(1024)
	cs := NewCookieProvider(CookieOpts{}, lmc)

	r, _ := http.NewRequest("GET", "/test", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if s, err := cs.GetSession(r); s == nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_CookieStore(b *testing.B) {

	lmc := cache.NewLruMemCache(1024)
	cs := NewCookieProvider(CookieOpts{}, lmc)

	r, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	c, _ := cs.GetSession(r)
	c.Set("1", 1)
	if err := cs.SaveSession(w, c); err != nil {
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
	c, _ = cs.GetSession(r)
	if c == nil || c.ValueOf("1").MustInt() != 1 {
		b.Error("unknown error")
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := cs.GetSession(r); err != nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_NoStore(b *testing.B) {
	lmc := cache.NewLruMemCache(1024)
	cs := NewProvider(nil, lmc)

	r, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	c, _ := cs.GetSession(r)
	c.Set("1", 1)
	if err := cs.SaveSession(w, c); err != nil {
		b.Error(err)
		return
	}
	cks := w.Result().Cookies()
	if len(cks) != 1 {
		b.Error("no cookie")
		return
	}
	r.AddCookie(cks[0])
	c, _ = cs.GetSession(r)
	if c == nil || c.ValueOf("1").MustInt() != 1 {
		b.Error("unknown error")
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := cs.GetSession(r); err != nil {
			b.Error(err)
			return
		}
	}
}
