// Copyright 2014 Manu Martinez-Almeida.  All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package api

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zltgo/api/render"
	"github.com/zltgo/api/render/sse"
)

func testRouteOK(method string, t *testing.T) {
	passed := false
	passedAny := false
	r := New()
	r.ANY("/test2", func(c *Context) {
		passedAny = true
	})
	r.Handle(method, "/test", func(c *Context) {
		passed = true
		c.Reply(200, "")
	})

	w := performRequest(r, method, "/test")
	assert.True(t, passed)
	assert.Equal(t, w.Code, http.StatusOK)

	performRequest(r, method, "/test2")
	assert.True(t, passedAny)
}

// TestSingleRouteOK tests that POST route is correctly invoked.
func testRouteNotOK(method string, t *testing.T) {
	passed := false
	router := New()
	router.Handle(method, "/test_2", func(c *Context) {
		passed = true
	})

	w := performRequest(router, method, "/test")

	assert.False(t, passed)
	assert.Equal(t, w.Code, http.StatusNotFound)
}

// TestSingleRouteOK tests that POST route is correctly invoked.
func testRouteNotOK2(method string, t *testing.T) {
	passed := false
	router := New()

	var methodRoute string
	if method == "POST" {
		methodRoute = "GET"
	} else {
		methodRoute = "POST"
	}
	router.Handle(methodRoute, "/test", func(c *Context) {
		passed = true
	})

	w := performRequest(router, method, "/test")

	assert.False(t, passed)
	assert.Equal(t, w.Code, http.StatusMethodNotAllowed)
}

func TestRouterMethod(t *testing.T) {
	router := New()
	router.PUT("/hey2", func(c *Context) {
		c.Reply(200, "sup2")
	})

	router.PUT("/hey", func(c *Context) {
		c.Reply(200, "called")
	})

	router.PUT("/hey3", func(c *Context) {
		c.Reply(200, "sup3")
	})

	w := performRequest(router, "PUT", "/hey")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "called")
}

func TestRouterGroupRouteOK(t *testing.T) {
	testRouteOK("GET", t)
	testRouteOK("POST", t)
	testRouteOK("PUT", t)
	testRouteOK("PATCH", t)
	testRouteOK("HEAD", t)
	testRouteOK("OPTIONS", t)
	testRouteOK("DELETE", t)
	testRouteOK("CONNECT", t)
	testRouteOK("TRACE", t)
}

func TestRouteNotOK(t *testing.T) {
	testRouteNotOK("GET", t)
	testRouteNotOK("POST", t)
	testRouteNotOK("PUT", t)
	testRouteNotOK("PATCH", t)
	testRouteNotOK("HEAD", t)
	testRouteNotOK("OPTIONS", t)
	testRouteNotOK("DELETE", t)
	testRouteNotOK("CONNECT", t)
	testRouteNotOK("TRACE", t)
}

func TestRouteNotOK2(t *testing.T) {
	testRouteNotOK2("GET", t)
	testRouteNotOK2("POST", t)
	testRouteNotOK2("PUT", t)
	testRouteNotOK2("PATCH", t)
	testRouteNotOK2("HEAD", t)
	testRouteNotOK2("OPTIONS", t)
	testRouteNotOK2("DELETE", t)
	testRouteNotOK2("CONNECT", t)
	testRouteNotOK2("TRACE", t)
}

