package ginx

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
)

var (
	GinContextKey = "_GinCtxKey"
)

// GinContextToContext stores gin.Context in c.Request.Context
func GinContextToHttpContext(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), GinContextKey, c)
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

// GetGinContext gets gin.Context from context.Context
func GetGinContext(ctx context.Context) (*gin.Context, error) {
	ginCtx := ctx.Value(GinContextKey)
	if ginCtx == nil {
		err := fmt.Errorf("could not retrieve gin.Context by %s", GinContextKey)
		return nil, err
	}

	gc, ok := ginCtx.(*gin.Context)
	if !ok {
		err := fmt.Errorf("gin.Context has wrong type: %T", ginCtx)
		return nil, err
	}
	return gc, nil
}

func MustGetGinContext(ctx context.Context) *gin.Context {
	gc, err := GetGinContext(ctx)
	if err != nil {
		panic(err)
	}
	return gc
}
