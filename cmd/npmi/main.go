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

	caches, err := initCaches(options)
	if err != nil {
		log.Fatal(err)
	}

	for _, cache := range caches {
		fmt.Printf("CACHE: %T\n", cache)
		hit, err := cache.Has(hash)
		if err != nil {
			log.Fatal(err)
		}
		if hit {
			fmt.Println("Cache HIT")
			f, err := cache.Get(hash)
			if err != nil {
				log.Fatal(err)
			}
			err = archive.DecompressModules(f)
			if err != nil {
				log.Fatal(err)
			}
			break
		}

		if options.Force || !hit {
			fmt.Println("MISS, caching content")
			filename, err := archive.CompressModules()
			if err != nil {
				log.Fatal(err)
			}

			f, err := os.Open(filename)
			defer f.Close()
			if err != nil {
				log.Fatal(err)
			}

			err = cache.Put(hash, f)
			if err != nil {
				log.Fatal(err)
			}
		}

	}

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
