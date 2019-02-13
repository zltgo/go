package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zltgo/reflectx/values"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/zltgo/api/cache"
	"gopkg.in/mgo.v2"
)

func TestMongoStore(t *testing.T) {
	cp := cache.NewLruMemCache(1)

	mpd, err := NewMongoProvider(MongoOpts{
		//Url: "mongodb://zyx:112358@localhost:27017/test",
		Url: "mongodb://localhost:27017/test",
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
		So(c.ValueOf("ping").String(), ShouldEqual, "pang")
		t.Log(cp.Stats())
	})

	Convey("get session in mongodb", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		cp.Set("any id", "delete cookie cache")
		c, err := mpd.GetSession(r)
		So(err, ShouldBeNil)
		So(c.ValueOf("ping").String(), ShouldEqual, "pang")
		t.Log(cp.Stats())
	})

	Convey("remove ID, should return ErrIdMismatch", t, func() {
		r, _ := http.NewRequest("GET", "/test", nil)
		r.AddCookie(cks[0])

		c, err := mpd.GetSession(r)
		c.Id = "" //RemoveAll is better

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
		c.RemoveAll()

		w := httptest.NewRecorder()
		err = mpd.SaveSession(w, c)
		So(err, ShouldBeNil)

		cks := w.Result().Cookies()
		So(len(cks), ShouldEqual, 1)
		So(cks[0].MaxAge, ShouldEqual, -1)

		c, err = mpd.GetSession(r)
		So(err, ShouldEqual, ErrIdMismatch)
		So(c.Len(), ShouldEqual, 1)

		//delete from lrumemcache
		cp.Set("any id", "delete cookie cache")
		c, err = mpd.GetSession(r)
		So(err, ShouldEqual, mgo.ErrNotFound)
		So(c.Len(), ShouldEqual, 1)
	})
}

func Benchmark_UpdateMongoSession(b *testing.B) {
	lmc := cache.NewLruMemCache(1024)

	mpd, err := NewMongoProvider(MongoOpts{
		Url: "mongodb://zyx:112358@localhost:27017/test",
		C:   "t",
	}, lmc)

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
	if c == nil || c.ValueOf("1").MustInt() != 1 {
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
		values.AddInt(se, "1", 1)
		err = mpd.SaveSession(w, se)
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func Benchmark_UpdateCookieSession(b *testing.B) {
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
		se, err := cs.GetSession(r)
		if err != nil {
			b.Error(err)
			return
		}
		values.AddInt(se, "1", 1)
		err = cs.SaveSession(w, se)
		if err != nil {
			b.Error(err)
			return
		}
	}
}
