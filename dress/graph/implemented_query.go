package graph

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/zltgo/dress/model"
	"github.com/zltgo/webkit/ginx"
	gm "github.com/zltgo/webkit/graphorm"
	"github.com/zltgo/webkit/jwt"
	"net/http"
)

type Resolver struct{}

func (r *Resolver) Account() AccountResolver {
	return &accountResolver{r}
}
func (r *Resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}
func (r *Resolver) Query() QueryResolver {
	return &queryResolver{r}
}
func (r *Resolver) Store() StoreResolver {
	return &storeResolver{r}
}
func (r *Resolver) User() UserResolver {
	return &userResolver{r}
}

type mutationResolver struct{ *Resolver }

type queryResolver struct{ *Resolver }

func (r *Resolver) GetCtx(ctx context.Context, opName string) (c *gin.Context, m *model.Model, au *jwt.Auth, err error) {
	c = ginx.MustGetGinContext(ctx)
	m = c.MustGet("model").(*model.Model)
	au = c.MustGet("auth").(*jwt.Auth)

	clientIp := c.ClientIP()
	if clientIp == "" {
		err = gm.NewError(http.StatusBadRequest, fmt.Errorf("can not resolve remote addr: %s", c.Request.RemoteAddr))
		return
	}

	if code, e := m.CheckPermission(nil, clientIp, opName); e != nil {
		err = gm.NewError(code, e)
	}
	return
}

// Get Model and AuthInfo
func (r *Resolver) GetAuthInfo(ctx context.Context, opName string) (m *model.Model, af *model.AuthInfo, err error) {
	c := ginx.MustGetGinContext(ctx)
	m = c.MustGet("model").(*model.Model)
	au := c.MustGet("auth").(*jwt.Auth)
	
	//get client ip
	clientIp := c.ClientIP()
	if clientIp == "" {
		err = gm.NewError(http.StatusBadRequest, fmt.Errorf("can not resolve remote addr: %s", c.Request.RemoteAddr))
		return
	}
	
	af = &model.AuthInfo{}
	if err = au.GetAccessInfo(c.Request, af); err != nil {
		return
	}

	if code, e := m.CheckPermission(af, clientIp, opName); e != nil {
		err = gm.NewError(code, e)
	}
	return
}


func (r *queryResolver) AdminToken(ctx context.Context, name string, password string) (*jwt.AuthToken, error) {
	c, m, au, err := r.GetCtx(ctx, "Admin")
	if err != nil {
		return nil, err
	}
	
	vs, code, err := m.Admin(name, password)
	if err != nil {
		var message string
		switch code {
		case http.StatusUnauthorized:
			message = "用户名或密码错误"
		}
		return nil, gm.NewError(code, err, message)
	}

	var tk *jwt.AuthToken
	if tk, err = au.NewAuthToken(c.Request, vs); err != nil {
		return nil, gm.NewError(http.StatusInternalServerError, err)
	}
	return tk, nil
}

func (r *queryResolver) AccountID(ctx context.Context, mobile string) (gm.UUID, error) {
	_, m, _, err := r.GetCtx(ctx, "GetAccountIdByMobile")
	if err != nil {
		return gm.ZeroUUID(), err
	}

	id, code, err := m.GetAccountIdByMobile(mobile)
	if err != nil {
		return id, gm.NewError(code, err)
	}
	return id, nil
}

func (r *queryResolver) AuthToken(ctx context.Context, input model.AuthForm) (*jwt.AuthToken, error) {
	c, m, au, err := r.GetCtx(ctx, "Login")
	if err != nil {
		return nil, err
	}

	vs, code, err := m.Login(&input)
	if err != nil {
		var message string
		switch code {
		case http.StatusUnauthorized, http.StatusNotFound:
			message = "用户名或密码错误"
		case http.StatusPaymentRequired:
			message = "账户已到期"
		}
		return nil, gm.NewError(code, err, message)
	}

	var tk *jwt.AuthToken
	if tk, err = au.NewAuthToken(c.Request, vs); err != nil {
		return nil, gm.NewError(http.StatusInternalServerError, err)
	}
	return tk, nil
}

