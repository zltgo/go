package api

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

const (
	noWritten     = -1
	defaultStatus = 200
)

type ResponseWriter interface {
	http.ResponseWriter
	http.Hijacker
	http.Flusher
	http.CloseNotifier

	// Returns the HTTP response status code of the current request.
	Status() int

	// Returns the number of bytes already written into the response http body.
	// See Written()
	Size() int

	// Writes the string into the response body.
	WriteString(string) (int, error)

	// Returns true if the response body was already written.
	Written() bool

	// Forces to write the http header (status code + headers).
	WriteHeaderNow()

	// Set functions should call before write response.
	Defer(f func())
}

var _ ResponseWriter = &responseWriter{}

type responseWriter struct {
	http.ResponseWriter
	beforeFuncs []func()
	size        int
	status      int
}

// Set functions should call before write response.
// It is useful for some middleware, see 'zltgo/api/session' for example.
func (m *responseWriter) Defer(f func()) {
	m.beforeFuncs = append(m.beforeFuncs, f)
}

func (w *responseWriter) reset(writer http.ResponseWriter) {
	w.ResponseWriter = writer
	w.beforeFuncs = nil
	w.size = noWritten
	w.status = defaultStatus
}

func (w *responseWriter) WriteHeader(code int) {
	if code > 0 && w.status != code {
		if w.Written() {
			debugPrint("[WARNING] Headers were already written. Wanted to override status code %d with %d", w.status, code)
		}
		w.status = code
	}
}

func (w *responseWriter) WriteHeaderNow() {
	if !w.Written() {
		w.size = 0
		for i := len(w.beforeFuncs) - 1; i >= 0; i-- {
			w.beforeFuncs[i]()
		}
		w.ResponseWriter.WriteHeader(w.status)
	}
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	w.WriteHeaderNow()
	n, err = w.ResponseWriter.Write(data)
	w.size += n
	return
}

func (w *responseWriter) WriteString(s string) (n int, err error) {
	w.WriteHeaderNow()
	n, err = io.WriteString(w.ResponseWriter, s)
	w.size += n
	return
}

func (w *responseWriter) Status() int {
	return w.status
}

func (w *responseWriter) Size() int {
	return w.size
}

func (w *responseWriter) Written() bool {
	return w.size != noWritten
}

// Hijack implements the http.Hijacker interface.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.size < 0 {
		w.size = 0
	}
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

// CloseNotify implements the http.CloseNotify interface.
func (w *responseWriter) CloseNotify() <-chan bool {
	return w.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Flush implements the http.Flush interface.
func (w *responseWriter) Flush() {
	w.ResponseWriter.(http.Flusher).Flush()
}