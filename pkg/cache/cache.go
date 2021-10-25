package cache

import (
	"fmt"
	"io"

	"github.com/hermo/npmi-go/pkg/hash"
)

// Cacher represents a cache
type Cacher interface {
	Has(key string) (bool, error)
	Put(key string, reader io.Reader) error
	Get(key string) (io.Reader, error)
}

func CreateKey(platformKey string, lockFileHash string, precacheCommand string) (string, error) {
	cacheKey := fmt.Sprintf("%s-%s", platformKey, lockFileHash)
	if precacheCommand == "" {
		return cacheKey, nil
	}
	precacheHash, err := hash.String(precacheCommand)
	if err != nil {
		return "", fmt.Errorf("could not hash precache command: %v", err)
	}

	cacheKey = fmt.Sprintf("%s-%s", cacheKey, precacheHash)
	return cacheKey, nil
}
