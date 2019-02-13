package api

import (
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/zltgo/api/tree"
)

// Version is Framework's version.
const (
	Version                = "v0.1"
	defaultMultipartMemory = 32 << 20 // 32 MB
)

var default404Handler = func(ctx *Context) {
	ctx.Writer.WriteHeader(http.StatusNotFound)
	ctx.Writer.WriteString("404 page not found")
}

var default405Handler = func(ctx *Context) {
	ctx.Writer.WriteHeader(http.StatusMethodNotAllowed)
	ctx.Writer.WriteString("405 method not allowed")
}

type Handler func(*Context)

// Wrap the function to api handler.
func H(fn interface{}) Handler {
	switch h := fn.(type) {
	case Handler:
		return h
	case func(*Context):
		return h
	case http.Handler:
		return func(c *Context) {
			h.ServeHTTP(c.Writer, c.Request)
		}
	case http.HandlerFunc:
		return func(c *Context) {
			h.ServeHTTP(c.Writer, c.Request)
		}
	case func(http.ResponseWriter, *http.Request):
		return func(c *Context) {
			h(c.Writer, c.Request)
		}
	case nil:
		panic("input function can't be nil")
	}

	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		panic("can not warp " + t.String() + "to api.Handler")
	}

	return func(ctx *Context) {
		vs, err := ctx.Invoke(fn)
		if err != nil {
			// validate failed.
			ctx.Reply(http.StatusBadRequest, err)
			return
		}
		// do nothing if fn does not have return values
		if len(vs) > 0 {
			ctx.ReplyValues(vs)
		}
	}
}

// Server is the framework's instance, it contains the muxer, middleware and configuration settings.
// Create an instance of Server, by using New() or Default()
type Server struct {
	pool       sync.Pool
	router     tree.Trees
	middleware []Handler
	notFound   []Handler
	noMethod   []Handler

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// Value of 'maxMemory' param that is given to http.Request's ParseMultipartForm
	// method call.
	MaxMultipartMemory int64
}

// New returns a new blank Server instance without any middleware attached.
// By default the configuration is:
// - RedirectTrailingSlash:  true
// - RedirectFixedPath:      false
// - HandleMethodNotAllowed: false
// - ForwardedByClientIP:    true
// - UseRawPath:             false
// - UnescapePathValues:     true
func New(middleware ...Handler) *Server {
	debugPrintWARNINGNew()
	serv := &Server{
		router:                make(tree.Trees, 0, 9),
		notFound:              []Handler{default404Handler},
		noMethod:              []Handler{default405Handler},
		middleware:            middleware,
		RedirectTrailingSlash: true,
		RedirectFixedPath:     false,
		MaxMultipartMemory:    defaultMultipartMemory,
	}

	serv.pool.New = func() interface{} {
		return serv.allocateContext()
	}
	return serv
}

func (serv *Server) allocateContext() *Context {
	return &Context{middleware: serv.middleware}
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default() *Server {
	return New(Logger(), Recovery())
}

// Use attachs a global middleware to the router. ie. the middleware attached though Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (serv *Server) Use(middleware ...Handler) {
	serv.middleware = append(serv.middleware, middleware...)
}

// Handle registers a new request handle and middleware with the given path and method.
// The last handler should be the real handler, the other ones should be middleware that can and should be shared among different routes.
// See the example code in github.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (serv *Server) Handle(method, url string, handlers ...Handler) {
	Assert(url[0] == '/', "url must begin with '/'")
	Assert(len(method) > 0, "http method can not be empty")
	Assert(len(handlers) > 0, "there must be at least one handler")
	Assert(len(serv.middleware)+len(handlers) < int(abortIndex), "too many handlers")

	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE":
		serv.router.Add(method, url, handlers)
	case "ANY":
		serv.router.Add("GET", url, handlers)
		serv.router.Add("POST", url, handlers)
		serv.router.Add("PUT", url, handlers)
		serv.router.Add("DELETE", url, handlers)
		serv.router.Add("PATCH", url, handlers)
		serv.router.Add("HEAD", url, handlers)
		serv.router.Add("OPTIONS", url, handlers)
		serv.router.Add("CONNECT", url, handlers)
		serv.router.Add("TRACE", url, handlers)
	default:
		panic("unknown http method: " + method)
	}

	debugPrintRoute(method, url, handlers)
	return
}

