package cache

import (
	"sync"

	"github.com/golang/groupcache/lru"
)

var _ Cache = &LruMemCache{}

type LruMemCache struct {
	lc     *lru.Cache
	mu     sync.Mutex
	nhit   int64 // number of hit
	nget   int64 // number of get
	nevict int64 // number of evictions
}

// LruStats are returned by stats accessors on Group.
type LruStats struct {
	Items     int
	Gets      int64
	Hits      int64
	Evictions int64
}

// MaxEntries is the maximum number of cache entries before
// an item is evicted. Zero means no limit.
// OnEvicted optionally specificies a callback function to be
// executed when an entry is purged from the cache.
func NewLruMemCache(maxEntries int) *LruMemCache {
	lms := &LruMemCache{lc: lru.New(maxEntries)}
	lms.lc.OnEvicted = func(lru.Key, interface{}) {
		lms.nevict++
	}
	return lms
}

func (m *LruMemCache) Stats() LruStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	return LruStats{
		Items:     m.lc.Len(),
		Gets:      m.nget,
		Hits:      m.nhit,
		Evictions: m.nevict,
	}
}

// Get returns the cached values by the provided id.
// It returns nil if the the provided id  does not exist.
func (m *LruMemCache) Get(id interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nget++
	if v, ok := m.lc.Get(id); ok {
		m.nhit++
		return v, nil
	}

	return nil, ErrNotExist
}

// Set a id and values pair in cache.
func (m *LruMemCache) Set(id interface{}, vs interface{}) error {
	m.mu.Lock()
	m.lc.Add(id, vs)
	m.mu.Unlock()
	return nil
}

// Remove removes the provided id from the cache.
func (m *LruMemCache) Remove(id interface{}) error {
	m.mu.Lock()
	m.lc.Remove(id)
	m.mu.Unlock()
	return nil
}

// If id does not exist, create a new value by fn.
// the value will insert to cache if the value is not nil.
func (m *LruMemCache) Getsert(id interface{}, fn func() interface{}) (interface{}, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nget++
	v, ok := m.lc.Get(id)
	if ok {
		m.nhit++
	} else {
		if v = fn(); v != nil {
			m.lc.Add(id, v)
		}
	}

	return v, nil
}
