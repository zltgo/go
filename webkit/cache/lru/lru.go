/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package lru implements an LRU cache.
package lru

import (
	"container/list"
	"sync"
)

// Cache is an LRU cache. It is safe for concurrent access.
type Cache struct {
	// MaxEntries is the maximum number of cache entries before
	// an item is evicted. Zero means no limit.
	MaxEntries int

	// OnEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	OnEvicted func(key Key, value interface{})

	ll    *list.List
	cache map[interface{}]*list.Element
	mu     sync.RWMutex

	//stats
	nhit   int64 // number of hit
	nget   int64 // number of get
	nevict int64 // number of evictions
}

// A Key may be any value that is comparable. See http://golang.org/ref/spec#Comparison_operators
type Key interface{}

type entry struct {
	key   Key
	value interface{}
}

// LruStats are returned by stats accessors on Group.
type Stats struct {
	Items     int
	Gets      int64
	Hits      int64
	Evictions int64
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func New(maxEntries int) *Cache {
	return &Cache{
		MaxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[interface{}]*list.Element),
	}
}

func (c *Cache) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return Stats{
		Items:     c.ll.Len(),
		Gets:      c.nget,
		Hits:      c.nhit,
		Evictions: c.nevict,
	}
}

// Add adds a value to the cache.
//if number of entries reach the MaxEntries, remove the oldest and return it.
func (c *Cache) Add(key Key, value interface{}) {
	c.mu.Lock()
	removed := c.add(key, value)
	c.mu.Unlock()

	if removed != nil && c.OnEvicted != nil {
		c.OnEvicted(removed.key, removed.value)
	}
}

// if number of entries reach the MaxEntries, remove the oldest and return it.
func (c *Cache) add(key Key, value interface{}) (removed *entry) {
	if c.cache == nil {
		c.cache = make(map[interface{}]*list.Element)
		c.ll = list.New()
	}
	if ee, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ee)
		ee.Value.(*entry).value = value
		return nil
	}
	ele := c.ll.PushFront(&entry{key, value})
	c.cache[key] = ele

	//remove oldest
	if c.MaxEntries != 0 && c.ll.Len() > c.MaxEntries {
		if ele = c.ll.Back(); ele != nil {
			c.ll.Remove(ele)
			removed = ele.Value.(*entry)
			delete(c.cache, removed.key)
			c.nevict++
		}
	}
	return removed
}


// Get looks up a key's value from the cache.
func (c *Cache) Get(key Key) (value interface{}, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.get(key)
}

func (c *Cache) get(key Key) (value interface{}, ok bool) {
	c.nget++
	if c.cache == nil {
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		c.nhit++
		return ele.Value.(*entry).value, true
	}
	return
}

// If id does not exist, create a new value by fn.
// the value will insert to cache if the value is not nil.
func (c *Cache) Getsert(key Key, fn func() interface{}) interface{} {
	c.mu.Lock()
	if v, ok := c.get(key); ok {
		c.mu.Unlock()
		return v
	}
	//create a new element by fn
	newValue := fn()
	removed := c.add(key, newValue)
	c.mu.Unlock()

	if removed != nil && c.OnEvicted != nil {
		c.OnEvicted(removed.key, removed.value)
	}
	return newValue
}

// Remove removes the provided key from the cache.
func (c *Cache) Remove(key Key) {
	var removed *entry
	c.mu.Lock()
	if c.cache == nil {
		c.mu.Unlock()
		return
	}
	if ele, hit := c.cache[key]; hit {
		c.ll.Remove(ele)
		removed = ele.Value.(*entry)
		delete(c.cache, removed.key)
		c.nevict++
	}
	c.mu.Unlock()

	//eviction
	if c.OnEvicted != nil && removed != nil {
		c.OnEvicted(removed.key, removed.value)
	}
}

// RemoveOldest removes the oldest item from the cache.
func (c *Cache) RemoveOldest() {
	var oldest *entry
	c.mu.Lock()
	if c.cache == nil {
		c.mu.Unlock()
		return
	}
	if ele := c.ll.Back();ele != nil {
		c.ll.Remove(ele)
		oldest := ele.Value.(*entry)
		delete(c.cache, oldest.key)
		c.nevict++
	}
	c.mu.Unlock()

	//eviction
	if c.OnEvicted != nil {
		c.OnEvicted(oldest.key, oldest.value)
	}
}


// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cache == nil {
		return 0
	}
	return c.ll.Len()
}

// Clear purges all stored items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	c.nevict += int64(c.ll.Len())
	c.ll = nil
	tmp := c.cache
	c.cache = nil
	c.mu.Unlock()

	if c.OnEvicted != nil {
		for _, e := range tmp {
			kv := e.Value.(*entry)
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

