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

type Main struct {
	options          *cli.Options
	nodeConfig       *npmi.NodeConfig
	verboseConsole   cli.ConsoleWriter
	caches           []cache.Cacher
	modulesDirectory string
	lockFile         string
	installer        *npmi.Installer
}

func main() {
	options, err := cli.ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	m, err := NewMain(options)
	if err != nil {
		log.Fatal(err)
	}

	err = m.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func NewMain(options *cli.Options) (*Main, error) {
	nodeConfig := npmi.NewNodeConfig()
	if err := nodeConfig.Run(); err != nil {
		return nil, err
	}

	caches, err := initCaches(options)
	if err != nil {
		return nil, fmt.Errorf("cache init error: %v", err)
	}

	return &Main{
		modulesDirectory: DefaultModulesDirectory,
		lockFile:         DefaultLockFile,
		options:          options,
		nodeConfig:       nodeConfig,
		verboseConsole:   cli.NewConsole(options.Verbose),
		installer:        npmi.NewInstaller(nodeConfig),
		caches:           caches,
	}, nil
}

func (m *Main) Run() error {
	m.verboseConsole.Println("npmi-go start")

	platform, err := m.nodeConfig.GetPlatform()
	if err != nil {
		return fmt.Errorf("can't determine Node.js version: %v", err)
	}

	lockFileHash, err := npmi.HashFile(m.lockFile)
	if err != nil {
		return fmt.Errorf("can't hash lockfile: %v", err)
	}

	cacheKey, err := createCacheKey(platform, lockFileHash, m.options.PrecacheCommand)
	if err != nil {
		return fmt.Errorf("can't create cache key: %v", err)
	}

	installedFromCache, err := m.tryToInstallFromCache(cacheKey)
	if err != nil {
		return fmt.Errorf("can't install from cache: %v", err)
	}

	isInstallationFromNpmRequired := m.options.Force || !installedFromCache

	if isInstallationFromNpmRequired {
		m.verboseConsole.Println("Install start")
		if installedFromCache {
			m.verboseConsole.Println("NOTE: Cache was a HIT, install is forced")
		}
		err = m.installFromNpm(cacheKey)
		if err != nil {
			return fmt.Errorf("can't install from NPM: %v", err)
		}
	}

	m.verboseConsole.Println("npmi-go complete")
	return nil
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

func createCacheKey(platformKey string, lockFileHash string, precacheCommand string) (string, error) {
	cacheKey := fmt.Sprintf("%s-%s", platformKey, lockFileHash)
	if precacheCommand == "" {
		return cacheKey, nil
	}
	precacheHash, err := npmi.HashString(precacheCommand)
	if err != nil {
		return "", fmt.Errorf("could not hash precache command: %v", err)
	}

	cacheKey = fmt.Sprintf("%s-%s", cacheKey, precacheHash)
	return cacheKey, nil
}

func (m *Main) installFromNpm(cacheKey string) error {
	m.verboseConsole.Println("Install(npm).InstallPackages start")

	stdout, stderr, err := m.installer.Run()
	if err != nil {
		return fmt.Errorf("Install(npm).InstallPackages error: %v: %s", err, stderr)
	}

	m.verboseConsole.Printf("Install(npm).InstallPackages complete: success: %s\n", stdout)

	err = m.runPreCacheCommand()
	if err != nil {
		return fmt.Errorf("Install(npm).InstallPackages error: %v: %s", err, stderr)
	}

	if !files.DirectoryExists(m.modulesDirectory) {
		return fmt.Errorf("post-install: Modules directory not present after NPM install: %s", m.modulesDirectory)
	}
	m.verboseConsole.Println("Install complete")

	archiveFilename, err := m.createArchive(cacheKey)
	if err != nil {
		return fmt.Errorf("Install(npm).CreateArchive error: %v: %s", err, stderr)
	}
	defer m.removeArchiveAfterCaching(archiveFilename)

	err = m.storeArchiveInCache(cacheKey, archiveFilename)
	if err != nil {
		return fmt.Errorf("Install(npm).CreateArchive error: %v: %s", err, stderr)
	}
	return nil
}

func (m *Main) removeArchiveAfterCaching(archiveFilename string) {
	err := os.Remove(archiveFilename)
	if err != nil {
		log.Fatalf("Post-Archive: Could not remove temporary archive: %v", err)
	}

	m.verboseConsole.Printf("Post-Archive: Removed temporary archive %s\n", archiveFilename)
}

func (m *Main) createArchive(cacheKey string) (archiveFilename string, err error) {
	archiveFilename = createArchiveFilename(cacheKey)

	m.verboseConsole.Println("Archive start")
	m.verboseConsole.Printf("Archive creating %s\n", archiveFilename)

	err = archive.Create(archiveFilename, m.modulesDirectory)
	if err != nil {
		return "", fmt.Errorf("archive failed: %s", err)
	}

	m.verboseConsole.Println("Archive complete")
	return archiveFilename, nil
}

func createArchiveFilename(cacheKey string) string {
	return fmt.Sprintf("modules-%s.tar.gz", cacheKey)
}

func (m *Main) storeArchiveInCache(cacheKey string, archiveFilename string) error {
	m.verboseConsole.Println("Cache start")

	m.verboseConsole.Println("Cache.OpenArchive start")
	archiveFile, err := os.Open(archiveFilename)
	if err != nil {
		return fmt.Errorf("Cache.OpenArchive error: %s", err)
	}
	defer archiveFile.Close()
	m.verboseConsole.Println("Cache.OpenArchive complete")

	for _, cache := range m.caches {
		_, err := archiveFile.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("Cache(%s).ArchiveSeek error: %s", cache, err)
		}
		m.verboseConsole.Printf("Cache(%s).Put start\n", cache)
		err = cache.Put(cacheKey, archiveFile)
		if err != nil {
			return fmt.Errorf("Cache(%s).Put error: %s", cache, err)
		}

		m.verboseConsole.Printf("Cache(%s).Put complete\n", cache)
	}

	m.verboseConsole.Println("Cache complete")
	return nil
}

func (m *Main) runPreCacheCommand() error {
	if m.options.PrecacheCommand == "" {
		return nil
	}

	m.verboseConsole.Println("Install(npm).InstallPackages start")

	stdout, stderr, err := m.installer.RunPrecacheCommand(m.options.PrecacheCommand)
	if err != nil {
		return fmt.Errorf("Install(npm).PreCache error: %v: %s", err, stderr)
	}

	m.verboseConsole.Printf("Install(npm).PreCache complete: success: %s\n", stdout)
	return nil
}

func (m *Main) tryToInstallFromCache(cacheKey string) (foundInCache bool, err error) {
	m.verboseConsole.Printf("Lookup start, looking for cache key %s\n", cacheKey)

	foundInCache = false
	for _, cache := range m.caches {
		m.verboseConsole.Printf("Lookup(%s).Has start\n", cache)

		foundInCache, err := cache.Has(cacheKey)
		if err != nil {
			return false, fmt.Errorf("Lookup(%s).Has error: %s", cache, err)
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
			return false, fmt.Errorf("Lookup(%s).Get error: %s", cache, err)
		}

		m.verboseConsole.Printf("Lookup(%s).Get complete\n", cache)
		m.verboseConsole.Printf("Lookup(%s).Extract start\n", cache)

		if m.options.Force {
			m.verboseConsole.Printf("Lookup(%s).Extract SKIPPED, Force install requested\n", cache)
			continue
		}

		archiveManifest, err := archive.Extract(foundArchive)
		if err != nil {
			return false, fmt.Errorf("Lookup(%s).Extract error: %s", cache, err)
		}

		m.verboseConsole.Println("Cleanup start")

		numRemoved, err := files.RemoveFilesNotPresentInManifest(m.modulesDirectory, archiveManifest)
		if err != nil {
			return false, fmt.Errorf("cleanup error: %s", err)
		}

		m.verboseConsole.Printf("Cleanup complete, %d extraneous files removed\n", numRemoved)
		m.verboseConsole.Printf("Lookup(%s).Extract complete\n", cache)

		// Cache hit, no need to look further
		break
	}

	m.verboseConsole.Println("Lookup complete")
	return foundInCache, nil
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
