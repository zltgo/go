package jwt

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/zltgo/api"
	"github.com/zltgo/reflectx"
	"github.com/zltgo/reflectx/values"
)

var (
	//errors
	ErrNoUid = errors.New("jwt: uid is not present in token")

	//the key stored in map for userId
	KeyUserId    = "_uid"
	KeyAgentHash = "_agh"
	KeyGrantType = "_grt"
	TokenKeys    = []string{"ACCESS-TOKEN", "REFRESH-TOKEN"}
)

type UID string

// AuthMware provides a Json-Web-Token authentication implementation. On failure, a 401 HTTP response
// is returned. On success, the wrapped middleware is called, and the userId is made available as
// yourhandler(uid UID), note that uid can convert to string.
// Users can get a token by posting a json request to LoginHandler. The token then needs to be passed in
// the Authentication header. Example: ACCESS-TOKEN: XXX_TOKEN_XXX
type Auth struct {
	// access is used to create a AccessToken.
	// Optional, default is NewParser(1800, nil, nil).
	access Parser

	// refresh is used to create a RefreshToken.
	// Optional, default is NewParser(0, nil, nil).
	refresh Parser

	// Interface for extracting a token from an HTTP request.
	// Optional, default is HeaderGetter([]string{TokenKey})
	tg TokenGetter
}

type AuthOpts struct {
	//access token life-time, default is half an hour
	AccessMaxAge  int `default:"1800"`
	RefreshMaxAge int

	// 16, 32 or 64 len
	HashKey   string
	BlockKey  string
	TokenKeys []string `default:"ACCESS-TOKEN,REFRESH-TOKEN"`
}

func NewAuthByOpts(opts AuthOpts) *Auth {
	reflectx.SetDefault(&opts)

	if len(opts.HashKey) == 0 {
		opts.HashKey = RandString(16)
	}
	if len(opts.BlockKey) == 0 {
		opts.BlockKey = RandString(16)
	}
	return &Auth{
		access:  NewParser(opts.AccessMaxAge, []byte(opts.HashKey), []byte(opts.BlockKey)),
		refresh: NewParser(opts.RefreshMaxAge, []byte(opts.HashKey), []byte(opts.BlockKey)),
		tg:      HeaderGetter(opts.TokenKeys),
	}
}

func NewAuth(access Parser, refresh Parser, tg TokenGetter) *Auth {
	if access == nil {
		access = NewParser(1800, nil, nil)
	}

	if refresh == nil {
		refresh = NewParser(0, nil, nil)
	}

	if tg == nil {
		tg = HeaderGetter(TokenKeys)
	}

	return &Auth{
		access:  access,
		refresh: refresh,
		tg:      tg,
	}
}

type AuthToken struct {
	AccessToken  string
	RefreshToken string
	MaxAge       int //lifetime in seconds
}

// loginFunc is a callback function that should perform the authentication of the user.
// On success, loginFunc returns (200, uid).On failure, loginFunc returns (errCode, errString).
// for exmaple:
// type LoginForm struct {
//	 Name     string `binding:"alphanum,min=5,max=32"`
//	 Password string `binding:"alphanum,min=5,max=32"`
// }
// func login(lf LoginForm) (code int,  uid string) {
//	if(find name, password in db) {
//		return 200, uid
//	}else {
//		return 400, "name or password error"
//	}
// }
func (m *Auth) LoginHandler(loginFunc interface{}) api.Handler {
	//check loginFunc
	fV := reflect.ValueOf(loginFunc)
	fT := fV.Type()
	if fT.NumOut() != 2 || fT.Out(0).Kind() != reflect.Int || fT.Out(1).Kind() != reflect.String {
		panic("loginFunc must return (int, string)")
	}

	return func(ctx *api.Context) {
		rv, err := ctx.Invoke(loginFunc)

		if err != nil {
			// Bind error, usually because validate failed.
			ctx.Reply(http.StatusBadRequest, err)
			return
		}

		//check return code of loginFunc
		code := rv[0].Int()
		if code != http.StatusOK {
			ctx.Reply(int(code), errors.New(rv[1].String()))
			return
		}

		//create access token
		authToken, err := m.NewAuthToken(rv[1].Interface().(string), ctx.Request)
		if err != nil {
			ctx.Reply(http.StatusInternalServerError, err)
			return
		}

		ctx.Reply(http.StatusOK, authToken)
		return
	}
}

