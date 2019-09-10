package session

import (
	"net/http"

	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/jwt"
	"github.com/zltgo/ratelimit"
	"github.com/zltgo/structure"
)

var (
	DefaultCookiePd = NewCookieProvider(CookieOpts{}, cache.DefaultLruMemCache)
)

type CookieOpts struct {
	Cookie

	// token name in cookie
	TkName string `default:"_session_tk"`

	// Used for NewTokenOp.
	// If HashKey or BlockKey is empty, it will be created by RandString(16).
	HashKey  string
	BlockKey string

	// Rate limit options in seconds for create cookie cache per ip.
	// Default is nil, means no limit at all.
	// You'd better provide a rate limit options for this, because clients
	// may delete cookies themselves every time.
	// count of calls per ip are stored in LruMemStore, so LruMemStore must
	// big enough.
	RateLimit []int

	// Used for finding id in values.
	KeyCookieId string `default:"_cid"`
}

var _ Store = &CookieStore{}

type CookieStore struct {
	Cookie
	KeyCookieId string
	Tp          jwt.TokenOp
}

// NewCookieStore returns a CookieProvider that can create CookieStore.
func NewCookieProvider(opts CookieOpts, lmc *cache.LruMemCache) *Provider {
	if err := structure.SetDefault(&opts); err != nil {
		panic(err)
	}
	if len(opts.HashKey) == 0 {
		opts.HashKey = jwt.RandString(16)
	}
	if len(opts.BlockKey) == 0 {
		opts.BlockKey = jwt.RandString(16)
	}
	ck := opts.Cookie
	ck.Name = opts.TkName
	store := &CookieStore{
		Cookie:      ck,
		KeyCookieId: opts.KeyCookieId,
		Tp:          jwt.NewTokenOp(opts.MaxAge, []byte(opts.HashKey), []byte(opts.BlockKey)),
	}
	return &Provider{
		Cookie:          opts.Cookie,
		store:           store,
		lmc:             lmc,
		cookieRatelimit: ratelimit.SecOpts(opts.RateLimit...),
	}
}

// Get retruns data stored by id.
func (m *CookieStore) Get(r *http.Request, id string) (map[string]interface{}, error) {
	// get token in cookie
	tk, err := m.GetCookie(r)
	// http.ErrNoCookie
	if err != nil {
		return nil, err
	}
	// parse token
	claim, err := m.Tp.ParseToken(tk)
	if err != nil {
		return nil, err
	}
	// must check id in claim, otherwise client can reuse the same
	// token(but use a new id) in every request.
	if id != claim.GetString(m.KeyCookieId) {
		return nil, ErrIdMismatch
	}

	return claim, nil
}

// Save saves id and data to store.
// Save should removes data from store If id is empty or data is nil.
func (m *CookieStore) Save(w http.ResponseWriter, id string, mp map[string]interface{}) error {
	if len(mp) == 0 || id == "" {
		// delete cookie.
		m.DelCookie(w)
		return nil
	}
	// encode to token string
	var tk string
	mp[m.KeyCookieId] = id
	tk, err := m.Tp.CreateToken(mp)
	if err == nil {
		m.SetCookie(w, tk)
	}
	return err
}

// IdKey returns a key for finding id in map[string]interface{}.
func (m *CookieStore) IdKey() string {
	return m.KeyCookieId
}
