package cache

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/hermo/npmi-go/pkg/files"
)

type localCache struct {
	dir string
}

// NewLocalCache creates a Cacher, which stores data locally in a directory
func NewLocalCache(dir string) (Cacher, error) {
	if dir == "" {
		return nil, fmt.Errorf("no cache directory given")
	}
	if !files.DirectoryExists(dir) {
		return nil, fmt.Errorf("%s is not a valid directory", dir)
	}
	return &localCache{dir}, nil
}

func (cache *localCache) joinPath(key string) string {
	// TODO: Validate key format
	return path.Join(cache.dir, key)
}

// Has indicates whether a LocalCache contains a given key or not
func (cache *localCache) Has(key string) (bool, error) {
	return files.IsExistingFile(cache.joinPath(key))
}

// Get fetches something from the cache
func (cache *localCache) Get(key string) (io.Reader, error) {
	return os.Open(cache.joinPath(key))
}

// Put stores something in the cache
func (cache *localCache) Put(key string, reader io.Reader) error {
	f, err := os.Create(cache.joinPath(key))
	if err != nil {
		return nil
	}

	defer f.Close()
	_, err = io.Copy(f, reader)
	if err != nil {
		return err
	}
	return nil
}
