package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zltgo/api/cache"
	"gopkg.in/mgo.v2"
)

func TestMongoStore(t *testing.T) {
	cp := cache.NewLruMemCache(1)

	mpd, err := NewMongoProvider(MongoOpts{
		Url: "mongodb://zyx:112358@localhost:27017/tokens",
		C:   "t",
	}, cp)

	if err != nil {
		t.Error(err)
		return
	}

	cks := make([]*http.Cookie, 2)
	Convey("get session without session id", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		c, err := mpd.GetSession(r)
		So(err, ShouldEqual, http.ErrNoCookie)

		c.Set("ping", "pang")
		err = mpd.SaveSession(w, c)
		So(err, ShouldBeNil)

		cks = w.Result().Cookies()
		So(len(cks), ShouldEqual, 1)
		t.Log(cks[0])
		t.Log(cp.Stats())
	})

	Convey("get session in lru mem cache", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		c, err := mpd.GetSession(r)
		So(err, ShouldBeNil)
		So(c.Get("ping"), ShouldEqual, "pang")
		t.Log(cp.Stats())
	})

	Convey("get session in mongodb", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		cp.Get("delete cookie cache")
		c, err := mpd.GetSession(r)
		So(err, ShouldBeNil)
		So(c.Get("ping"), ShouldEqual, "pang")
		t.Log(cp.Stats())
	})

	Convey("remove session, should return ErrIdMismatch", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		c, err := mpd.GetSession(r)
		c.Assign(nil)

		w := httptest.NewRecorder()
		err = mpd.SaveSession(w, c)
		So(err, ShouldBeNil)

		cks := w.Result().Cookies()
		So(len(cks), ShouldEqual, 1)
		So(cks[0].MaxAge, ShouldEqual, -1)

		c, err = mpd.GetSession(r)
		So(err, ShouldEqual, ErrIdMismatch)
	})

	Convey("clear session, should return mgo.ErrNotFound", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		c, err := mpd.GetSession(r)
		c.Clear()

		w := httptest.NewRecorder()
		err = mpd.SaveSession(w, c)
		So(err, ShouldBeNil)

		cks := w.Result().Cookies()
		So(len(cks), ShouldEqual, 1)
		So(cks[0].MaxAge, ShouldEqual, -1)

		c, err = mpd.GetSession(r)
		So(err, ShouldEqual, mgo.ErrNotFound)
		So(c.Items(), ShouldEqual, 1)
	})
}

func Benchmark_UpdateMongoSession(b *testing.B) {
	mpd, err := NewMongoProvider(MongoOpts{
		Url: "mongodb://zyx:112358@localhost:27017/tokens",
		C:   "t",
	}, cache.DefaultLruMemCache)

	if err != nil {
		b.Error(err)
		return
	}

	r, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	c, _ := mpd.GetSession(r)
	c.Set("1", 1)
	if err := mpd.SaveSession(w, c); err != nil {
		b.Error(err)
		return
	}
	cks := w.Result().Cookies()
	if len(cks) != 1 {
		b.Error("no cookie")
		return
	}

	r.AddCookie(cks[0])
	c, _ = mpd.GetSession(r)
	if c == nil || c.GetInt("1") != 1 {
		b.Error("unknown error")
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		se, err := mpd.GetSession(r)
		if err != nil {
			b.Error(err)
			return
		}
		se.AddInt("1", 1)
		err = mpd.SaveSession(w, se)
		if err != nil {
			b.Error(err)
			return
		}
	}

	se, _ := mpd.GetSession(r)
	b.Log(se.GetInt("1"))
}

func Benchmark_UpdateCookieSession(b *testing.B) {
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
		se, err := DefaultCookiePd.GetSession(r)
		if err != nil {
			b.Error(err)
			return
		}
		se.AddInt("1", 1)
		err = DefaultCookiePd.SaveSession(w, se)
		if err != nil {
			b.Error(err)
			return
		}
	}

	se, _ := DefaultCookiePd.GetSession(r)
	b.Log(se.GetInt("1"))
}
