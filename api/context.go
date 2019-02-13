package api

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"

	"github.com/zltgo/api/bind"
	"github.com/zltgo/api/inject"
	"github.com/zltgo/api/render"
	"github.com/zltgo/api/render/sse"
	"github.com/zltgo/api/tree"
	"github.com/zltgo/reflectx"
)

const (
	abortIndex int8 = math.MaxInt8 / 2
)

var (
	TypeRequest            = reflect.TypeOf(&http.Request{})
	TypeHttpResponseWriter = inject.InterfaceOf((*http.ResponseWriter)(nil))
	TypeResponseWriter     = inject.InterfaceOf((*ResponseWriter)(nil))
	TypeContext            = reflect.TypeOf(&Context{})
)

// Context represents the runtime context of current request of Macaron instance.
// It is the integration of most frequently used middlewares and helper methods.
type Context struct {
	Params    tree.Params
	writermem responseWriter
	Writer    ResponseWriter
	Request   *http.Request

	// inject
	typePairs inject.TypePairs

	//handlers
	middleware []Handler
	handlers   []Handler

	index int8
	// Errors is a list of errors attached to all the handlers/middlewares who used this context.
	Errors []error
}

// Next should be used only inside middleware.
// It executes the pending handlers in the chain inside the calling handler.
func (ctx *Context) Next() {
	ctx.index++
	mid := int8(len(ctx.middleware))
	sum := mid + int8(len(ctx.handlers))

	// call global middwares
	for ; ctx.index < mid; ctx.index++ {
		ctx.middleware[ctx.index](ctx)
	}

	// call handlers
	for ; ctx.index < sum; ctx.index++ {
		ctx.handlers[ctx.index-mid](ctx)
	}
	return
}

// called by server, only once
func (ctx *Context) run(handlers []Handler) {
	ctx.handlers = handlers
	ctx.Next()

	// write the http status now if the response body was not written.
	ctx.Writer.WriteHeaderNow()
}

// If there is an error, it will be of type validator.ValidationErrors.
func (ctx *Context) Invoke(f interface{}) ([]reflect.Value, error) {
	rv, err := inject.Invoke(f, inject.GetterFunc(ctx.GetType))
	if err != nil && !bind.IsValidationError(err) {
		panic(fmt.Errorf("invoke %s: %v", FunctionName(f), err))
	}
	return rv, err
}

// Maps the concrete value of val to its dynamic type using reflect.TypeOf.
func (ctx *Context) Map(val interface{}) {
	ctx.typePairs.Set(reflect.TypeOf(val), reflect.ValueOf(val))
}

// MapTo is useful when you mapping a interface.
// For example:
// type SpecialString interface {}
// var s SpecialString = "string"
// reflect.TypeOf(s) is string, not SpecialString
func (ctx *Context) MapTo(val interface{}, ifacePtr interface{}) {
	ctx.typePairs.Set(inject.InterfaceOf(ifacePtr), reflect.ValueOf(val))
}

//  mapping a Type and Value.
func (ctx *Context) SetType(typ reflect.Type, val reflect.Value) {
	ctx.typePairs.Set(typ, val)
}

// Bind struct by Request and Params.
func (ctx *Context) Bind(ptr interface{}) error {
	var extra map[string][]string
	if len(ctx.Params) > 0 {
		extra = make(map[string][]string, len(ctx.Params))
		for _, v := range ctx.Params {
			extra[v.Key] = []string{v.Value}
		}
	}
	return bind.Bind(ptr, ctx.Request, extra)
}

//implementation of inject.GetterFunc, used for Invoke.
func (ctx *Context) GetType(typ reflect.Type) (reflect.Value, error) {
	// find types inside
	switch typ {
	case TypeRequest:
		return reflect.ValueOf(ctx.Request), nil
	case TypeHttpResponseWriter, TypeResponseWriter:
		return reflect.ValueOf(ctx.Writer), nil
	case TypeContext:
		return reflect.ValueOf(ctx), nil
	}

	//find type in typePairs
	v, err := ctx.typePairs.Get(typ)
	if err != nil && reflectx.Deref(typ).Kind() == reflect.Struct {
		// try to bind typ from request
		var extra map[string][]string
		if len(ctx.Params) > 0 {
			extra = make(map[string][]string, len(ctx.Params))
			for _, v := range ctx.Params {
				extra[v.Key] = []string{v.Value}
			}
		}
		v, err = bind.GetType(typ, ctx.Request, extra)
	}

	return v, err
}

