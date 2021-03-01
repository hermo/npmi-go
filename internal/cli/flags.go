package cli

import (
	"flag"
	"fmt"
	"os"
)

// MinioCacheOptions contains configuration for Minio Cache
type MinioCacheOptions struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
	UseTLS          bool
}

// LocalCacheOptions constains configuration for Local Cache
type LocalCacheOptions struct {
	Dir string
}

// Options describes the runtime configuration
type Options struct {
	Verbose       bool
	Force         bool
	UseLocalCache bool
	UseMinioCache bool
	MinioCache    *MinioCacheOptions
	LocalCache    *LocalCacheOptions
	PrecacheCmd   string
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	usage = `npmi %s, commit %s, built at %s.
npmi installs NPM packages from a cache to speed up repeating installations.

Supported caches:
-  local		Data is cached locally in a directory.
-  minio		Data is cached to a (shared) Minio instance.

When using both caches, the local one is accessed first.

USAGE:
 npmi [OPTIONS]

OPTIONS:
`
)

// ParseFlags parses command line flags
func ParseFlags() (*Options, error) {
	options := &Options{}
	localCache := &LocalCacheOptions{}
	minioCache := MinioCacheOptions{}
	options.LocalCache = localCache
	options.MinioCache = &minioCache

	flag.BoolVar(&options.Verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&options.Force, "force", false, "Force (re)installation of NPM deps and update cache(s)")
	flag.BoolVar(&options.UseLocalCache, "local", true, "Use local cache")
	flag.StringVar(&localCache.Dir, "local-dir", os.TempDir(), "Local cache directory")
	flag.BoolVar(&options.UseMinioCache, "minio", false, "Use Minio for caching")
	flag.StringVar(&minioCache.Endpoint, "minio-endpoint", "", "Minio endpoint")
	flag.StringVar(&minioCache.AccessKeyID, "minio-access-key-id", "", "Minio access key ID")
	flag.StringVar(&minioCache.SecretAccessKey, "minio-secret-access-key", "", "Minio secret access key")
	flag.StringVar(&minioCache.Bucket, "minio-bucket", "", "Minio Bucket")
	flag.BoolVar(&minioCache.UseTLS, "minio-tls", true, "Use TLS to access Minio cache")
	flag.StringVar(&options.PrecacheCmd, "precache", "", "Run the following shell command before caching packages")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, version, commit, date)
		flag.PrintDefaults()
	}

	flag.Parse()
	return options, nil
}
