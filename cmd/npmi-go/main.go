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
	DefaultModulesDirectory = "node_modules"
	DefaultLockFile         = "package-lock.json"
)

func main() {
	n := npmi.NewInstaller()

	if err := n.LocateRequiredBinaries(); err != nil {
		log.Fatal(err)
	}

	options, err := cli.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	m := NewMain(options)
	m.Run(n)
}

type Main struct {
	options          *cli.Options
	verboseConsole   cli.ConsoleWriter
	caches           []cache.Cacher
	modulesDirectory string
	lockFile         string
	n                *npmi.Installer
}

func NewMain(options *cli.Options) *Main {
	return &Main{
		modulesDirectory: DefaultModulesDirectory,
		lockFile:         DefaultLockFile,
		options:          options,
		verboseConsole:   cli.NewConsole(options.Verbose),
	}
}

func (m *Main) Run(n *npmi.Installer) {
	m.n = n
	if err := m.initCaches(); err != nil {
		log.Fatalf("Cache init error: %s", err)
	}

	platformKey, err := n.DeterminePlatformKey()
	if err != nil {
		log.Fatalf("Can't determine Node.js version: %v", err)
	}

	m.verboseConsole.Println("npmi-go start")

	lockFileHash, err := npmi.HashFile(m.lockFile)
	if err != nil {
		log.Fatalf("Can't hash lockfile: %v", err)
	}

	cacheKey := m.createCacheKey(platformKey, lockFileHash)
	installedFromCache := m.tryToInstallFromCache(cacheKey)
	isInstallationFromNpmRequired := m.options.Force || !installedFromCache

	if isInstallationFromNpmRequired {
		m.verboseConsole.Println("Install start")
		if installedFromCache {
			m.verboseConsole.Println("NOTE: Cache was a HIT, install is forced")
		}
		m.installFromNpm(cacheKey)
	}

	m.verboseConsole.Println("npmi-go complete")
}

func (m *Main) installFromNpm(cacheKey string) {
	m.verboseConsole.Println("Install(npm).InstallPackages start")

	stdout, stderr, err := m.n.InstallPackages()
	if err != nil {
		log.Fatalf("Install(npm).InstallPackages error: %v: %s", err, stderr)
	}

	m.verboseConsole.Printf("Install(npm).InstallPackages complete: success: %s\n", stdout)

	m.runPreCacheCommand()

	if !files.DirectoryExists(m.modulesDirectory) {
		log.Fatalf("Post-install: Modules directory not present after NPM install: %s", m.modulesDirectory)
	}
	m.verboseConsole.Println("Install complete")

	archiveFilename := m.createArchive(cacheKey)
	defer m.removeArchiveAfterCaching(archiveFilename)

	m.storeArchiveInCache(cacheKey, archiveFilename)
}

func (m *Main) removeArchiveAfterCaching(archiveFilename string) {
	err := os.Remove(archiveFilename)
	if err != nil {
		log.Fatalf("Post-Archive: Could not remove temporary archive: %v", err)
	}

	m.verboseConsole.Printf("Post-Archive: Removed temporary archive %s\n", archiveFilename)
}

func (m *Main) createArchive(cacheKey string) string {
	archiveFilename := createArchiveFilename(cacheKey)

	m.verboseConsole.Println("Archive start")
	m.verboseConsole.Printf("Archive creating %s\n", archiveFilename)

	err := archive.Create(archiveFilename, m.modulesDirectory)
	if err != nil {
		log.Fatalf("Archive failed: %s", err)
	}

	m.verboseConsole.Println("Archive complete")
	return archiveFilename
}

func createArchiveFilename(cacheKey string) string {
	return fmt.Sprintf("modules-%s.tar.gz", cacheKey)
}