func (r *queryResolver) RefreshToken(ctx context.Context) (*jwt.AuthToken, error) {
	c, m, au, err := r.GetCtx(ctx, "RefreshToken")
	if err != nil {
		return nil, err
	}

	refresh := model.AuthInfo{}
	if err := au.GetRefreshInfo(c.Request, &refresh); err != nil {
		return nil, gm.NewError(http.StatusUnauthorized, err)
	}

	vs, code, err := m.RefreshToken(&refresh)
	if err != nil {
		var message string
		switch code {
		case http.StatusUnauthorized, http.StatusNotFound:
			message = "用户名或密码错误"
		case http.StatusPaymentRequired:
			message = "账户已到期"
		}
		return nil, gm.NewError(code, err, message)
	}

	var tk *jwt.AuthToken
	if tk, err = au.NewAuthToken(c.Request, vs); err != nil {
		return nil, gm.NewError(http.StatusInternalServerError, err)
	}
	return tk, nil
}

func (r *queryResolver) Account(ctx context.Context, id gm.UUID) (*model.Account, error) {
	m, af, err := r.GetAuthInfo(ctx, "GetAccountByID")
	if err != nil {
		return nil, err
	}

	//Get fields of the account
	ac, code, err := m.GetAccountByID(af, id)
	if err != nil {
		return nil, gm.NewError(code, err)
	}
	return ac, nil
}

func (r *queryResolver) User(ctx context.Context, id gm.UUID) (*model.User, error) {
	m, af, err := r.GetAuthInfo(ctx, "GetUserByID")
	if err != nil {
		return nil, err
	}

	//Get account fields
	usr, code, err := m.GetUserByID(af, id)
	return usr, gm.NewError(code, err)
}

func (r *queryResolver) Store(ctx context.Context, id gm.UUID) (*model.Store, error) {
	m, af, err := r.GetAuthInfo(ctx, "GetStoreByID")
	if err != nil {
		return nil, err
	}

	//Get account fields
	st, code, err := m.GetStoreByID(af, id)
	return st, gm.NewError(code, err)

	//Get users if necessary
	//if gm.HasField(ctx, "users") {
	//	if code, err = m.GetUsersOfStore(af, store); err != nil {
	//		c.Error(err)
	//		gm.AddError(ctx, "users", code)
	//	}
	//}
	//return st, nil
}

func (r *queryResolver) AccountsConnection(ctx context.Context, filter *model.AccountFilter, first int, after *gm.UUID) (*AccountsConnection, error) {
	m, af, err := r.GetAuthInfo(ctx, "SearchAccounts")
	if err != nil {
		return nil, err
	}

	cnt := 0
	accounts, code, err := m.SearchAccounts(af, filter, first, after, &cnt)
	if err != nil {
		return nil, gm.NewError(code, err)
	}

	return &AccountsConnection{
		TotalCount: cnt,
		Accounts:   accounts,
	}, nil
}

type accountResolver struct{ *Resolver }

func (r *accountResolver) Manager(ctx context.Context, obj *model.Account) (*model.User, error) {
	m, af, err := r.GetAuthInfo(ctx, "GetManagerOfAccount")
	if err != nil {
		return nil, err
	}

	manager, code, err := m.GetManagerOfAccount(af, obj)
	return manager, gm.NewError(code, err)
}

func (r *accountResolver) Stores(ctx context.Context, obj *model.Account, unscoped bool) ([]*model.Store, error) {
	m, af, err := r.GetAuthInfo(ctx,"GetStoresOfAccount")
	if err != nil {
		return nil, err
	}

	stores, code, err := m.GetStoresOfAccount(af, obj, unscoped)
	if err != nil {
		return nil, gm.NewError(code, err)
	}
	return stores, nil
}

type storeResolver struct{ *Resolver }

func (r *storeResolver) Users(ctx context.Context, obj *model.Store, unscoped bool) ([]*model.User, error) {
	m, af, err := r.GetAuthInfo(ctx, "GetUsersOfStore")
	if err != nil {
		return nil, err
	}

	users, code, err := m.GetUsersOfStore(af, obj, unscoped)
	if err != nil {
		return nil, gm.NewError(code, err)
	}
	return users, nil
}

type userResolver struct{ *Resolver }

func (r *userResolver) Store(ctx context.Context, obj *model.User) (*model.Store, error) {
	m, af, err := r.GetAuthInfo(ctx, "GetStoreOfUser")
	if err != nil {
		return nil, err
	}

	st, code, err := m.GetStoreOfUser(af, obj)
	return st, gm.NewError(code, err)
}

func (r *userResolver) Account(ctx context.Context, obj *model.User) (*model.Account, error) {
	m, af, err := r.GetAuthInfo(ctx, "GetAccountOfUser")
	if err != nil {
		return nil, err
	}

	ac, code, err := m.GetAccountOfUser(af, obj)
	return ac, gm.NewError(code, err)
}
