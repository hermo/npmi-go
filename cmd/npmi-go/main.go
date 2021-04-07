package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hermo/npmi-go/internal/cli"
	"github.com/hermo/npmi-go/pkg/archive"
	"github.com/hermo/npmi-go/pkg/cache"
	"github.com/hermo/npmi-go/pkg/files"
	"github.com/hermo/npmi-go/pkg/npmi"
)

const (
	modulesDirectory = "node_modules"
	lockFile         = "package-lock.json"
)

func init() {
	if err := npmi.InitNodeBinaries(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	options, err := cli.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	env, err := npmi.DeterminePlatform()
	if err != nil {
		log.Fatalf("Can't determine Node.js version: %v", err)
	}

	caches, err := initCaches(options)
	if err != nil {
		log.Fatalf("Cache init error: %s", err)
	}

	if options.Verbose {
		fmt.Println("npmi-go start")
	}

	hash, err := npmi.HashFile(lockFile)
	if err != nil {
		log.Fatalf("Can't hash lockfile: %v", err)
	}

	key := createCacheKey(env, hash, options.PrecacheCmd)

	if options.Verbose {
		fmt.Printf("Lookup start, looking for key %s\n", key)
	}

	hit := false
	for _, cache := range caches {
		if options.Verbose {
			fmt.Printf("Lookup(%s).Has start\n", cache)
		}

		hit, err = cache.Has(key)
		if err != nil {
			log.Fatalf("Lookup(%s).Has error: %s", cache, err)
		}

		if !hit {
			if options.Verbose {
				fmt.Printf("Lookup(%s).Has complete: MISS\n", cache)
			}
			// Cache miss, continue with next cache
			continue
		}

		if options.Verbose {
			fmt.Printf("Lookup(%s).Has complete: HIT\n", cache)
			fmt.Printf("Lookup(%s).Get start\n", cache)
		}

		f, err := cache.Get(key)
		if err != nil {
			log.Fatalf("Lookup(%s).Get error: %s", cache, err)
		}

		if options.Verbose {
			fmt.Printf("Lookup(%s).Get complete\n", cache)
			fmt.Printf("Lookup(%s).Extract start\n", cache)
		}

		if options.Force {
			if options.Verbose {
				fmt.Printf("Lookup(%s).Extract SKIPPED, Force install requested\n", cache)
			}
			continue
		}

		manifest, err := archive.ExtractArchive("", f)
		if err != nil {
			log.Fatalf("Lookup(%s).Extract error: %s", cache, err)
		}

		if options.Verbose {
			fmt.Println("Cleanup start")
		}
		numRemoved, err := archive.CleanTree(modulesDirectory, manifest)
		if err != nil {
			log.Fatalf("Cleanup error: %s", err)
		}
		if options.Verbose {
			fmt.Printf("Cleanup complete, %d extraneous files removed\n", numRemoved)
		}

		if options.Verbose {
			fmt.Printf("Lookup(%s).Extract complete\n", cache)
		}

		// Cache hit, no need to look further
		break
	}

	if options.Verbose {
		fmt.Println("Lookup complete")
	}

	if options.Force || !hit {
		if options.Verbose {
			fmt.Println("Install start")
			if options.Force && hit {
				fmt.Println("NOTE: Cache was a HIT, install is forced")
			}
		}

		if options.Verbose {
			fmt.Println("Install(npm).InstallPackages start")
		}

		stdout, stderr, err := npmi.InstallPackages()
		if err != nil {
			log.Fatalf("Install(npm).InstallPackages error: %v: %s", err, stderr)
		}

		if options.Verbose {
			fmt.Printf("Install(npm).InstallPackages complete: success: %s\n", stdout)
		}

		if options.PrecacheCmd != "" {
			if options.Verbose {
				fmt.Println("Install(npm).InstallPackages start")
			}
			stdout, stderr, err = npmi.RunPrecacheCommand(options.PrecacheCmd)
			if err != nil {
				log.Fatalf("Install(npm).PreCache error: %v: %s", err, stderr)
			}
			if options.Verbose {
				fmt.Printf("Install(npm).PreCache complete: success: %s\n", stdout)
			}
		}

		if !files.IsExistingDir(modulesDirectory) {
			log.Fatalf("Post-install: Modules directory not present after NPM install: %s", modulesDirectory)
		}
		if options.Verbose {
			fmt.Println("Install complete")
		}
		filename := fmt.Sprintf("modules-%s.tar.gz", key)
		if options.Verbose {
			fmt.Println("Archive start")
			fmt.Printf("Archive creating %s\n", filename)
		}
		err = archive.Archive(filename, modulesDirectory)
		if err != nil {
			log.Fatalf("Archive failed: %s", err)
		}

		// Remove temp file when done
		defer func() {
			err := os.Remove(filename)
			if err != nil {
				log.Fatalf("Post-Archive: Could not remove temporary archive: %v", err)
			}
			if options.Verbose {
				fmt.Printf("Post-Archive: Removed temporary archive %s", filename)
			}
		}()

		if options.Verbose {
			fmt.Println("Archive complete")
			fmt.Println("Cache start")
		}

		for _, cache := range caches {
			if options.Verbose {
				fmt.Printf("Cache(%s).Open start\n", cache)
			}

			f, err := os.Open(filename)
			if err != nil {
				log.Fatalf("Cache(%s).Open error: %s", cache, err)
			}
			defer f.Close()

			if options.Verbose {
				fmt.Printf("Cache(%s).Open complete\n", cache)
				fmt.Printf("Cache(%s).Put start\n", cache)
			}
			err = cache.Put(key, f)
			if err != nil {
				log.Fatalf("Cache(%s).Put error: %s", cache, err)
			}
			if options.Verbose {
				fmt.Printf("Cache(%s).Put complete\n", cache)
			}
		}
		if options.Verbose {
			fmt.Println("Cache complete")
		}
	}

	if options.Verbose {
		fmt.Println("npmi-go complete")
	}
}

func createCacheKey(env string, hash string, precacheCmd string) string {
	key := fmt.Sprintf("%s-%s", env, hash)
	if precacheCmd != "" {
		if precacheHash, err := npmi.HashString(precacheCmd); err != nil {
			log.Fatalf("Could not hash precache command: %v", err)
		} else {
			return fmt.Sprintf("%s-%s", key, precacheHash)
		}
	}
	return key
}

func initMinioCache(options *cli.MinioCacheOptions) (cache.Cacher, error) {
	cache := cache.NewMinioCache(options.Endpoint, options.AccessKeyID, options.SecretAccessKey, options.Bucket, options.UseTLS)
	err := cache.Dial()
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func initLocalCache(options *cli.LocalCacheOptions) (cache.Cacher, error) {
	return cache.NewLocalCache(options.Dir)
}

func initCaches(options *cli.Options) ([]cache.Cacher, error) {
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
