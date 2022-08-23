package npmi

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	"github.com/hermo/npmi-go/pkg/archive"
	"github.com/hermo/npmi-go/pkg/cache"
	"github.com/hermo/npmi-go/pkg/files"
	"github.com/hermo/npmi-go/pkg/hash"
)

const (
	defaultModulesDirectory = "node_modules"
	defaultLockFile         = "package-lock.json"
)

type main struct {
	caches           []cache.Cacher
	installer        *NpmInstaller
	lockFile         string
	modulesDirectory string
	options          *Options
	platform         string
	log              hclog.Logger
}

// New builds a configuration for the current runtime and returns a pre-configured NPMI main
func New(options *Options) (*main, error) {
	builder := NewConfigBuilder()
	builder.WithNodeAndNpmFromPath()
	config, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return NewWithConfig(options, config)
}

// NewWithConfig creates a NPMI main using the supplied options and config
func NewWithConfig(options *Options, config *Config) (*main, error) {
	caches, err := initCaches(options)
	if err != nil {
		return nil, fmt.Errorf("cache init error: %v", err)
	}

	log := hclog.New(&hclog.LoggerOptions{
		Name:       "npmi",
		Level:      hclog.LevelFromString(options.LogLevel.String()),
		JSONFormat: false,
	})

	return &main{
		modulesDirectory: defaultModulesDirectory,
		lockFile:         defaultLockFile,
		options:          options,
		log:              log,
		platform:         config.Platform,
		installer:        NewNpmInstaller(config, log.Named("npmInstaller")),
		caches:           caches,
	}, nil
}

// Run determines and performs the steps required to install the desired dependencies
func (m *main) Run() error {
	m.log.Trace("start")
	cacheKey, err := m.createCacheKey()
	if err != nil {
		return err
	}

	installedFromCache, err := m.tryToInstallFromCache(cacheKey)
	if err != nil {
		return err
	}

	isInstallationFromNpmRequired := m.options.Force || !installedFromCache

	if isInstallationFromNpmRequired {
		log := m.log.Named("install")
		log.Trace("start")
		if installedFromCache {
			log.Warn("Package found in cache, force install is enabled")
		}

		err = m.installFromNpm()
		if err != nil {
			return err
		}

		err = m.cacheInstalledPackages(cacheKey)
		if err != nil {
			return err
		}
	}

	m.log.Trace("complete")
	m.log.Debug("Installation complete")
	return nil
}

