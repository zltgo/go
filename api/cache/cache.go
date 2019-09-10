package cache

import (
	"errors"
)

var (
	ErrNotExist = errors.New("cache: provided id does not exist in cache")
)

type Cache interface {
	// Get values  from cache by  id. If provided id does not exist in cache,
	// usually you should simply create a new values.
	Get(id interface{}) (interface{}, error)

	// Save id and values to the cache.
	// It replaces any existing values.
	Set(id interface{}, v interface{}) error

	// If id does not exist, create a new value by fn.
	Getsert(id interface{}, fn func() interface{}) (interface{}, error)

	// Remove values form cache by id.
	Remove(id interface{}) error

	// Clear purges all stored items from the cache.
	Clear() 
}