func (m *Main) storeArchiveInCache(cacheKey string, archiveFilename string) {
	m.verboseConsole.Println("Cache start")

	m.verboseConsole.Println("Cache.OpenArchive start")
	archiveFile, err := os.Open(archiveFilename)
	if err != nil {
		log.Fatalf("Cache.OpenArchive error: %s", err)
	}
	defer archiveFile.Close()
	m.verboseConsole.Println("Cache.OpenArchive complete")

	for _, cache := range m.caches {
		_, err := archiveFile.Seek(0, 0)
		if err != nil {
			log.Fatalf("Cache(%s).ArchiveSeek error: %s", cache, err)
		}
		m.verboseConsole.Printf("Cache(%s).Put start\n", cache)
		err = cache.Put(cacheKey, archiveFile)
		if err != nil {
			log.Fatalf("Cache(%s).Put error: %s", cache, err)
		}

		m.verboseConsole.Printf("Cache(%s).Put complete\n", cache)
	}

	m.verboseConsole.Println("Cache complete")
}

func (m *Main) runPreCacheCommand() {
	if m.options.PrecacheCommand == "" {
		return
	}

	m.verboseConsole.Println("Install(npm).InstallPackages start")

	stdout, stderr, err := m.n.RunPrecacheCommand(m.options.PrecacheCommand)
	if err != nil {
		log.Fatalf("Install(npm).PreCache error: %v: %s", err, stderr)
	}

	m.verboseConsole.Printf("Install(npm).PreCache complete: success: %s\n", stdout)
}

func (m *Main) tryToInstallFromCache(cacheKey string) (foundInCache bool) {
	m.verboseConsole.Printf("Lookup start, looking for cache key %s\n", cacheKey)

	foundInCache = false
	for _, cache := range m.caches {
		m.verboseConsole.Printf("Lookup(%s).Has start\n", cache)

		foundInCache, err := cache.Has(cacheKey)
		if err != nil {
			log.Fatalf("Lookup(%s).Has error: %s", cache, err)
		}

		if !foundInCache {
			m.verboseConsole.Printf("Lookup(%s).Has complete: MISS\n", cache)
			// Cache miss, continue with next cache
			continue
		}
		m.verboseConsole.Printf("Lookup(%s).Has complete: HIT\n", cache)

		m.verboseConsole.Printf("Lookup(%s).Get start\n", cache)
		foundArchive, err := cache.Get(cacheKey)
		if err != nil {
			log.Fatalf("Lookup(%s).Get error: %s", cache, err)
		}

		m.verboseConsole.Printf("Lookup(%s).Get complete\n", cache)
		m.verboseConsole.Printf("Lookup(%s).Extract start\n", cache)

		if m.options.Force {
			m.verboseConsole.Printf("Lookup(%s).Extract SKIPPED, Force install requested\n", cache)
			continue
		}

		archiveManifest, err := archive.Extract(foundArchive)
		if err != nil {
			log.Fatalf("Lookup(%s).Extract error: %s", cache, err)
		}

		m.verboseConsole.Println("Cleanup start")

		numRemoved, err := archive.RemoveFilesNotPresentInManifest(m.modulesDirectory, archiveManifest)
		if err != nil {
			log.Fatalf("Cleanup error: %s", err)
		}

		m.verboseConsole.Printf("Cleanup complete, %d extraneous files removed\n", numRemoved)
		m.verboseConsole.Printf("Lookup(%s).Extract complete\n", cache)

		// Cache hit, no need to look further
		break
	}

	m.verboseConsole.Println("Lookup complete")
	return foundInCache
}

func (m *Main) createCacheKey(platformKey string, lockFileHash string) string {
	cacheKey := fmt.Sprintf("%s-%s", platformKey, lockFileHash)
	if m.options.PrecacheCommand != "" {
		if precacheHash, err := npmi.HashString(m.options.PrecacheCommand); err != nil {
			log.Fatalf("Could not hash precache command: %v", err)
		} else {
			cacheKey = fmt.Sprintf("%s-%s", cacheKey, precacheHash)
		}
	}
	return cacheKey
}

func (m *Main) initCaches() error {
	var caches []cache.Cacher
	if m.options.UseLocalCache {
		cache, err := initLocalCache(m.options.LocalCache)
		if err != nil {
			return err
		}
		caches = append(caches, cache)
	}

	if m.options.UseMinioCache {
		cache, err := initMinioCache(m.options.MinioCache)
		if err != nil {
			return err
		}
		caches = append(caches, cache)
	}
	m.caches = caches
	return nil
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