func (m *main) createCacheKey() (string, error) {
	lockFileHash, err := hash.File(m.lockFile)
	if err != nil {
		return "", fmt.Errorf("can't hash lockfile: %v", err)
	}

	cacheKey, err := cache.CreateKey(m.platform, lockFileHash, m.options.PrecacheCommand)
	if err != nil {
		return "", err
	}
	return cacheKey, nil

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

func (m *main) installFromNpm() error {
	log := m.log.Named("installPackages")
	log.Trace("start")

	stdout, stderr, err := m.installer.Run()
	if err != nil {
		log.Error("failed", "error", err, "stderr", hclog.Quote(stderr))
		return fmt.Errorf("installPackages: %v", err)
	}

	log.Trace("complete", "stdout", hclog.Quote(stdout))

	if !files.DirectoryExists(m.modulesDirectory) {
		return fmt.Errorf("post-install: Modules directory '%s' not present after NPM install", m.modulesDirectory)
	}

	err = m.runPreCacheCommand()
	if err != nil {
		return fmt.Errorf("pre-cache: %v: %s", err, stderr)
	}

	log.Trace("complete")
	return nil
}

func (m *main) cacheInstalledPackages(cacheKey string) error {
	archiveFilename, err := m.createArchive(cacheKey)
	if err != nil {
		return fmt.Errorf("create-archive: %v", err)
	}
	defer m.removeArchiveAfterCaching(archiveFilename)

	err = m.storeArchiveInCache(cacheKey, archiveFilename)
	if err != nil {
		return fmt.Errorf("store-archive: %v", err)
	}
	return nil
}

func (m *main) removeArchiveAfterCaching(archiveFilename string) {
	log := m.log.Named("createArchive")
	err := os.Remove(archiveFilename)
	if err != nil {
		log.Error("Could not remove temporary archive: %v", err)
		os.Exit(1)
	}

	log.Debug("Removed temporary archive", "path", archiveFilename)
}

func (m *main) createArchive(cacheKey string) (archiveFilename string, err error) {
	log := m.log.Named("createArchive")
	log.Trace("start")

	archivePath := filepath.Join(m.options.TempDir, createArchiveFilename(cacheKey))
	log.Debug("Creating archive", "path", archivePath)

	err = archive.Create(archivePath, m.modulesDirectory)
	if err != nil {
		log.Error("failed", "error", err)
		return "", err
	}

	log.Trace("complete")
	return archivePath, nil
}

func createArchiveFilename(cacheKey string) string {
	return fmt.Sprintf("modules-%s.tar.gz", cacheKey)
}

func (m *main) storeArchiveInCache(cacheKey string, archiveFilename string) error {
	log := m.log.Named("cache")
	log.Trace("start")

	archiveFile, err := os.Open(archiveFilename)
	if err != nil {
		return fmt.Errorf("Cache.OpenArchive error: %s", err)
	}
	defer archiveFile.Close()

	for _, cache := range m.caches {
		cLog := log.Named(fmt.Sprint(cache))
		_, err := archiveFile.Seek(0, 0)
		if err != nil {
			cLog.Error("Archive seek failed", "error", err)
			return err
		}
		cLog.Trace("start")
		err = cache.Put(cacheKey, archiveFile)
		if err != nil {
			cLog.Error("Put failed", "error", err)
			return err
		}

		cLog.Trace("complete")
	}

	log.Trace("complete")
	return nil
}

func (m *main) runPreCacheCommand() error {
	if m.options.PrecacheCommand == "" {
		return nil
	}

	log := m.log.Named("preCache")

	log.Trace("start")

	stdout, stderr, err := m.installer.RunPrecacheCommand(m.options.PrecacheCommand)
	if err != nil {
		log.Error("Precache command failed", "command", hclog.Quote(m.options.PrecacheCommand), "error", err, "stderr", hclog.Quote(stderr))
		return err
	}

	log.Trace("complete", "stdout", hclog.Quote(stdout))
	return nil
}

func (m *main) tryToInstallFromCache(cacheKey string) (foundInCache bool, err error) {
	log := m.log.Named("cache")
	log.Trace("start", "cacheKey", cacheKey)

	if len(m.caches) == 0 {
		log.Warn("No caches configured, cache lookup will always fail")
	}

	foundInCache = false
	for _, cache := range m.caches {
		cLog := log.Named(fmt.Sprint(cache))
		lookupLog := cLog.Named("lookup")
		lookupLog.Trace("start")

		foundInCache, err = cache.Has(cacheKey)
		if err != nil {
			lookupLog.Error("failed", "error", err)
			return false, err
		}

		if !foundInCache {
			lookupLog.Trace("complete")
			lookupLog.Debug("cache MISS")
			// Cache miss, continue with next cache
			continue
		}

		lookupLog.Trace("complete")
		lookupLog.Debug("cache HIT")

		fetchLog := cLog.Named("fetch")

		fetchLog.Trace("start")
		foundArchive, err := cache.Get(cacheKey)
		if err != nil {
			fetchLog.Error("failed", "error", err)
			return false, err
		}
		fetchLog.Trace("complete")

		if m.options.Force {
			cLog.Debug("Force install requested, skipping Extraction")
			continue
		}

		extractLog := cLog.Named("extract")
		extractLog.Trace("start")

		archiveManifest, err := archive.Extract(foundArchive)
		if err != nil {
			extractLog.Error("failed", "error", err)
			return false, err
		}

		cleanupLog := extractLog.Named("cleanup")
		cleanupLog.Trace("start")

		numRemoved, err := files.RemoveFilesNotPresentInManifest(m.modulesDirectory, archiveManifest)
		if err != nil {
			cleanupLog.Error("failed", "error", err)
			return false, err
		}

		cleanupLog.Trace("complete", "numFilesRemoved", numRemoved)
		extractLog.Trace("complete")
		cLog.Debug("packages successfully installed from cache")

		// Cache hit, no need to look further
		break
	}

	log.Trace("complete")
	return foundInCache, nil
}

func initMinioCache(options *MinioCacheOptions) (cache.Cacher, error) {
	cache := cache.NewMinioCache(options.Endpoint, options.AccessKeyID, options.SecretAccessKey, options.Bucket, options.UseTLS, options.InsecureTLS)
	err := cache.Dial()
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func initLocalCache(options *LocalCacheOptions) (cache.Cacher, error) {
	return cache.NewLocalCache(options.Dir)
}