func TestRouteRedirectTrailingSlash(t *testing.T) {
	router := New()
	router.RedirectFixedPath = false
	router.RedirectTrailingSlash = true
	router.GET("/path", func(c *Context) {})
	router.GET("/path2/", func(c *Context) {})
	router.POST("/path3", func(c *Context) {})
	router.PUT("/path4/", func(c *Context) {})

	w := performRequest(router, "GET", "/path/")
	assert.Equal(t, w.Header().Get("Location"), "/path")
	assert.Equal(t, w.Code, 301)

	w = performRequest(router, "GET", "/path2")
	assert.Equal(t, w.Header().Get("Location"), "/path2/")
	assert.Equal(t, w.Code, 301)

	w = performRequest(router, "POST", "/path3/")
	assert.Equal(t, w.Header().Get("Location"), "/path3")
	assert.Equal(t, w.Code, 307)

	w = performRequest(router, "PUT", "/path4")
	assert.Equal(t, w.Header().Get("Location"), "/path4/")
	assert.Equal(t, w.Code, 307)

	w = performRequest(router, "GET", "/path")
	assert.Equal(t, w.Code, 200)

	w = performRequest(router, "GET", "/path2/")
	assert.Equal(t, w.Code, 200)

	w = performRequest(router, "POST", "/path3")
	assert.Equal(t, w.Code, 200)

	w = performRequest(router, "PUT", "/path4/")
	assert.Equal(t, w.Code, 200)

	router.RedirectTrailingSlash = false

	w = performRequest(router, "GET", "/path/")
	assert.Equal(t, w.Code, 404)
	w = performRequest(router, "GET", "/path2")
	assert.Equal(t, w.Code, 404)
	w = performRequest(router, "POST", "/path3/")
	assert.Equal(t, w.Code, 404)
	w = performRequest(router, "PUT", "/path4")
	assert.Equal(t, w.Code, 404)
}

func TestRouteRedirectFixedPath(t *testing.T) {
	router := New()
	router.RedirectFixedPath = true
	router.RedirectTrailingSlash = false

	router.GET("/path", func(c *Context) {})
	router.GET("/Path2", func(c *Context) {})
	router.POST("/PATH3", func(c *Context) {})
	router.POST("/Path4/", func(c *Context) {})

	w := performRequest(router, "GET", "/PATH")
	assert.Equal(t, w.Header().Get("Location"), "/path")
	assert.Equal(t, w.Code, 301)

	w = performRequest(router, "GET", "/path2")
	assert.Equal(t, w.Header().Get("Location"), "/Path2")
	assert.Equal(t, w.Code, 301)

	w = performRequest(router, "POST", "/path3")
	assert.Equal(t, w.Header().Get("Location"), "/PATH3")
	assert.Equal(t, w.Code, 307)

	w = performRequest(router, "POST", "/path4")
	assert.Equal(t, w.Header().Get("Location"), "/Path4/")
	assert.Equal(t, w.Code, 307)
}

// TestContextParamsGet tests that a parameter can be parsed from the URL.
func TestRouteParamsByName(t *testing.T) {
	name := ""
	lastName := ""
	wild := ""
	router := New()
	router.GET("/test/:name/:last_name/*wild", func(c *Context) {
		name = c.Params.ByName("name")
		lastName = c.Params.ByName("last_name")
		var ok bool
		wild, ok = c.Params.Get("wild")

		assert.True(t, ok)
		assert.Equal(t, name, c.Params.ByName("name"))
		assert.Equal(t, name, c.Params.ByName("name"))
		assert.Equal(t, lastName, c.Params.ByName("last_name"))

		assert.Empty(t, c.Params.ByName("wtf"))
		assert.Empty(t, c.Params.ByName("wtf"))

		wtf, ok := c.Params.Get("wtf")
		assert.Empty(t, wtf)
		assert.False(t, ok)
	})

	w := performRequest(router, "GET", "/test/john/smith/is/super/great")

	assert.Equal(t, w.Code, 200)
	assert.Equal(t, name, "john")
	assert.Equal(t, lastName, "smith")
	assert.Equal(t, wild, "/is/super/great")
}

// TestHandleStaticFile - ensure the static file handles properly
func TestRouteStaticFile(t *testing.T) {
	// SETUP file
	testRoot, _ := os.Getwd()
	f, err := ioutil.TempFile(testRoot, "")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("Api Web Framework")
	f.Close()

	dir, filename := filepath.Split(f.Name())

	// SETUP api
	router := New()
	router.Static("/using_static", dir)
	router.StaticFile("/result", f.Name())

	w := performRequest(router, "GET", "/using_static/"+filename)
	w2 := performRequest(router, "GET", "/result")

	assert.Equal(t, w, w2)
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "Api Web Framework")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/plain; charset=utf-8")

	w3 := performRequest(router, "HEAD", "/using_static/"+filename)
	w4 := performRequest(router, "HEAD", "/result")

	assert.Equal(t, w3, w4)
	assert.Equal(t, w3.Code, 200)
}

// TestHandleHeadToDir - ensure the root/sub dir handles properly
func TestRouteStaticNoListing(t *testing.T) {
	router := New()
	router.Static("/", "./")

	w := performRequest(router, "GET", "/")

	assert.Equal(t, w.Code, 404)
	assert.NotContains(t, w.Body.String(), "server.go")
}

