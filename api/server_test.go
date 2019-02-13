package api

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-martini/martini"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/zltgo/api/bind"
	"github.com/zltgo/api/render"
)

func testRequest(t *testing.T, url string) {
	resp, err := http.Get(url)
	assert.NoError(t, err)
	defer resp.Body.Close()

	body, ioerr := ioutil.ReadAll(resp.Body)
	assert.NoError(t, ioerr)
	assert.Equal(t, "it worked", string(body), "resp body should match")
	assert.Equal(t, "200 OK", resp.Status, "should get a 200")
}

func TestRunEmpty(t *testing.T) {
	os.Setenv("PORT", "")
	router := New()
	go func() {
		router.GET("/example", func(c *Context) { c.Reply(http.StatusOK, "it worked") })
		assert.NoError(t, router.Run(":8081"))
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(10 * time.Millisecond)

	assert.Error(t, router.Run(":8081"))
	testRequest(t, "http://localhost:8081/example")
}

func TestRunWithPort(t *testing.T) {
	router := New()
	go func() {
		router.GET("/example", func(c *Context) { c.Reply(http.StatusOK, "it worked") })
		assert.NoError(t, router.Run(":5150"))
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	assert.Error(t, router.Run(":5150"))
	testRequest(t, "http://localhost:5150/example")
}

func TestUnixSocket(t *testing.T) {
	router := New()

	go func() {
		router.GET("/example", func(c *Context) { c.Reply(http.StatusOK, "it worked") })
		assert.NoError(t, router.RunUnix("/tmp/unix_unit_test"))
	}()
	// have to wait for the goroutine to start and run the server
	// otherwise the main thread will complete
	time.Sleep(5 * time.Millisecond)

	c, err := net.Dial("unix", "/tmp/unix_unit_test")
	assert.NoError(t, err)

	fmt.Fprintf(c, "GET /example HTTP/1.0\r\n\r\n")
	scanner := bufio.NewScanner(c)
	var response string
	for scanner.Scan() {
		response += scanner.Text()
	}
	assert.Contains(t, response, "HTTP/1.0 200", "should get a 200")
	assert.Contains(t, response, "it worked", "resp body should match")
}

func TestBadUnixSocket(t *testing.T) {
	router := New()
	assert.Error(t, router.RunUnix("#/tmp/unix_unit_test"))
}

func TestWithHttptestWithAutoSelectedPort(t *testing.T) {
	router := New()
	router.GET("/example", func(c *Context) { c.Reply(http.StatusOK, "it worked") })

	ts := httptest.NewServer(router)
	defer ts.Close()

	testRequest(t, ts.URL+"/example")
}

func TestWithHttptestWithSpecifiedPort(t *testing.T) {
	router := New()
	router.GET("/example", func(c *Context) { c.Reply(http.StatusOK, "it worked") })

	l, _ := net.Listen("tcp", ":8033")
	ts := httptest.Server{
		Listener: l,
		Config:   &http.Server{Handler: router},
	}
	ts.Start()
	defer ts.Close()

	testRequest(t, "http://localhost:8033/example")
}

type FB struct {
	Foo string `validate:"max=3"`
	Bar int    `validate:"max=3"`
}

func TestRender(t *testing.T) {
	Convey("render xml", t, func() {
		s := New()

		s.Handle("GET", "/test", func(ctx *Context) {
			var fb FB
			if err := ctx.Bind(&fb); err != nil {
				ctx.Reply(400, err)
				return
			}
			ctx.Reply(200, render.XML{fb})
		})

		r, err := http.NewRequest("GET", "/test?Foo=f&Bar=2", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		So(w.Body.String(), ShouldEqual, "<FB><Foo>f</Foo><Bar>2</Bar></FB>")
		So(w.Code, ShouldEqual, 200)
	})

	Convey("render xml", t, func() {
		s := New()

		s.Handle("GET", "/test", func(ctx *Context) {
			var fb FB
			if err := ctx.Bind(&fb); err != nil {
				ctx.Reply(400, err)
				return
			}
			ctx.Reply(200, render.XML{M{
				"Foo": "f",
				"Bar": 2,
			}})
		})

		r, err := http.NewRequest("GET", "/test", nil)
		So(err, ShouldBeNil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)

		equal := w.Body.String() == "<map><Foo>f</Foo><Bar>2</Bar></map>" || w.Body.String() == "<map><Bar>2</Bar><Foo>f</Foo></map>"
		So(equal, ShouldBeTrue)
		So(w.Code, ShouldEqual, 200)
	})
}

func f1(ctx *Context) {
	var fb FB
	if err := ctx.Bind(&fb); err != nil {
		ctx.Reply(400, err)
	}
	ctx.Map(fmt.Sprintf("%s%v", fb.Foo, fb.Bar))
}

func f2(ctx *Context) {
	ctx.Map(2)
}

func f3(i int, str string) (interface{}, error) {
	return str + fmt.Sprintf("f%v", i) + "f3", errors.New("in the end")
}

func TestMustGet(t *testing.T) {
	ctx := &Context{}
	r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
	var w http.ResponseWriter = httptest.NewRecorder()

	ctx.reset(w, r)
	ctx.Map(1)
	ctx.Map("string")

	var err error = errors.New("test")
	ctx.SetType(reflect.TypeOf(&err), reflect.ValueOf(&err))

	Convey("must get int and string", t, func() {
		var i int
		ctx.MustGet(&i)
		So(i, ShouldEqual, 1)

		var str string
		ctx.MustGet(&str)
		So(str, ShouldEqual, "string")
	})

	Convey("must get struct ptr", t, func() {
		var rr *http.Request
		ctx.MustGet(&rr)
		So(rr, ShouldEqual, r)

		var cc *Context
		ctx.MustGet(&cc)
		So(cc, ShouldEqual, ctx)
	})

	Convey("must get interface", t, func() {
		var ww http.ResponseWriter
		ctx.MustGet(&ww)
		So(ww, ShouldEqual, ctx.Writer)
	})

	Convey("must get interface ptr", t, func() {
		var perr *error
		ctx.MustGet(&perr)
		So(*perr, ShouldEqual, err)
	})
}

func TestMiddwareDefer(t *testing.T) {
	SetMode(TestMode)
	Convey("modify renderer", t, func() {
		s := New()
		cnt := 1
		addDefer := func(ctx *Context) {
			ctx.Writer.Defer(func() {
				cnt++
			})
		}

		s.Handle("GET", "/test", addDefer, func(ctx *Context) {
			ctx.Reply(200, nil)
		})
		r, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, r)
		So(w.Body.String(), ShouldEqual, "null")
		So(w.Code, ShouldEqual, 200)
		So(cnt, ShouldEqual, 2)
	})
}

func TestServer(t *testing.T) {
	SetMode(TestMode)
	Convey("api server returns string", t, func() {
		s := New()
		s.Handle("GET", "/test/:Foo", NewMiddware(f1, f2).Then(f3)...)
		r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, r)
		So(w.Body.String(), ShouldEqual, "f1f2f3")
	})
	Convey("api returns error 400", t, func() {
		f := func(fb FB) int {
			return fb.Bar
		}

		s := New()
		s.Handle("GET", "/test/:Foo", H(f))
		r, _ := http.NewRequest("GET", "/test/f?Bar=4", nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 400)
	})
	Convey("api returns []byte", t, func() {
		s := New()
		s.Use(f1)
		F3 := func(i int, str string) (interface{}, error) {
			return []byte(str + fmt.Sprintf("f%v", i) + "f3"), errors.New("in the end")
		}

		s.Handle("GET", "/test/:Foo", NewMiddware(f2).Then(F3)...)
		r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 200)
		So(w.Body.String(), ShouldEqual, "f1f2f3")
	})
	Convey("api returns error 500", t, func() {
		s := New(Recovery())

		F3 := func(c *Context, err error) (interface{}, error) {
			return nil, nil
		}

		s.Handle("GET", "/test/:Foo", NewMiddware(f1, f2).Then(F3)...)
		r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, r)
		So(w.Code, ShouldEqual, 500)
	})

	Convey("api returns json", t, func() {
		s := New()
		s.Use(f1, f2)
		type Rep struct {
			Code int
			Str  string
		}

		F3 := func(i int, str string) (interface{}, error) {
			return Rep{
				Code: 200,
				Str:  str + fmt.Sprintf("f%v", i) + "f3",
			}, errors.New("in the end")
		}

		s.GET("/test/:Foo", H(F3))
		r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		So(GetContentType(w.Header()), ShouldEqual, "application/json")

		rv := Rep{}
		err := json.NewDecoder(w.Body).Decode(&rv)
		So(err, ShouldBeNil)
		So(rv.Code, ShouldEqual, 200)
		So(rv.Str, ShouldEqual, "f1f2f3")
	})
}

