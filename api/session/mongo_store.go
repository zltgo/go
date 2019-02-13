package session

import (
	"net/http"
	"time"

	"github.com/zltgo/api/cache"
	"github.com/zltgo/api/ratelimit"
	"github.com/zltgo/reflectx"
	"gopkg.in/mgo.v2"
)

const (
	KeyUpdateTime = "ut"
	KeyMongoId    = "_id"
)

//Options for mongodb
type MongoOpts struct {
	Cookie

	// Url for Dial with mongodb, it must include database name.
	Url string

	// Collection name.
	C string `default:"sessions"`

	// Milliseconds to wait for W before timing out.
	// WMode and J is default.
	WTimeout int `default:"2000"`

	// TTL>0 means Max-Age attribute present and given in seconds for TTL index.
	// Default is 30 days.
	TTL int `default:"2592000"`

	// Rate limit options in seconds for create cookie cache per ip.
	// Default is nil, means no limit at all.
	// You'd better provide a rate limit options for this, because clients
	// may delete cookies themselves every time.
	// count of calls per ip are stored in LruMemStore, so LruMemStore must
	// big enough.
	RateSec []int
}

var _ Store = &MongoStore{}

type MongoStore struct {
	// Collection name.
	C  string
	Ms *mgo.Session
}

// NewCookieStore returns a CookieProvider that can create CookieStore.
func NewMongoProvider(opts MongoOpts, lmc *cache.LruMemCache) (*Provider, error) {
	reflectx.SetDefault(&opts)

	ms, err := mgo.Dial(opts.Url)
	if err != nil {
		return nil, err
	}

	ms.SetMode(mgo.Primary, true)
	ms.SetSafe(&mgo.Safe{WTimeout: opts.WTimeout})

	// token TTL index
	if opts.TTL > 0 {
		if err := ms.DB("").C(opts.C).EnsureIndex(mgo.Index{
			Key:         []string{KeyUpdateTime},
			ExpireAfter: time.Duration(opts.TTL) * time.Second,
			Background:  true,
		}); err != nil {
			return nil, err
		}
	}

	return &Provider{
		Cookie: opts.Cookie,
		store: &MongoStore{
			C:  opts.C,
			Ms: ms,
		},
		lmc:     lmc,
		RateOpt: ratelimit.SecOpts(opts.RateSec...),
	}, nil
}

// Get retruns data stored by id.
func (m *MongoStore) Get(r *http.Request, id string) (map[string]interface{}, error) {
	// get data in db
	ms := m.Ms.Clone()
	defer ms.Close()
	mc := ms.DB("").C(m.C)

	mp := make(map[string]interface{})
	if err := mc.FindId(id).One(&mp); err != nil {
		return nil, err
	}

	return mp, nil
}

// Save saves id and data to store.
func (m *MongoStore) Save(w http.ResponseWriter, id string, mp map[string]interface{}) error {
	ms := m.Ms.Clone()
	defer ms.Close()
	mc := ms.DB("").C(m.C)

	mp[KeyMongoId] = id
	mp[KeyUpdateTime] = time.Now()
	_, err := mc.UpsertId(id, mp)
	return err
}

// Remove removes id from store.
func (m *MongoStore) Remove(w http.ResponseWriter, id string) error {
	ms := m.Ms.Clone()
	defer ms.Close()
	mc := ms.DB("").C(m.C)

	return mc.RemoveId(id)
}

// IdKey returns a key for finding id in map[string]interface{}.
func (m *MongoStore) IdKey() string {
	return KeyMongoId
}
