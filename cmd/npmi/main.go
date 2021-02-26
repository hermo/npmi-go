package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hermo/npmi-go/pkg/archive"
	"github.com/hermo/npmi-go/pkg/cache"
	"github.com/hermo/npmi-go/pkg/npmi"
)

func main() {
	options, _ := ParseFlags()
	fmt.Printf("Flags: %+v\n", options)
	env, err := npmi.DeterminePlatform()
	if err != nil {
		log.Fatalf("Can't determine Node.js version: %v", err)
	}
	fmt.Printf("Node env: %s\n", env)

	hash, err := npmi.HashFile("package-lock.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Hash: %s\n", hash)

	key := createKey(env, hash)

	caches, err := initCaches(options)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("-- Lookup phase --")
	hit := false
	for _, cache := range caches {
		hit, err := cache.Has(key)
		if err != nil {
			log.Fatal(err)
		}
		if hit {
			fmt.Printf("%T HIT, extracting\n", cache)
			f, err := cache.Get(key)
			if err != nil {
				log.Fatal(err)
			}
			err = archive.ExtractArchive(f)
			if err != nil {
				log.Fatal(err)
			}
			break
		}
	}

	fmt.Println("-- Cache phase --")
	filename := fmt.Sprintf("%s.tar.gz", key)
	err = archive.Archive(filename, "pkg")
	if err != nil {
		log.Fatal(err)
	}

	for _, cache := range caches {
		if options.Force || !hit {
			fmt.Printf("%T MISS, updating cache\n", cache)

			f, err := os.Open(filename)
			defer f.Close()
			if err != nil {
				log.Fatal(err)
			}

			err = cache.Put(key, f)
			if err != nil {
				log.Fatal(err)
			}
		}

	}

}

func createKey(env string, hash string) string {
	return fmt.Sprintf("%s-%s", env, hash)
}

func initMinioCache(options *MinioCacheOptions) (cache.Cacher, error) {
	cache := cache.NewMinioCache()
	err := cache.Dial(options.Endpoint, options.AccessKeyID, options.SecretAccessKey, options.UseTLS)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func initLocalCache(options *LocalCacheOptions) (cache.Cacher, error) {
	return cache.NewLocalCache(options.Dir)
}

func initCaches(options *Options) ([]cache.Cacher, error) {
	var caches []cache.Cacher
	if options.UseLocalCache {
		cache, err := initLocalCache(options.LocalCache)
		if err != nil {
			return nil, err
		}
		caches = append(caches, cache)
	}

	if options.UseMinioCache {
		cache, err := initMinioCache(options.MinioCache)
		if err != nil {
			return nil, err
		}
		caches = append(caches, cache)
	}
	return caches, nil
}