func h1(c *gin.Context) {
	fb := FB{}
	if err := bind.Bind(&fb, c.Request, nil); err != nil {
		c.AbortWithError(400, err)
		return
	}
	fb.Foo = c.Param("Foo")
	c.Set("f1", fmt.Sprintf("%s%v", fb.Foo, fb.Bar))
	return
}
func h2(c *gin.Context) {
	c.Set("2", 2)
	return
}
func h3(c *gin.Context) {
	i, ok := c.Get("2")
	if !ok {
		c.AbortWithError(500, errors.New("can not find 3"))
	}

	f1, ok := c.Get("f1")
	if !ok {
		c.AbortWithError(500, errors.New("can not find f1"))
	}

	c.Error(errors.New("in the end"))
	c.String(200, "%sf%v%s", f1, i, "f3")
	return
}

func BenchmarkGin(b *testing.B) {
	gin.SetMode(gin.TestMode)
	s := gin.New()
	s.GET("/test/:Foo", h1, h2, h3)

	//test first
	r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, r)
	if w.Body.String() != "f1f2f3" {
		b.Error("ServeHTTP error")
		return
	}

	runRequest(b, s, "GET", "/test/f?Bar=1")
}

func F3(ctx *Context) {
	var i int
	var str string
	ctx.MustGet(&i)
	ctx.MustGet(&str)
	ctx.Error(errors.New("in the end"))
	ctx.Reply(200, str+fmt.Sprintf("f%v", i)+"f3")
}

