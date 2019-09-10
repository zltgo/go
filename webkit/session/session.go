package session

import (
	"errors"
	"net/http"
	"strings"

	"github.com/zltgo/api"
	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/jwt"
	"github.com/zltgo/ratelimit"
	"github.com/zltgo/structure"
	"gopkg.in/mgo.v2/bson"
)

var (
	ErrOverrun         = errors.New("session: too much call")
	ErrIdMismatch      = errors.New("session:  id mismatched")
	ErrIp              = errors.New("session: remote ip is invalid")
	ErrNoConfig        = errors.New("session: ratelimit configuration not found")
	PrefixRatelimitKey = "rl:"
)

// Session stores the values and optional configuration for a session.
type Session interface {
	// Return the session Id.
	Id() string

	//Clear all values.
	Clear()

	cache.Values
}

var _ Session = session{}

type session struct {
	id string
	cache.Values
}

func (m session) Id() string {
	return m.id
}

//Clear all values.
func (m session) Clear() {
	m.Assign(nil)
}

type Store interface {
	// Get retruns data stored by id.
	Get(r *http.Request, id string) (map[string]interface{}, error)
	// Save saves id and data to store.
	// Save should removes data from store If id is empty or data is nil.
	Save(w http.ResponseWriter, id string, mp map[string]interface{}) error

	// IdKey returns a key for finding id in map[string]interface{}.
	IdKey() string
}

type Provider struct {
	Cookie
	store Store
	lmc   *cache.LruMemCache

	//rate limit options for create cookie per client
	cookieRatelimit []ratelimit.Option
	urlRatelimit    map[string][]ratelimit.Option
}

func NewProvider(store Store, lmc *cache.LruMemCache) *Provider {
	var ck Cookie
	if err := structure.SetDefault(&ck); err != nil {
		panic(err)
	}
	return &Provider{
		Cookie: ck,
		store:  store,
		lmc:    lmc,
	}
}

// Rate limit options for create cookie per ip
// It is not thread-safe.
func (m *Provider) SetCookieRatelimit(rateOpt []int) {
	m.cookieRatelimit = ratelimit.SecOpts(rateOpt...)
}

// Provider can used to cache  RateStats for a specific http.Request.
// rateCfg map configurations for url prefix.
// Ensure evey url has a ratelimit configuration, otherwise Check function
// will return ErrNoConfig.
//
// Example:
// rateCfg{"GET:/api", []int{100, 1}} means "GET:/api" group are limited to
// 100 calls per second by the same client.
// "ANY:/api/usr" means all the methods with "/api/usr" prefix are reckon in.
// It is not thread-safe.Call this function at the beginning.
func (m *Provider) SetUrlRatelimit(rateCfg map[string][]int) {
	// convert rateCfg to rateOpts
	m.urlRatelimit = make(map[string][]ratelimit.Option, len(rateCfg))
	for k, v := range rateCfg {
		m.urlRatelimit[k] = ratelimit.SecOpts(v...)
	}
}

// Wrap a session middleware.
func (m *Provider) SessionMware(next api.Handler) api.Handler {
	return func(ctx *api.Context) {
		se, err := m.GetSession(ctx.Request)
		if se == nil {
			// too much call to create session.
			if err == ErrOverrun {
				ctx.Render(http.StatusTooManyRequests, err)
			} else {
				ctx.Render(http.StatusInternalServerError, err)
			}
			return
		}
		// just log error if err is not nil
		ctx.Error(err)

		//session middleware
		ctx.MapTo(se, (*Session)(nil))
		// save session before render
		ctx.Defer(func() {
			ctx.Error(m.SaveSession(ctx.Writer(), se))
		})

		//rate limit
		if len(m.urlRatelimit) > 0 {
			if err = CheckUrlRate(ctx.Request, se, m.urlRatelimit); err != nil {
				// DOH! Over limit!
				ctx.Render(http.StatusTooManyRequests, err)
				return
			}
		}

		next(ctx)
		return
	}
}

type Id string
type RatelimitId string

// Get  returns a cached session.
// Get should return nil session if ErrOverrun occurred.
// Usually you should simply create a new Session if an error occurred.
func (m *Provider) GetSession(r *http.Request) (Session, error) {
	// get id in cookie
	id, err := m.GetCookie(r)
	// http.ErrNoCookie
	if err != nil {
		return m.newSession(r, err)
	}

	// find values in LruMemCache
	vs, err := m.lmc.Get(Id(id))
	if err == nil {
		// It is necessary to check id, because clients can use a rand id
		// or something like ip to hit LruMemCache.
		if id != vs.GetString(m.store.IdKey()) {
			return m.newSession(r, ErrIdMismatch)
		}
		return session{id, vs}, nil
	}

	// get data from store
	mp, err := m.store.Get(r, id)
	if err != nil {
		return m.newSession(r, err)
	}

	// cache the values in LruMemCache
	vs.Assign(mp)
	return session{id, vs}, nil
}

// Save should persist session to the underlying store implementation.
// If session.Items() retruns zero, session should removed from the Store.
func (m *Provider) SaveSession(w http.ResponseWriter, se Session) (err error) {
	se.IfModified(func(mp map[string]interface{}) {
		// save session id in cookie
		if len(mp) == 0 || se.Id() == "" {
			m.DelCookie(w)
			// Do not delete from LruMemCache, otherwise clients can
			// reuse id and token themselves.
			//m.lmc.Del(se.Id())
		} else {
			m.SetCookie(w, se.Id())
		}
		// save values
		err = m.store.Save(w, se.Id(), mp)
	})
	return
}

// Create a new session, ratelimited if Options.Ratelimit is not nil.
func (m *Provider) newSession(r *http.Request, err error) (Session, error) {
	// rate limit for create cookie cache
	if len(m.cookieRatelimit) > 0 {
		// get client ip
		ip, err := getIp(r)
		if err != nil || ip == "" {
			return nil, ErrIp
		}

		//get ratelimiter in LruMemCache
		rl := m.lmc.Getsert(RatelimitId(ip), func() interface{} {
			return ratelimit.New(m.cookieRatelimit)
		})

		if rl.(*ratelimit.RateLimiter).Limit() {
			return nil, ErrOverrun
		}
	}

	// create a new session
	id := bson.NewObjectId().Hex()
	vs := cache.NewValues(jwt.Claim{m.store.IdKey(): id})
	m.lmc.Set(Id(id), vs)
	return session{id, vs}, err
}

// Not thread-safe to modify urlRatelimit when checking.
func CheckUrlRate(r *http.Request, se Session, rateOpts map[string][]ratelimit.Option) error {
	// find url key and ratelimit configuration in mp.
	// ensure evey url has a ratelimit configuration.
	url1 := r.Method + ":" + r.URL.Path
	url2 := "ANY:" + r.URL.Path

	flag := 0
	var find bool
	for k, opts := range rateOpts {
		if strings.HasPrefix(url1, k) || strings.HasPrefix(url2, k) {
			find = true
			rl := se.Getsert(PrefixRatelimitKey+k, func() interface{} {
				return ratelimit.New(opts)
			})

			if rl.(*ratelimit.RateLimiter).Limit() {
				flag++
			}
		}
	}

	if flag > 0 {
		return ErrOverrun
	}

	if !find {
		return ErrNoConfig
	}
	return nil
}
