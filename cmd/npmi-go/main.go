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

var (
	verboseConsole cli.ConsoleWriter
)

func main() {
	if err := npmi.LocateRequiredBinaries(); err != nil {
		log.Fatal(err)
	}

	options, err := cli.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	verboseConsole = cli.NewConsole(options.Verbose)

	platformKey, err := npmi.DeterminePlatformKey()
	if err != nil {
		log.Fatalf("Can't determine Node.js version: %v", err)
	}

	caches, err := initCaches(options)
	if err != nil {
		log.Fatalf("Cache init error: %s", err)
	}

	installDependencies(platformKey, caches, options)
}

func installDependencies(platformKey string, caches []cache.Cacher, options *cli.Options) {
	verboseConsole.Println("npmi-go start")

	lockFileHash, err := npmi.HashFile(lockFile)
	if err != nil {
		log.Fatalf("Can't hash lockfile: %v", err)
	}

	cacheKey := createCacheKey(platformKey, lockFileHash, options)
	installedFromCache := tryToInstallFromCache(cacheKey, caches, options)
	isInstallationFromNpmRequired := options.Force || !installedFromCache

	if isInstallationFromNpmRequired {
		verboseConsole.Println("Install start")
		if installedFromCache {
			verboseConsole.Println("NOTE: Cache was a HIT, install is forced")
		}
		installFromNpm(cacheKey, caches, options)
	}

	verboseConsole.Println("npmi-go complete")
}

func installFromNpm(cacheKey string, caches []cache.Cacher, options *cli.Options) {
	verboseConsole.Println("Install(npm).InstallPackages start")

	stdout, stderr, err := npmi.InstallPackages()
	if err != nil {
		log.Fatalf("Install(npm).InstallPackages error: %v: %s", err, stderr)
	}

	verboseConsole.Printf("Install(npm).InstallPackages complete: success: %s\n", stdout)

	runPreCacheCommand(options)

	if !files.DirectoryExists(modulesDirectory) {
		log.Fatalf("Post-install: Modules directory not present after NPM install: %s", modulesDirectory)
	}
	verboseConsole.Println("Install complete")

	archiveFilename := createArchive(cacheKey, options)
	defer removeArchiveAfterCaching(archiveFilename)

	storeArchiveInCache(cacheKey, archiveFilename, caches, options)
}

func removeArchiveAfterCaching(archiveFilename string) {
	err := os.Remove(archiveFilename)
	if err != nil {
		log.Fatalf("Post-Archive: Could not remove temporary archive: %v", err)
	}

	verboseConsole.Printf("Post-Archive: Removed temporary archive %s", archiveFilename)
}

func createArchive(cacheKey string, options *cli.Options) string {
	archiveFilename := createArchiveFilename(cacheKey)

	verboseConsole.Println("Archive start")
	verboseConsole.Printf("Archive creating %s\n", archiveFilename)

	err := archive.Create(archiveFilename, modulesDirectory)
	if err != nil {
		log.Fatalf("Archive failed: %s", err)
	}

	verboseConsole.Println("Archive complete")
	return archiveFilename
}

func createArchiveFilename(cacheKey string) string {
	return fmt.Sprintf("modules-%s.tar.gz", cacheKey)
}

func storeArchiveInCache(cacheKey string, archiveFilename string, caches []cache.Cacher, options *cli.Options) {
	verboseConsole.Println("Cache start")

	verboseConsole.Printf("Cache.OpenArchive start")
	archiveFile, err := os.Open(archiveFilename)
	if err != nil {
		log.Fatalf("Cache.OpenArchive error: %s", err)
	}
	defer archiveFile.Close()
	verboseConsole.Println("Cache.OpenArchive complete")

	for _, cache := range caches {
		archiveFile.Seek(0, 0)
		verboseConsole.Printf("Cache(%s).Put start\n", cache)
		err = cache.Put(cacheKey, archiveFile)
		if err != nil {
			log.Fatalf("Cache(%s).Put error: %s", cache, err)
		}

		verboseConsole.Printf("Cache(%s).Put complete\n", cache)
	}

	verboseConsole.Println("Cache complete")
}

func runPreCacheCommand(options *cli.Options) {
	if options.PrecacheCommand == "" {
		return
	}

	verboseConsole.Println("Install(npm).InstallPackages start")

	stdout, stderr, err := npmi.RunPrecacheCommand(options.PrecacheCommand)
	if err != nil {
		log.Fatalf("Install(npm).PreCache error: %v: %s", err, stderr)
	}

	verboseConsole.Printf("Install(npm).PreCache complete: success: %s\n", stdout)
}

func tryToInstallFromCache(cacheKey string, caches []cache.Cacher, options *cli.Options) (foundInCache bool) {
	verboseConsole.Printf("Lookup start, looking for cache key %s\n", cacheKey)

	foundInCache = false
	for _, cache := range caches {
		verboseConsole.Printf("Lookup(%s).Has start\n", cache)

		foundInCache, err := cache.Has(cacheKey)
		if err != nil {
			log.Fatalf("Lookup(%s).Has error: %s", cache, err)
		}

		if !foundInCache {
			verboseConsole.Printf("Lookup(%s).Has complete: MISS\n", cache)
			// Cache miss, continue with next cache
			continue
		}
		verboseConsole.Printf("Lookup(%s).Has complete: HIT\n", cache)

		verboseConsole.Printf("Lookup(%s).Get start\n", cache)
		foundArchive, err := cache.Get(cacheKey)
		if err != nil {
			log.Fatalf("Lookup(%s).Get error: %s", cache, err)
		}

		verboseConsole.Printf("Lookup(%s).Get complete\n", cache)
		verboseConsole.Printf("Lookup(%s).Extract start\n", cache)

		if options.Force {
			verboseConsole.Printf("Lookup(%s).Extract SKIPPED, Force install requested\n", cache)
			continue
		}

		archiveManifest, err := archive.Extract(foundArchive)
		if err != nil {
			log.Fatalf("Lookup(%s).Extract error: %s", cache, err)
		}

		verboseConsole.Println("Cleanup start")

		numRemoved, err := archive.RemoveFilesNotPresentInManifest(modulesDirectory, archiveManifest)
		if err != nil {
			log.Fatalf("Cleanup error: %s", err)
		}

		verboseConsole.Printf("Cleanup complete, %d extraneous files removed\n", numRemoved)
		verboseConsole.Printf("Lookup(%s).Extract complete\n", cache)

		// Cache hit, no need to look further
		break
	}

	verboseConsole.Println("Lookup complete")
	return foundInCache
}

func createCacheKey(platformKey string, lockFileHash string, options *cli.Options) string {
	cacheKey := fmt.Sprintf("%s-%s", platformKey, lockFileHash)
	if options.PrecacheCommand != "" {
		if precacheHash, err := npmi.HashString(options.PrecacheCommand); err != nil {
			log.Fatalf("Could not hash precache command: %v", err)
		} else {
			cacheKey = fmt.Sprintf("%s-%s", cacheKey, precacheHash)
		}
	}
	return cacheKey
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