func BenchmarkApi1(b *testing.B) {

	SetMode(TestMode)
	s := New()
	s.GET("/test/:Foo", f1, f2, F3)

	//test
	r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, r)
	if w.Body.String() != "f1f2f3" {
		b.Error("ServeHTTP error")
		return
	}

	runRequest(b, s, "GET", "/test/f?Bar=1")
}

func BenchmarkApi2(b *testing.B) {

	SetMode(TestMode)
	s := New()
	s.Use(f1, f2)
	s.GET("/test/:Foo", H(F3))

	//test
	r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, r)
	if w.Body.String() != "f1f2f3" {
		b.Error("ServeHTTP error")
		return
	}

	runRequest(b, s, "GET", "/test/f?Bar=1")
}

func BenchmarkApiHF(b *testing.B) {
	SetMode(TestMode)
	s := New()
	s.GET("/test/:Foo", f1, f2, H(f3))

	//test
	r, _ := http.NewRequest("GET", "/test/f?Bar=1", nil)
	w := httptest.NewRecorder()

	s.ServeHTTP(w, r)
	if w.Body.String() != "f1f2f3" {
		b.Error("ServeHTTP error")
		return
	}

	runRequest(b, s, "GET", "/test/f?Bar=1")
}

func m1(w http.ResponseWriter, r *http.Request, ctx martini.Context) {
	fb := FB{}
	if err := bind.Bind(&fb, r, nil); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	ctx.Map("f1")
	return
}

func m2(ctx martini.Context) {
	ctx.Map(2)
	return
}

func m3(i int, str string, ctx martini.Context) string {
	return str + fmt.Sprintf("f%v", i) + "f3"
}

func BenchmarkMartini(b *testing.B) {
	r := martini.NewRouter()
	m := martini.New()
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)

	r.Get("/test", m1, m2, m3)

	//test
	req, _ := http.NewRequest("GET", "/test?Foo=f&Bar=1", nil)
	w := httptest.NewRecorder()

	m.ServeHTTP(w, req)
	if w.Body.String() != "f1f2f3" {
		b.Error("ServeHTTP error")
		return
	}

	runRequest(b, m, "GET", "/test?Foo=f&Bar=1")
}
