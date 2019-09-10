package ginx

import (
	"github.com/zltgo/webkit/cache/lru"
	"github.com/zltgo/webkit/ratelimit"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHTTPMiddleware(t *testing.T) {
	is := require.New(t)
	gin.SetMode(gin.TestMode)

	request, err := http.NewRequest("GET", "/", nil)
	is.NoError(err)
	is.NotNil(request)
	request.RemoteAddr = "localhost:8080"

	lc := lru.New(1000)
	middleware := NewRatelimiter(lc, ratelimit.Opts(time.Minute, 10, 1))

	router := gin.New()
	router.Use(middleware)
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	success := int64(10)
	clients := int64(100)

	//
	// Sequential
	//

	for i := int64(1); i <= clients; i++ {

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, request)

		if i <= success {
			is.Equal(resp.Code, http.StatusOK)
		} else {
			is.Equal(resp.Code, http.StatusTooManyRequests)
		}
	}

	//
	// Concurrent
	//

	wg := &sync.WaitGroup{}
	counter := int64(0)
	t.Log(lc.GetStats())
	lc.Clear()

	for i := int64(1); i <= clients; i++ {
		wg.Add(1)
		go func() {

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, request)

			if resp.Code == http.StatusOK {
				atomic.AddInt64(&counter, 1)
			}

			wg.Done()
		}()
	}

	wg.Wait()
	is.Equal(success, atomic.LoadInt64(&counter))
}