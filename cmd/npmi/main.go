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

	cache, err := cache.NewLocalCache("./cache")
	if err != nil {
		log.Fatal(err)
	}

	if cache.Has(hash) {
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
