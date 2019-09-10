package ginx

import (
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/zltgo/webkit/cache/lru"
	"github.com/zltgo/webkit/ratelimit"
	"golang.org/x/exp/errors/fmt"
	"net/http"
)

type IP string

func NewRatelimiter(lc *lru.Cache, rates []ratelimit.Rate) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("can not resolve remote addr: %s", c.Request.RemoteAddr))
			return
		}

		//get rate limit stats
		limit := lc.Getsert(IP(ip), func()interface{}{
			return ratelimit.NewLimiter(rates)
		})

		if limit.(*ratelimit.Limiter).Reached() {
			c.AbortWithError(http.StatusTooManyRequests, errors.New("rate limited"))
			return
		}
		c.Next()
	}
}