package session

import (
	"errors"
	"net/http"

	"gopkg.in/mgo.v2/bson"

	"github.com/zltgo/api"
	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/ratelimit"
	"github.com/zltgo/reflectx"
	"github.com/zltgo/reflectx/values"
)

var (
	ErrOverrun    = errors.New("session: too much call")
	ErrIdMismatch = errors.New("session: id mismatched")
	ErrIp         = errors.New("session: remote ip is invalid")
)

type Session struct {
	Id string
	values.Values
}

type Store interface {
	// Get retruns data stored by id.
	// map must be nil in case of any error.
	Get(r *http.Request, id string) (map[string]interface{}, error)
	// Save saves id and data to store.
	Save(w http.ResponseWriter, id string, data map[string]interface{}) error
	// Remove removes id from store.
	Remove(w http.ResponseWriter, id string) error

	// IdKey returns a key for finding id in map[string]interface{}.
	IdKey() string
}

type Provider struct {
	Cookie
	store Store
	lmc   *cache.LruMemCache

	// Rate limit options in seconds for create cookie cache per ip.
	// Default is nil, means no limit at all.
	// You'd better provide a rate limit options for this, because clients
	// may delete cookies themselves every time.
	// count of calls per ip are stored in LruMemStore, so LruMemStore must
	// big enough.
	RateOpt []ratelimit.Option
}

func NewProvider(store Store, lmc *cache.LruMemCache) *Provider {
	var ck Cookie
	reflectx.SetDefault(&ck)

	if store == nil {
		store = NoStore{}
	}

	return &Provider{
		Cookie: ck,
		store:  store,
		lmc:    lmc,
	}
}

// Wrap a session middleware.
func (p *Provider) SessionHandler(ctx *api.Context) {
	// se will be a new session in case of ErrNotFound
	se, err := p.GetSession(ctx.Request)
	if se == nil {
		// too much call to create session.
		if err == ErrOverrun {
			ctx.Reply(http.StatusTooManyRequests, err)
		} else {
			ctx.Reply(http.StatusInternalServerError, err)
		}
		return
	}
	// just log error if err is not nil
	ctx.Error(err)

	//save session to context
	ctx.Map(se)
	// save session before render, LockGuard is necessay.
	// for example:
	// a->encode
	// b->update->encode->save
	// a->save
	ctx.Writer.Defer(func() {
		ctx.Error(p.SaveSession(ctx.Writer, se))
	})
	return
}

type SessionId string
type RatelimitId string

// Get returns a cached session.
// Get should return nil session if ErrOverrun occurred.
// Usually you should simply create a new Session if an error occurred.
func (p *Provider) GetSession(r *http.Request) (*Session, error) {
	// get id in cookie
	id, err := p.GetCookie(r)
	// create a new session in case of http.ErrNoCookie
	if err != nil {
		return p.newSession(r, err)
	}

	// find values in LruMemCache
	sm, _ := p.lmc.Getsert(SessionId(id), func() interface{} {
		// get data from store
		var data map[string]interface{}
		data, err = p.store.Get(r, id)
		if err != nil || data == nil {
			return nil
		}
		return values.NewSafeMap("json", data)
	})

	if sm == nil {
		return p.newSession(r, err)
	}

	vs := sm.(*values.SafeMap)
	// It is necessary to check id, because clients can use a rand id to hit LruMemCache.
	// Take CookieStore for instance, the client can reuse the same token(but use a new id) in every request.
	if id != vs.ValueOf(p.store.IdKey()).String() {
		return p.newSession(r, ErrIdMismatch)
	}
	return &Session{id, vs}, err
}

// save session before render, LockGuard is necessay.
// for example:
// a->encode
// b->update->encode->save
// a->save
func (p *Provider) SaveSession(w http.ResponseWriter, se *Session) (err error) {
	//RemoveAll is better if you want to remove the session from store.
	if se.Id == "" {
		se.RemoveAll()
		p.DelCookie(w)
		return nil
	}

	sm := se.Values.(*values.SafeMap)
	sm.LockGuard(func(data map[string]interface{}) {
		//remove from store if data is empty
		if len(data) == 0 {
			// Do not delete from LruMemCache, otherwise clients can
			// reuse id and token themselves.
			// m.lmc.Del(se.Id())
			p.DelCookie(w)
			err = p.store.Remove(w, se.Id)
		} else {
			p.SetCookie(w, se.Id)
			err = p.store.Save(w, se.Id, data)
		}
	})
	return err
}

// Create a new session, ratelimited if Options.Ratelimit is not nil.
// Note that newSession is not thread-safe for the same client.
func (p *Provider) newSession(r *http.Request, err error) (*Session, error) {
	// rate limit for create cookie
	if len(p.RateOpt) > 0 {
		// get client ip
		ip, err := getIp(r)
		if err != nil || ip == "" {
			return nil, ErrIp
		}

		//get ratelimiter in LruMemCache
		rl, _ := p.lmc.Getsert(RatelimitId(ip), func() interface{} {
			return ratelimit.New(p.RateOpt)
		})

		if rl.(*ratelimit.RateLimiter).Limit() {
			return nil, ErrOverrun
		}
	}
	//create id and an empty safemap
	id := bson.NewObjectId().Hex()
	data := make(map[string]interface{})
	data[p.store.IdKey()] = id
	sm := values.NewSafeMap("json", data)

	// cache the session to LruMemCache
	p.lmc.Set(SessionId(id), sm)
	return &Session{id, sm}, err
}

//****************** nostore*********************************
// NoStore can used for caching session just cached in LruMemCache
type NoStore struct{}

func (n NoStore) Get(r *http.Request, id string) (map[string]interface{}, error) {
	return nil, errors.New("session dose not stored")
}
func (n NoStore) Save(w http.ResponseWriter, id string, data map[string]interface{}) error {
	return nil
}
func (n NoStore) Remove(w http.ResponseWriter, id string) error {
	return nil
}
func (n NoStore) IdKey() string {
	return "_id"
}
