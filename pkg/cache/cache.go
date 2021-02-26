package cache

import "io"

// Cacher represents a cache
type Cacher interface {
	Has(key string) (bool, error)
	Put(key string, reader io.Reader) error
	Get(key string) (io.Reader, error)
}
