package graphorm

import (
	"context"
	"github.com/99designs/gqlgen/graphql"
	"github.com/jinzhu/gorm"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/vektah/gqlparser/gqlerror"
	"github.com/zltgo/webkit/ginx"
	"golang.org/x/exp/errors/fmt"
	"net/http"
)

//just transfer the panic and gin will handler it.
func Recover(ctx context.Context, err interface{}) error {
	errWithStack := errors.WithStack(fmt.Errorf("%v", err))
	errWithStack = fmt.Errorf("%+v", errWithStack)
	return NewError(http.StatusInternalServerError, errWithStack)
}

type Error struct {
	code    int
	message string
	err     error
}

func NewError(code int, err error, message ...string) error {
	if err == nil {
		return nil
	}

	myError := &Error{
		code: code,
		err:  err,
	}
	if len(message) == 0 || message[0] == "" {
		myError.message = StatusText(code)
	} else {
		myError.message = message[0]
	}
	return myError
}

// StatusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	var txt string
	switch code {
	case http.StatusBadRequest:
		txt = "无效的请求参数"
	case http.StatusUnauthorized:
		txt = "认证信息已失效，请重新登录"
	case http.StatusPaymentRequired:
		txt = "需要付费"
	case http.StatusNotFound:
		txt = "资源不存在"
	case http.StatusForbidden:
		txt = "无权访问"
	case http.StatusConflict:
		txt = "唯一约束失败"
	case http.StatusPreconditionFailed:
		txt = "条件不允许"
	case http.StatusInternalServerError:
		txt = "服务器内部错误"
	default:
		txt = http.StatusText(code)
	}
	return txt
}

func (e *Error) Error() string {
	return e.err.Error()
}

func ErrorPresenter(ctx context.Context, e error) *gqlerror.Error {
	c := ginx.MustGetGinContext(ctx)
	c.Error(e)

	// any special logic you want to do here. Must specify path for correct null bubbling behaviour.
	if err, ok := e.(*Error); ok {
		e = &gqlerror.Error{
			Message: err.message,
			Extensions: map[string]interface{}{
				"code": err.code,
			},
		}
	}

	return graphql.DefaultErrorPresenter(ctx, e)
}

// check error type from db driver
func ErrorCode(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if gorm.IsRecordNotFoundError(err) {
		return http.StatusNotFound
	}
	//TODO: mysql,postgres
	if sqliteErr, ok := err.(sqlite3.Error); ok {
		if sqliteErr.Code == sqlite3.ErrConstraint {
			return http.StatusConflict
		}
	}
	return http.StatusInternalServerError
}

//IsServerError returns true in case of InternalServerError
func IsInternalServerError(err error) bool {
	return ErrorCode(err) == http.StatusInternalServerError
}