// POST is a shortcut for router.Handle("POST", url, handle).
func (serv *Server) POST(url string, handlers ...Handler) {
	serv.Handle("POST", url, handlers...)
}

// GET is a shortcut for router.Handle("GET", url, handle).
func (serv *Server) GET(url string, handlers ...Handler) {
	serv.Handle("GET", url, handlers...)
}

// DELETE is a shortcut for router.Handle("DELETE", url, handle).
func (serv *Server) DELETE(url string, handlers ...Handler) {
	serv.Handle("DELETE", url, handlers...)
}

// PATCH is a shortcut for router.Handle("PATCH", url, handle).
func (serv *Server) PATCH(url string, handlers ...Handler) {
	serv.Handle("PATCH", url, handlers...)
}

// PUT is a shortcut for router.Handle("PUT", url, handle).
func (serv *Server) PUT(url string, handlers ...Handler) {
	serv.Handle("PUT", url, handlers...)
}

// OPTIONS is a shortcut for router.Handle("OPTIONS", url, handle).
func (serv *Server) OPTIONS(url string, handlers ...Handler) {
	serv.Handle("OPTIONS", url, handlers...)
}

// HEAD is a shortcut for router.Handle("HEAD", url, handle).
func (serv *Server) HEAD(url string, handlers ...Handler) {
	serv.Handle("HEAD", url, handlers...)
}

// Any registers a route that matches all the HTTP methods.
// GET, POST, PUT, PATCH, HEAD, OPTIONS, DELETE, CONNECT, TRACE.
func (serv *Server) ANY(url string, handlers ...Handler) {
	serv.Handle("ANY", url, handlers...)
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (serv *Server) Run(addr string) (err error) {
	defer func() { debugPrintError(err) }()

	debugPrint("Listening and serving HTTP on %s\n", addr)
	err = http.ListenAndServe(addr, serv)
	return
}

// RunTLS attaches the router to a http.Server and starts listening and serving HTTPS (secure) requests.
// It is a shortcut for http.ListenAndServeTLS(addr, certFile, keyFile, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (serv *Server) RunTLS(addr, certFile, keyFile string) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugPrintError(err) }()

	err = http.ListenAndServeTLS(addr, certFile, keyFile, serv)
	return
}

// RunUnix attaches the router to a http.Server and starts listening and serving HTTP requests
// through the specified unix socket (ie. a file).
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (serv *Server) RunUnix(file string) (err error) {
	debugPrint("Listening and serving HTTP on unix:/%s", file)
	defer func() { debugPrintError(err) }()

	os.Remove(file)
	listener, err := net.Listen("unix", file)
	if err != nil {
		return
	}
	defer listener.Close()
	err = http.Serve(listener, serv)
	return
}

// ServeHTTP conforms to the http.Handler interface.
func (serv *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := serv.pool.Get().(*Context)
	c.reset(w, req)

	serv.handleHTTPRequest(c)

	serv.pool.Put(c)
}

// NotFound adds handlers for 404 error.
func (serv *Server) NotFound(handlers ...Handler) {
	Assert(len(handlers) > 0, "there must be at least one handler")
	Assert(len(serv.middleware)+len(handlers) < int(abortIndex), "too many handlers")
	serv.notFound = handlers
}

// NoMethod sets the handlers called in case of 405 error.
func (serv *Server) NoMethod(handlers ...Handler) {
	Assert(len(handlers) > 0, "there must be at least one handler")
	Assert(len(serv.middleware)+len(handlers) < int(abortIndex), "too many handlers")
	serv.noMethod = handlers
}

func (serv *Server) handleHTTPRequest(ctx *Context) {
	httpMethod := ctx.Request.Method
	path := ctx.Request.URL.Path

	ctx.middleware = serv.middleware
	// Find root of the tree for the given HTTP method
	root := serv.router.GetTree(httpMethod)
	if root != nil {
		// Find route in tree
		handle, params, tsr := root.GetHandle(path, ctx.Params)
		if handle != nil {
			ctx.Params = params
			ctx.run(handle.([]Handler))
			return
		}

		// redirect if necessary.
		if httpMethod != "CONNECT" && path != "/" {
			if tsr && serv.RedirectTrailingSlash {
				redirectTrailingSlash(ctx)
				return
			}
			if serv.RedirectFixedPath && redirectFixedPath(ctx, root, serv.RedirectFixedPath) {
				return
			}
		}
	}

	// check other method exist or not
	for _, tree := range serv.router {
		// Skip the requested method - we already tried this one
		if tree.Name != httpMethod {
			if h, _, _ := tree.Root.GetHandle(path, nil); h != nil {
				ctx.run(serv.noMethod)
				return
			}
		}
	}

	//not found
	ctx.run(serv.notFound)
	return
}

