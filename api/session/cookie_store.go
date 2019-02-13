package session

import (
	"net/http"

	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/jwt"
	"github.com/zltgo/api/ratelimit"
	"github.com/zltgo/reflectx"
)

type CookieOpts struct {
	Cookie

	// token name in cookie
	TkName string `default:"_session_tk"`

	// hashKey is used to authenticate values using HMAC.
	// It is recommended to use a key with 16 , 32, 48 or 64 bytes
	// to select sha1, sha256, sha384, or sha512.
	// If hashKey is nil, it will be created by RandBytes(16).
	HashKey string

	// The blockKey argument should be the AES key,either 16, 24, or 32 bytes to select
	// AES-128, AES-192, or AES-256.
	BlockKey string

	// Rate limit options in seconds for create cookie cache per ip.
	// Default is nil, means no limit at all.
	// You'd better provide a rate limit options for this, because clients
	// may delete cookies themselves every time.
	// count of calls per ip are stored in LruMemStore, so LruMemStore must
	// big enough.
	RateSec []int

	// Used for finding id in values.
	IdKey string `default:"_cid"`
}

var _ Store = &CookieStore{}

type CookieStore struct {
	Cookie
	idKey    string
	tkParser jwt.Parser
}

// The blockKey argument should be the AES key,either 16, 24, or 32 bytes to select
// AES-128, AES-192, or AES-256.
// hashKey is used to authenticate values using HMAC.
// It is recommended to use a key with 16 , 32, 48 or 64 bytes
// to select sha1, sha256, sha384, or sha512.
// If hashKey is nil, it will be created by RandBytes(16).
func NewCookieStore(ck Cookie, idKey, hashKey, blockKey string) *CookieStore {
	reflectx.SetDefault(&ck)
	if idKey == "" {
		idKey = "_cid"
	}
	if len(hashKey) == 0 {
		hashKey = jwt.RandString(16)
	}
	if len(blockKey) == 0 {
		blockKey = jwt.RandString(16)
	}
	return &CookieStore{
		Cookie:   ck,
		idKey:    idKey,
		tkParser: jwt.NewParser(ck.MaxAge, []byte(hashKey), []byte(blockKey)),
	}
}

// NewCookieStore returns a CookieProvider that can create CookieStore.
func NewCookieProvider(opts CookieOpts, lmc *cache.LruMemCache) *Provider {
	reflectx.SetDefault(&opts)
	ck := opts.Cookie
	ck.Name = opts.TkName

	return &Provider{
		Cookie:  opts.Cookie,
		store:   NewCookieStore(ck, opts.IdKey, opts.HashKey, opts.BlockKey),
		lmc:     lmc,
		RateOpt: ratelimit.SecOpts(opts.RateSec...),
	}
}

// Get retruns data stored by id.
func (cs *CookieStore) Get(r *http.Request, id string) (map[string]interface{}, error) {
	// get token in cookie
	tk, err := cs.GetCookie(r)
	// http.ErrNoCookie
	if err != nil {
		return nil, err
	}
	// parse token
	return cs.tkParser.ParseToken(tk)
}

// Save saves id and data to store.
func (cs *CookieStore) Save(w http.ResponseWriter, id string, mp map[string]interface{}) error {
	// encode to token string
	var tk string
	mp[cs.idKey] = id
	tk, err := cs.tkParser.CreateToken(mp)
	if err != nil {
		return err
	}
	cs.SetCookie(w, tk)
	return nil
}

// Remove removes id from store.
func (cs *CookieStore) Remove(w http.ResponseWriter, id string) error {
	cs.DelCookie(w)
	return nil
}

// IdKey returns a key for finding id in map[string]interface{}.
func (cs *CookieStore) IdKey() string {
	return cs.idKey
}
