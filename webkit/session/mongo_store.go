package session

import (
	"net/http"
	"time"

	"github.com/zltgo/api/cache"
	"github.com/zltgo/ratelimit"
	"github.com/zltgo/structure"
	"gopkg.in/mgo.v2"
)

//Options for mongodb
type MongoOpts struct {
	Cookie

	// Url for Dial with mongodb, it must include database name.
	Url string

	// Collection name.
	C string `default:"sessions"`

	// Milliseconds to wait for W before timing out.
	// WMode and J  is default.
	WTimeout int `default:"2000"`

	// TTL>0 means Max-Age attribute present and given in seconds for TTL index.
	// Default is 30 days.
	TTL int `default:"2592000"`
	// TTL index name.
	KeyUpdateTime string `default:"ut"`

	// Rate limit options in seconds for create cookie cache per ip.
	// Default is nil, means no limit at all.
	// You'd better provide a rate limit options for this, because clients
	// may delete cookies themselves every time.
	// count of calls per ip are stored in LruMemStore, so LruMemStore must
	// big enough.
	RateLimit []int

	// Used for finding id in values.
	KeyMongoId string `default:"_id"`
}

var _ Store = &MongoStore{}

type MongoStore struct {
	// Collection name.
	C             string
	KeyUpdateTime string
	KeyMongoId    string
	Ms            *mgo.Session
}

// NewCookieStore returns a CookieProvider that can create CookieStore.
func NewMongoProvider(opts MongoOpts, lmc *cache.LruMemCache) (*Provider, error) {
	if err := structure.SetDefault(&opts); err != nil {
		panic(err)
	}

	ms, err := mgo.Dial(opts.Url)
	if err != nil {
		return nil, err
	}

	ms.SetMode(mgo.Primary, true)
	ms.SetSafe(&mgo.Safe{WTimeout: opts.WTimeout})

	// token TTL index
	if opts.TTL > 0 {
		if err := ms.DB("").C(opts.C).EnsureIndex(mgo.Index{
			Key:         []string{opts.KeyUpdateTime},
			ExpireAfter: time.Duration(opts.TTL) * time.Second,
			Background:  true,
		}); err != nil {
			return nil, err
		}
	}

	store := &MongoStore{
		C:             opts.C,
		KeyUpdateTime: opts.KeyUpdateTime,
		KeyMongoId:    opts.KeyMongoId,
		Ms:            ms,
	}
	return &Provider{
		Cookie:          opts.Cookie,
		store:           store,
		lmc:             lmc,
		cookieRatelimit: ratelimit.SecOpts(opts.RateLimit...),
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
// Save should removes data from store If id is empty or data is nil.
func (m *MongoStore) Save(w http.ResponseWriter, id string, mp map[string]interface{}) error {
	ms := m.Ms.Clone()
	defer ms.Close()
	mc := ms.DB("").C(m.C)
	if len(mp) == 0 || id == "" {
		err := mc.RemoveId(id)
		if err == mgo.ErrNotFound {
			return nil
		}
		return err
	}

	// save data to db
	mp[m.KeyMongoId] = id
	mp[m.KeyUpdateTime] = time.Now()
	_, err := mc.UpsertId(id, mp)
	return err
}

// IdKey returns a key for finding id in map[string]interface{}.
func (m *MongoStore) IdKey() string {
	return m.KeyMongoId
}
