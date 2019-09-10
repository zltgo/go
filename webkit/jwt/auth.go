package jwt

import (
	"net/http"

	"github.com/zltgo/reflectx"
)

// AuthMware provides a Json-Web-Token authentication implementation.
// The token then needs to be passed inthe Authentication header.
// Example: ACCESS-TOKEN: XXX_TOKEN_XXX
type Auth struct {
	// access is used to create a AccessToken.
	// Optional, default is NewParser(3600, nil, nil).
	access Token

	// refresh is used to create a RefreshToken.
	// Optional, default is NewParser(0, nil, nil).
	refresh Token
}

type AuthOpts struct {
	//access token life-time, default is an hour
	AccessMaxAge  int `default:"3600"`
	RefreshMaxAge int

	// 16, 32 or 64 len
	HashKey  string
	BlockKey string

	AccessHeaders  []string `default:"ACCESS-TOKEN"`
	RefreshHeaders []string `default:"REFRESH-TOKEN"`
}

func NewAuth(opts AuthOpts) *Auth {
	reflectx.SetDefault(&opts)
	access := Token{
		TokenGetter: HeaderGetter(opts.AccessHeaders),
		Parser:      NewParser(opts.AccessMaxAge, []byte(opts.HashKey), []byte(opts.BlockKey)),
		GrantType:   "access",
	}
	refresh := Token{
		TokenGetter: HeaderGetter(opts.RefreshHeaders),
		Parser:      NewParser(opts.RefreshMaxAge, []byte(opts.HashKey), []byte(opts.BlockKey)),
		GrantType:   "refresh",
	}
	return &Auth{
		access:  access,
		refresh: refresh,
	}
}

type AuthToken struct {
	AccessToken  string
	RefreshToken string
	MaxAge       int //lifetime in seconds
}

// Create a new auth token.
// usrInfo can be any type you want stored in token.
func (m *Auth) NewAuthToken(r *http.Request, meta interface{}) (*AuthToken, error) {
	at, err := m.access.EncodeValues(r, meta)
	if err != nil {
		return nil, err
	}

	rt, err := m.refresh.EncodeValues(r, meta)
	if err != nil {
		return nil, err
	}

	return &AuthToken{
		AccessToken:  at,
		RefreshToken: rt,
		MaxAge:       m.access.MaxAge(),
	}, nil
}

// Get values from token in http.Request.
func (m *Auth) GetAccessInfo(r *http.Request, pMeta interface{}) error {
	return m.access.DecodeValues(r, pMeta)
}

// Get values from token in http.Request.
func (m *Auth) GetRefreshInfo(r *http.Request, pMeta interface{}) error {
	return m.refresh.DecodeValues(r, pMeta)
}