// Used to get a mapped value.
// MustGet panics if value does not exsit.
// Example:
// 	var r *http.request
// 	ctx.MustGet(&r)
//	var w http.ResponseWriter
// 	ctx.MustGet(&w)
func (ctx *Context) MustGet(ptr interface{}) {
	typ := reflect.TypeOf(ptr).Elem()
	v := reflect.ValueOf(ptr).Elem()
	rv, err := ctx.GetType(typ)
	if err != nil {
		panic(err)
	}
	v.Set(rv)
}

// Reset resets a context for a new http request.
func (ctx *Context) reset(w http.ResponseWriter, req *http.Request) {
	ctx.writermem.reset(w)
	ctx.Writer = &ctx.writermem
	ctx.Request = req

	ctx.typePairs = ctx.typePairs[0:0]
	ctx.Params = ctx.Params[0:0]
	ctx.Errors = ctx.Errors[0:0]

	ctx.middleware = nil
	ctx.handlers = nil
	ctx.index = -1
}

// Status sets the HTTP response code.
func (ctx *Context) Status(code int) {
	ctx.writermem.WriteHeader(code)
}

func (ctx *Context) Error(err error) {
	if err != nil {
		ctx.Errors = append(ctx.Errors, err)
	}
}

// write code and value to ResponseWriter.
// Render prevents pending handlers from being called.
// Let's say you have an authorization middleware that validates that the current request is authorized.
// If the authorization fails (ex: the password does not match), call Render to ensure the remaining handlers
// for this request are not called.
func (ctx *Context) Reply(code int, val interface{}) {
	ctx.Status(code)
	ctx.index = abortIndex

	if !bodyAllowedForStatus(code) {
		ctx.Writer.WriteHeaderNow()
		return
	}

	var err error
	switch v := val.(type) {
	case []byte:
		_, err = ctx.writermem.Write(v)
	case string:
		_, err = ctx.writermem.WriteString(v)
	case func(w io.Writer) bool:
		ctx.Stream(v)
	case io.Reader:
		_, err = io.Copy(ctx.Writer, v)
	case render.Render:
		err = v.Render(ctx.Writer)
	default: //nil or obj
		err = render.JSON{Data: val}.Render(ctx.Writer)
	}

	if err != nil {
		panic(err)
	}
}

//process the return values of ctx.Invoke or HandlerFunc.
func (ctx *Context) ReplyValues(values []reflect.Value) {
	// Process error and code first
	var code int
	idx := -1
	for i := range values {
		switch v := values[i].Interface().(type) {
		case nil:
		case error:
			ctx.Error(v)
		case int:
			code = v
		default:
			//get index of obj
			idx = i
		}
	}
	if code == 0 {
		// do nothing if no code and obj
		if idx < 0 {
			return
		}
		// Default code is http.StatusOK
		code = http.StatusOK
	}
	// render
	if idx < 0 {
		ctx.Reply(code, nil)
	} else {
		ctx.Reply(code, values[idx].Interface())
	}
	return
}

func (ctx *Context) Stream(step func(w io.Writer) bool) {
	w := ctx.Writer
	clientGone := w.CloseNotify()
	for {
		select {
		case <-clientGone:
			return
		default:
			keepOpen := step(w)
			w.Flush()
			if !keepOpen {
				return
			}
		}
	}
}

// SSEvent writes a Server-Sent Event into the body stream.
func (ctx *Context) SSEvent(name string, message interface{}) {
	sse.Event{
		Event: name,
		Data:  message,
	}.Render(ctx.Writer)
}

// bodyAllowedForStatus is a copy of http.bodyAllowedForStatus non-exported function.
func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == 204:
		return false
	case status == 304:
		return false
	}
	return true
}