func TestRouterMiddlewareAndStatic(t *testing.T) {
	router := New()
	router.Use(func(c *Context) {
		c.Writer.Header().Add("Last-Modified", "Mon, 02 Jan 2006 15:04:05 MST")
		c.Writer.Header().Add("Expires", "Mon, 02 Jan 2006 15:04:05 MST")
		c.Writer.Header().Add("X-Api", "Api Framework")
	})
	router.Static("/", "./")

	w := performRequest(router, "GET", "/server.go")

	assert.Equal(t, w.Code, 200)
	assert.Contains(t, w.Body.String(), "package api")
	assert.Equal(t, w.HeaderMap.Get("Content-Type"), "text/plain; charset=utf-8")
	assert.NotEqual(t, w.HeaderMap.Get("Last-Modified"), "Mon, 02 Jan 2006 15:04:05 MST")
	assert.Equal(t, w.HeaderMap.Get("Expires"), "Mon, 02 Jan 2006 15:04:05 MST")
	assert.Equal(t, w.HeaderMap.Get("x-Api"), "Api Framework")
}

func TestRouteNotAllowedEnabled(t *testing.T) {
	router := New()
	router.POST("/path", func(c *Context) {})
	w := performRequest(router, "GET", "/path")
	assert.Equal(t, w.Code, http.StatusMethodNotAllowed)

	router.NoMethod(func(c *Context) {
		c.Reply(http.StatusTeapot, "responseText")
	})
	w = performRequest(router, "GET", "/path")
	assert.Equal(t, w.Body.String(), "responseText")
	assert.Equal(t, w.Code, http.StatusTeapot)
}

func TestRouterNotFound(t *testing.T) {
	router := New()
	router.RedirectFixedPath = true
	router.GET("/path", func(c *Context) {})
	router.GET("/dir/", func(c *Context) {})
	router.GET("/", func(c *Context) {})

	testRoutes := []struct {
		route    string
		code     int
		location string
	}{
		{"/path/", 301, "/path"},   // TSR -/
		{"/dir", 301, "/dir/"},     // TSR +/
		{"", 301, "/"},             // TSR +/
		{"/PATH", 301, "/path"},    // Fixed Case
		{"/DIR/", 301, "/dir/"},    // Fixed Case
		{"/PATH/", 301, "/path"},   // Fixed Case -/
		{"/DIR", 301, "/dir/"},     // Fixed Case +/
		{"/../path", 301, "/path"}, // CleanPath
		{"/nope", 404, ""},         // NotFound
	}
	for _, tr := range testRoutes {
		w := performRequest(router, "GET", tr.route)
		assert.Equal(t, w.Code, tr.code)
		if w.Code != 404 {
			assert.Equal(t, fmt.Sprint(w.Header().Get("Location")), tr.location)
		}
	}

	// Test custom not found handler
	var notFound bool
	router.NotFound(func(c *Context) {
		c.Reply(404, "")
		notFound = true
	})
	w := performRequest(router, "GET", "/nope")
	assert.Equal(t, w.Code, 404)
	assert.True(t, notFound)

	// Test other method than GET (want 307 instead of 301)
	router.PATCH("/path", func(c *Context) {})
	w = performRequest(router, "PATCH", "/path/")
	assert.Equal(t, w.Code, 307)
	assert.Equal(t, fmt.Sprint(w.Header()), "map[Location:[/path]]")

	// Test special case where no node for the prefix "/" exists
	router = New()
	router.GET("/a", func(c *Context) {})
	w = performRequest(router, "GET", "/")
	assert.Equal(t, w.Code, 404)
}

func TestRouteRawPath(t *testing.T) {
	route := New()
	route.POST("/project/*name", func(c *Context) {
		name := c.Params.ByName("name")

		assert.Equal(t, "/Some/Other/Project", name)
		t.Log(c.Request.URL.Path, c.Request.URL.EscapedPath())
	})

	w := performRequest(route, "POST", "/project/Some%2FOther%2FProject")
	assert.Equal(t, 200, w.Code)
}

