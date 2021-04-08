package cache

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/hermo/npmi-go/pkg/files"
)

// LocalCache represents a local cache instance
type LocalCache struct {
	dir string
}

// NewLocalCache creates a new LocalCache
func NewLocalCache(dir string) (Cacher, error) {
	if dir == "" {
		return nil, fmt.Errorf("No cache directory given")
	}
	if !files.DirectoryExists(dir) {
		return nil, fmt.Errorf("%s is not a valid directory", dir)
	}
	return &LocalCache{dir}, nil
}

func (cache *LocalCache) joinPath(key string) string {
	// TODO: Validate key format
	return path.Join(cache.dir, key)
}

// Has indicates whether a LocalCache contains a given key or not
func (cache *LocalCache) Has(key string) (bool, error) {
	return files.IsExistingFile(cache.joinPath(key))
}

// Get fetches something from the cache
func (cache *LocalCache) Get(key string) (io.Reader, error) {
	return os.Open(cache.joinPath(key))
}

// Put stores something in the cache
func (cache *LocalCache) Put(key string, reader io.Reader) error {
	f, err := os.Create(cache.joinPath(key))
	if err != nil {
		return nil
	}
	_, err = io.Copy(f, reader)
	if err != nil {
		return err
	}
	return nil
}

func (cache *LocalCache) String() string {
	return "local"
}
