package main

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
}

const (
	usage = `usage: %s

Options:
`
)

// ParseFlags parses command line flags given
func ParseFlags() (*Options, error) {
	options := &Options{}
	localCache := &LocalCacheOptions{}
	minioCache := MinioCacheOptions{}
	options.LocalCache = localCache
	options.MinioCache = &minioCache

	flag.BoolVar(&options.Verbose, "verbose", false, "Verbose output (default: false)")
	flag.BoolVar(&options.Force, "force", false, "Force install & cache updates (default: false)")
	flag.BoolVar(&options.UseLocalCache, "local-cache", true, "Use local cache (default: true)")
	flag.StringVar(&localCache.Dir, "local-cache-dir", os.TempDir(), "Local cache directory")
	flag.BoolVar(&options.UseMinioCache, "minio-cache", false, "Use Minio cache")
	flag.StringVar(&minioCache.Endpoint, "minio-cache-endpoint", "", "Minio cache endpoint")
	flag.StringVar(&minioCache.AccessKeyID, "minio-cache-access-key-id", "", "Minio cache access key ID")
	flag.StringVar(&minioCache.SecretAccessKey, "minio-cache-secret-access-key", "", "Minio cache secret access key")
	flag.BoolVar(&minioCache.UseTLS, "minio-cache-tls", true, "Use TLS to access Minio cache (default: true)")

	flag.Usage = func() { // [4]
		fmt.Fprintf(flag.CommandLine.Output(), usage, os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	return options, nil
}