// StaticFile registers a single route in order to server a single file of the local filesystem.
// router.StaticFile("/favicon.ico", "./resources/favicon.ico")
func (serv *Server) StaticFile(url, filepath string) {
	if strings.Contains(url, ":") || strings.Contains(url, "*") {
		panic("URL parameters can not be used when serving a static file")
	}
	handler := func(c *Context) {
		http.ServeFile(c.Writer, c.Request, filepath)
	}
	serv.GET(url, handler)
	serv.HEAD(url, handler)
	return
}

// Static serves files from the given file system root.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use :
//     router.Static("/static", "./tmp/files")
func (serv *Server) Static(urlPrefix, root string) {
	if strings.Contains(urlPrefix, ":") || strings.Contains(urlPrefix, "*") {
		panic("URL parameters can not be used when serving a static folder")
	}
	handler := func(c *Context) {
		// Open file
		fpath := strings.TrimPrefix(c.Request.URL.Path, urlPrefix)
		f, fi, err := openFile(filepath.Join(root, fpath))

		if err != nil {
			if os.IsNotExist(err) {
				default404Handler(c)
			} else {
				c.Error(err)
				c.Writer.WriteHeader(http.StatusInternalServerError)
				c.Writer.WriteString("500 internal server error")
			}
			return
		}
		defer f.Close()
		http.ServeContent(c.Writer, c.Request, fi.Name(), fi.ModTime(), f)
		return
	}

	url := path.Join(urlPrefix, "/*filepath")
	// Register GET and HEAD handlers
	serv.GET(url, handler)
	serv.HEAD(url, handler)
	return
}

func redirectTrailingSlash(c *Context) {
	req := c.Request
	path := req.URL.Path
	code := http.StatusMovedPermanently // Permanent redirect, request with GET method
	if req.Method != "GET" {
		code = http.StatusTemporaryRedirect
	}

	if len(path) > 1 && path[len(path)-1] == '/' {
		req.URL.Path = path[:len(path)-1]
	} else {
		req.URL.Path = path + "/"
	}
	debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
	http.Redirect(c.Writer, req, req.URL.String(), code)
	c.writermem.WriteHeaderNow()
}

func redirectFixedPath(c *Context, root *tree.Node, trailingSlash bool) bool {
	req := c.Request
	path := req.URL.Path

	fixedPath, found := root.FindCaseInsensitivePath(
		cleanPath(path),
		trailingSlash,
	)
	if found {
		code := http.StatusMovedPermanently // Permanent redirect, request with GET method
		if req.Method != "GET" {
			code = http.StatusTemporaryRedirect
		}
		req.URL.Path = string(fixedPath)
		debugPrint("redirecting request %d: %s --> %s", code, path, req.URL.String())
		http.Redirect(c.Writer, req, req.URL.String(), code)
		c.writermem.WriteHeaderNow()
		return true
	}
	return false
}

/****************route info**********************/
type Route struct {
	Method   string
	Url      string
	Handlers []Handler
}

type Routes []Route

// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path and the handler name.
func (serv *Server) GetRoutes() (routes Routes) {
	for _, tree := range serv.router {
		tree.Root.Walk(func(path string, handle interface{}) bool {
			if handle != nil {
				routes = append(routes, Route{
					Method:   tree.Name,
					Url:      path,
					Handlers: handle.([]Handler),
				})
			}
			return true
		})
	}
	return routes
}

func (serv *Server) AddRoutes(routes Routes) {
	for i := range routes {
		serv.Handle(routes[i].Method, routes[i].Url, routes[i].Handlers...)
	}
	return
}

// Add a Route to Routes.
// e.g. Add("POST:/api/login", jwt.LoginHandle)
func (routes *Routes) Add(url string, handlers ...interface{}) {
	slice := strings.SplitN(url, ":", 2)

	*routes = append(*routes, Route{
		Method:   slice[0],
		Url:      slice[1],
		Handlers: Middware{}.Then(handlers...),
	})
	return
}
