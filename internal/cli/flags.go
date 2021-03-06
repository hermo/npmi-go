package cli

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
)

// MinioCacheOptions contains configuration for Minio Cache
type MinioCacheOptions struct {
	Endpoint        string `env:"NPMI_MINIO_ENDPOINT"`
	AccessKeyID     string `env:"NPMI_MINIO_ACCESS_KEY_ID"`
	SecretAccessKey string `env:"NPMI_MINIO_SECRET_ACCESS_KEY"`
	Bucket          string `env:"NPMI_MINIO_BUCKET"`
	UseTLS          bool   `env:"NPMI_MINIO_TLS"`
}

// LocalCacheOptions constains configuration for Local Cache
type LocalCacheOptions struct {
	Dir string `env:"NPMI_LOCAL_DIR"`
}

// Options describes the runtime configuration
type Options struct {
	Verbose       bool `env:"NPMI_VERBOSE"`
	Force         bool `env:"NPMI_FORCE"`
	UseLocalCache bool `env:"NPMI_LOCAL"`
	UseMinioCache bool `env:"NPMI_MINIO"`
	MinioCache    *MinioCacheOptions
	LocalCache    *LocalCacheOptions
	PrecacheCmd   string `env:"NPMI_PRECACHE"`
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const (
	usage = `npmi-go %s, commit %s, built at %s.
npmi-go installs NPM packages from a cache to speed up repeating installations.

Supported caches:
-  local          Data is cached locally in a directory.
-  minioData is cached to a (shared) Minio instance.

When using both caches, the local one is accessed first.

USAGE:
 npmi-go [OPTIONS]

ENVIRONMENT VARIABLES:
Use the following env variables to set default options.

  NPMI_VERBOSE   Verbose output
  NPMI_FORCE     Force (re)installation of deps
  NPMI_PRECACHE  Pre-cache command

Local cache:
  NPMI_LOCAL      Use local cache
  NPMI_LOCAL_DIR  Local cache directory

Minio cache:
  NPMI_MINIO                    Use Minio cache
  NPMI_MINIO_ENDPOINT           Minio endpoint URL
  NPMI_MINIO_ACCESS_KEY_ID      Minio access key ID
  NPMI_MINIO_SECRET_ACCESS_KEY  Minio secret access key
  NPMI_MINIO_BUCKET             Minio bucket name
  NPMI_MINIO_TLS                Use TLS when connection to minio

OPTIONS:
`
)

// ParseFlags parses command line flags
func ParseFlags() (*Options, error) {
	options := &Options{
		Verbose:       false,
		Force:         false,
		UseLocalCache: true,
		UseMinioCache: false,
		PrecacheCmd:   "",
	}
	localCache := &LocalCacheOptions{
		Dir: os.TempDir(),
	}
	minioCache := &MinioCacheOptions{
		Endpoint:        "",
		AccessKeyID:     "",
		SecretAccessKey: "",
		Bucket:          "",
		UseTLS:          true,
	}
	options.LocalCache = localCache
	options.MinioCache = minioCache

	if err := env.Parse(options); err != nil {
		log.Fatalf("Could not parse env options: %+v", err)
	}

	if err := env.Parse(localCache); err != nil {
		log.Fatalf("Could not parse env options: %+v", err)
	}

	if err := env.Parse(minioCache); err != nil {
		log.Fatalf("Could not parse env options: %+v", err)
	}

	flag.BoolVar(&options.Verbose, "verbose", options.Verbose, "Verbose output")
	flag.BoolVar(&options.Force, "force", options.Force, "Force (re)installation of NPM deps and update cache(s)")
	flag.BoolVar(&options.UseLocalCache, "local", options.UseLocalCache, "Use local cache")
	flag.StringVar(&localCache.Dir, "local-dir", options.LocalCache.Dir, "Local cache directory")
	flag.BoolVar(&options.UseMinioCache, "minio", options.UseMinioCache, "Use Minio for caching")
	flag.StringVar(&minioCache.Endpoint, "minio-endpoint", minioCache.Endpoint, "Minio endpoint")
	flag.StringVar(&minioCache.AccessKeyID, "minio-access-key-id", minioCache.AccessKeyID, "Minio access key ID")
	flag.StringVar(&minioCache.SecretAccessKey, "minio-secret-access-key", minioCache.SecretAccessKey, "Minio secret access key")
	flag.StringVar(&minioCache.Bucket, "minio-bucket", minioCache.Bucket, "Minio Bucket")
	flag.BoolVar(&minioCache.UseTLS, "minio-tls", minioCache.UseTLS, "Use TLS to access Minio cache")
	flag.StringVar(&options.PrecacheCmd, "precache", options.PrecacheCmd, "Run the following shell command before caching packages")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, version, commit, date)
		flag.PrintDefaults()
	}

	flag.Parse()
	return options, nil
}