// Create a new auth token
func (m *Auth) NewAuthToken(uid string, r *http.Request) (*AuthToken, error) {
	vs := values.JsonMap{}
	vs.Set(KeyUserId, uid)
	vs.Set(KeyAgentHash, Hash64(r.UserAgent()))
	vs.Set(KeyGrantType, "access")
	acc, err := m.access.CreateToken(vs)
	if err != nil {
		return nil, errors.New("jwt: " + err.Error())
	}

	vs.Set(KeyGrantType, "refresh")
	ref, err := m.refresh.CreateToken(vs)
	if err != nil {
		return nil, errors.New("jwt: " + err.Error())
	}

	return &AuthToken{
		AccessToken:  acc,
		RefreshToken: ref,
		MaxAge:       m.access.MaxAge(),
	}, nil
}

// RefreshHandler can be used to refresh a token.The RefreshToken needs to be passed in
// the Authentication header. Example: r.Header.Set("REFRESH-TOKEN", "your-refresh-token-got-by-login")
func (m *Auth) RefreshHandler(ctx *api.Context) {
	//get refresh token
	ref, err := m.tg.GetToken(ctx.Request)
	if err != nil {
		ctx.Reply(http.StatusUnauthorized, err)
		return
	}
	//parse token
	var vs values.JsonMap
	vs, err = m.refresh.ParseToken(ref)
	if err != nil {
		ctx.Reply(http.StatusUnauthorized, err)
		return
	}

	// validate grant type and user agent
	grt := vs.ValueOf(KeyGrantType).String()
	if grt != "refresh" {
		ctx.Reply(http.StatusUnauthorized, errors.New("jwt: grant type mismatched: expect refresh, got "+grt))
		return
	}
	if vs.ValueOf(KeyAgentHash).String() != Hash64(ctx.Request.UserAgent()) {
		ctx.Reply(http.StatusUnauthorized, errors.New("jwt: user agent mismatched: "+ctx.Request.UserAgent()))
		return
	}

	//create access token
	vs.Set(KeyGrantType, "access")
	acc, err := m.access.CreateToken(vs)
	if err != nil {
		ctx.Reply(http.StatusInternalServerError, err)
		return
	}

	ctx.Reply(http.StatusOK, &AuthToken{
		AccessToken:  acc,
		RefreshToken: ref,
		MaxAge:       m.access.MaxAge(),
	})
	return
}

// AuthHandler provides a Json-Web-Token authentication implementation. On failure, a 401 HTTP response
// is returned. On success, the wrapped middleware is called, and the userId is made available as
// yourhandler(uid UID), note that uid can convert to string.
// Users can get a token by posting a json request to LoginHandler. The token then needs to be passed in
// the Authentication header. Example: r.Header.Set("ACCESS-TOKEN", "your-access-token-got-by-login")
func (m *Auth) AuthHandler(ctx *api.Context) {
	uid, err := m.AuthFunc(ctx.Request)
	if err != nil {
		ctx.Reply(http.StatusUnauthorized, err)
		return
	}
	ctx.Map(UID(uid))
	return
}

// Get uid from token in http.Request.
func (m *Auth) AuthFunc(r *http.Request) (string, error) {
	//get access token
	token, err := m.tg.GetToken(r)
	if err != nil {
		return "", err
	}
	//parse token
	var vs values.JsonMap
	vs, err = m.refresh.ParseToken(token)
	if err != nil {
		return "", err
	}

	// validate grant type and user agent
	grt := vs.ValueOf(KeyGrantType).String()
	if grt != "access" {
		return "", errors.New("jwt: grant type mismatched: expect access, got " + grt)
	}
	if vs.ValueOf(KeyAgentHash).String() != Hash64(r.UserAgent()) {
		return "", errors.New("jwt: user agent mismatched: " + r.UserAgent())
	}

	uid := vs.ValueOf(KeyUserId).String()
	if uid == "" {
		return "", ErrNoUid
	}

	return uid, nil
}
