package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"

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

	_, err = archive.CompressModules()
	if err != nil {
		log.Fatal(err)
	}

	if options.UseLocalCache {
		cache, err := cache.NewLocalCache(options.LocalCache.Dir)
		if err != nil {
			log.Fatalf("Can't use local cache: %v", err)
		}

		hit := cache.Has(hash)
		if hit {
			fmt.Println("Local HIT, contents follow:")
			f, err := cache.Get(hash)
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(os.Stdout, f)
			fmt.Println("")
		}

		if options.Force || !hit {
			fmt.Println("Local MISS, caching content")
			var buf bytes.Buffer

			t := time.Now()
			buf.Write([]byte(t.Format(time.UnixDate)))

			err := cache.Put(hash, &buf)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	if options.UseMinioCache {
		cache := cache.NewMinioCache()
		err := cache.Dial(options.MinioCache.Endpoint, options.MinioCache.AccessKeyID, options.MinioCache.SecretAccessKey, options.MinioCache.UseTLS)
		if err != nil {
			log.Fatal(err)
		}
		hit, err := cache.Has(hash)
		if err != nil {
			log.Fatalf("Minio Cache error: %#v", err)
		}
		if hit {
			fmt.Println("Minio HIT, contents follow: ")
			f, err := cache.Get(hash)
			if err != nil {
				log.Fatal(err)
			}
			io.Copy(os.Stdout, f)
			fmt.Println("")
		}

		if options.Force || !hit {
			fmt.Println("Minio MISS")
			t := time.Now()

			var buf bytes.Buffer

			buf.Write([]byte(t.Format(time.UnixDate)))

			err := cache.Put(hash, &buf)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("Minio cache updated")
		}
	}
}
