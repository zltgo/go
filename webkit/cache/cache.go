package cache

import (
	"errors"
	"github.com/zltgo/webkit/cache/lru"
	"github.com/zltgo/webkit/cache/singleflight"
)

var (
	ErrNotExist = errors.New("cache: provided id does not exist")
)

//type Cache interface {
//	// Get values  from cache by  id. If provided id does not exist in cache,
//	// usually you should simply create a new values.
//	Get(id interface{}) (interface{}, error)
//
//	// Save id and values to the cache.
//	// It replaces any existing values.
//	Set(id interface{}, v interface{}) error
//
//	// If id does not exist, create a new value by fn.
//	Getsert(id interface{}, fn func() interface{}) (interface{}, error)
//
//	// Remove values form cache by id.
//	Remove(id interface{}) error
//
//	// Clear purges all stored items from the cache.
//	Clear()
//}

type Cache struct {
	group *singleflight.Group
	lc    *lru.Cache
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{}
type GetFunc func(Key) (interface{}, error)

type Evictor interface {
	OnEvicted(Key)
}

func New(maxEntries int) Cache {
	lc := lru.New(maxEntries)

	// if value implement the Evictor interface, call it on evicted.
	lc.OnEvicted = func(k lru.Key, v interface{}) {
		if evictor, ok := v.(Evictor); ok {
			evictor.OnEvicted(k)
		}
	}

	return Cache{
		group: &singleflight.Group{},
		lc:    lc,
	}
}

// Get looks up a key's value from the cache.
// if key does not exist, getter will be called as single flight.
func (c Cache) Get(k Key, getter GetFunc) (interface{}, error) {
	if v, ok := c.lc.Get(k); ok {
		return v, nil
	}
	if getter == nil {
		return nil, ErrNotExist
	}

	//look up value by GetFunc
	value, err := c.group.Do(k, func() (interface{}, error) {
		// Check the cache again because singleflight can only dedup calls
		// that overlap concurrently.  It's possible for 2 concurrent
		// requests to miss the cache, resulting in 2 load() calls.  An
		// unfortunate goroutine scheduling would result in this callback
		// being run twice, serially.  If we don't check the cache again,
		// cache.nbytes would be incremented below even though there will
		// be only one entry for this key.
		//
		// Consider the following serialized event ordering for two
		// goroutines in which this callback gets called twice for the
		// same key:
		// 1: Get("key")
		// 2: Get("key")
		// 1: lookupCache("key")
		// 2: lookupCache("key")
		// 1: loadGroup.Do("key", fn)
		// 1: fn()
		// 2: loadGroup.Do("key", fn)
		// 2: fn()
		if v, ok := c.lc.Get(k); ok {
			return v, nil
		}
		v, e := getter(k)
		if e == nil {
			//add value to cache in case of succeed.
			c.lc.Add(k, v)
		}
		return v, e
	})

	return value, err
}

func (c Cache) Remove(k Key) {
	c.lc.Remove(k)
}

func (c Cache) Clear() {
	c.lc.Clear()
}

func (c Cache) Stats() lru.Stats {
	return c.lc.GetStats()
}

// Save id and values to the cache.
// It replaces any existing values.
// Just Remove the key if DB updated.
// Consider the following serialized event ordering for two
// goroutines in which this callback sets called twice for the
// same key:
// 1: UpdateDB("key", "V1")
// 2: UpdateDB("key", "V2")
// 2: Set("key", "V2")
// 1: Set("key", "V1")
// So we doesn't need a Set function, use Get instead
//func (c Cache) Set(k Key, v interface{}) {
//c.lc.Add(k, v)
//}
