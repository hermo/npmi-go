package cache

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/hashicorp/go-hclog"
	"github.com/hermo/npmi-go/pkg/files"
)

type localCache struct {
	dir string
	log hclog.Logger
}

// NewLocalCache creates a Cacher, which stores data locally in a directory
func NewLocalCache(dir string, log hclog.Logger) (Cacher, error) {
	if dir == "" {
		return nil, fmt.Errorf("no cache directory given")
	}
	if !files.DirectoryExists(dir) {
		return nil, fmt.Errorf("'%s' is not a valid directory", dir)
	}
	return &localCache{dir, log}, nil
}

func (cache *localCache) joinPath(key string) string {
	// TODO: Validate key format
	return path.Join(cache.dir, key)
}

// Has indicates whether a LocalCache contains a given key or not
func (cache *localCache) Has(key string) (bool, error) {
	log := cache.log.Named("has")
	path := cache.joinPath(key)
	log.Trace("start", "key", key, "path", path)
	return files.IsExistingFile(path)
}

// Get fetches something from the cache
func (cache *localCache) Get(key string) (io.Reader, error) {
	log := cache.log.Named("get")
	path := cache.joinPath(key)
	log.Trace("start", "key", key, "path", path)
	return os.Open(path)
}

// Put stores something in the cache
func (cache *localCache) Put(key string, reader io.Reader) error {
	log := cache.log.Named("put")
	path := cache.joinPath(key)
	log.Trace("start", "key", key, "path", path)
	f, err := os.Create(path)
	if err != nil {
		log.Error("create failed", "error", err)
		return nil
	}

	defer f.Close()
	_, err = io.Copy(f, reader)
	if err != nil {
		log.Error("copy failed", "error", err)
		return err
	}
	log.Trace("complete")
	return nil
}

func (cache *localCache) String() string {
	return "local"
}
