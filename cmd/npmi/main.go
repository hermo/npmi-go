package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

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

	if options.UseLocalCache {

		cache, err := cache.NewLocalCache(options.LocalCache.Dir)
		if err != nil {
			log.Fatalf("Can't use local cache: %v", err)
		}

		if !options.Force && cache.Has(hash) {
			fmt.Println("HIT, contents follow:")
			f, err := cache.GetReader(hash)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			io.Copy(os.Stdout, f)
		} else {
			fmt.Println("MISS, caching content")
			f, err := cache.GetWriter(hash)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			t := time.Now()
			f.Write([]byte(t.Format(time.UnixDate)))
		}
	}

}