func TestRouteServeErrorWithWriteHeader(t *testing.T) {
	route := New()
	route.Use(func(c *Context) {
		c.Status(421)
		c.Next()
	})
	route.NotFound(func(*Context) {})
	w := performRequest(route, "GET", "/NotFound")
	assert.Equal(t, 421, w.Code)
	assert.Equal(t, 0, w.Body.Len())
}

func TestMiddlewareGeneralCase(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *Context) {
		signature += "C"
	})
	router.GET("/", func(c *Context) {
		signature += "D"
		c.Reply(200, nil)
	})
	router.NotFound(func(c *Context) {
		signature += " X "
	})
	router.NoMethod(func(c *Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, "ACDB", signature)
}

func TestMiddlewareNoRoute(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *Context) {
		signature += "C"
		c.Next()
		c.Next()
		c.Next()
		c.Next()
		signature += "D"
	})
	router.NotFound(func(c *Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *Context) {
		signature += "G"
		c.Reply(404, nil)
		signature += "H"
	})
	router.NoMethod(func(c *Context) {
		signature += " X "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 404, w.Code)
	assert.Equal(t, "ACEGHFDB", signature)
}

func TestMiddlewareNoMethodDisabled(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		signature += "B"
	})
	router.Use(func(c *Context) {
		signature += "C"
		c.Next()
		signature += "D"
	})
	router.NoMethod(func(c *Context) {
		signature += "E"
		c.Next()
		signature += "F"
	}, func(c *Context) {
		signature += "G"
		c.Reply(405, nil)
		signature += "H"
	})
	router.NotFound(func(c *Context) {
		signature += " X "
		c.Reply(404, nil)
	})
	router.POST("/", func(c *Context) {
		signature += " XX "
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 405, w.Code)
	assert.Equal(t, "ACEGHFDB", signature)
}

func TestMiddlewareAbort(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
	})
	router.Use(func(c *Context) {
		signature += "C"
		c.Reply(401, nil)
		c.Next()
		signature += "D"
	})
	router.GET("/", func(c *Context) {
		signature += " X "
		c.Next()
		signature += " XX "
	})

	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 401, w.Code)
	assert.Equal(t, "ACD", signature)
}

func TestMiddlewareAbortHandlersChainAndNext(t *testing.T) {
	signature := ""
	router := New()
	router.Use(func(c *Context) {
		signature += "A"
		c.Next()
		c.Reply(401, nil)
		signature += "B"

	})
	router.GET("/", func(c *Context) {
		signature += "C"
		//c.Reply(200, nil)
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 401, w.Code)
	assert.Equal(t, "ACB", signature)
}

// TestFailHandlersChain - ensure that Fail interrupt used middleware in fifo order as
// as well as Abort
func TestMiddlewareFailHandlersChain(t *testing.T) {
	// SETUP
	signature := ""
	router := New()
	router.Use(func(context *Context) {
		signature += "A"
		context.Reply(500, errors.New("foo"))
	})
	router.Use(func(context *Context) {
		signature += "B"
		context.Next()
		signature += "C"
	})
	// RUN
	w := performRequest(router, "GET", "/")

	// TEST
	assert.Equal(t, 500, w.Code)
	assert.Equal(t, "A", signature)
}

//sse, render
func TestMiddlewareWrite(t *testing.T) {
	router := New()
	router.Use(func(c *Context) {
		c.Status(400)
		c.Writer.WriteString("hola\n")
	})
	router.Use(func(c *Context) {
		c.Status(400)
		render.XML{Data: M{"foo": "bar"}}.Render(c.Writer)
	})
	router.Use(func(c *Context) {
		c.Status(400)
		render.JSON{Data: M{"foo": "bar"}}.Render(c.Writer)
	})
	router.GET("/", func(c *Context) {
		c.Status(400)
		render.JSON{Data: M{"foo": "bar"}}.Render(c.Writer)
	}, func(c *Context) {
		c.Reply(400, sse.Event{
			Event: "test",
			Data:  "message",
		})
	})

	w := performRequest(router, "GET", "/")

	assert.Equal(t, 400, w.Code)
	assert.Equal(t, strings.Replace("hola\n<map><foo>bar</foo></map>{\"foo\":\"bar\"}{\"foo\":\"bar\"}event:test\ndata:message\n\n", " ", "", -1), strings.Replace(w.Body.String(), " ", "", -1))
}
