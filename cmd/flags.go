package cmd

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/caarlos0/env/v6"
	"github.com/hermo/npmi-go/pkg/npmi"
)

const (
	usage = `npmi-go %s, commit %s, built at %s.
npmi-go installs NPM packages from a cache to speed up repeating installations.

Supported caches:
-  local          Data is cached locally in a directory.
-  minio          Data is cached to a (shared) Minio instance.

When using both caches, the local one is accessed first.

USAGE:
 npmi-go [OPTIONS]

ENVIRONMENT VARIABLES:
Use the following env variables to set default options.

  NPMI_LOGLEVEL  Log level. One of info|debug|trace (Default: "info")
  NPMI_JSON      Use JSON for log output (Default: false)
  NPMI_VERBOSE   Verbose output. DEPRECATED
                 Please use NPMI_LOGLEVEL with 'debug' or 'trace'
  NPMI_FORCE     Force (re)installation of deps
  NPMI_PRECACHE  Pre-cache command
  NPMI_TEMP_DIR  Use specified temp directory when creating archives (Default: system temp)

Local cache:
  NPMI_LOCAL      Use local cache
  NPMI_LOCAL_DIR  Local cache directory (Default: system temp)

Minio cache:
  NPMI_MINIO                    Use Minio cache
  NPMI_MINIO_ENDPOINT           Minio endpoint URL
  NPMI_MINIO_ACCESS_KEY_ID      Minio access key ID
  NPMI_MINIO_SECRET_ACCESS_KEY  Minio secret access key
  NPMI_MINIO_BUCKET             Minio bucket name
  NPMI_MINIO_TLS                Use TLS when connection to minio
  NPMI_MINIO_TLS_INSECURE       Disable TLS certificate checks

OPTIONS:
`
)

// ParseFlags parses command line flags
func ParseFlags() (*npmi.Options, error) {
	options := &npmi.Options{
		LogLevel: npmi.Info,
		Force:    false,

		UseLocalCache:   true,
		UseMinioCache:   false,
		PrecacheCommand: "",
		TempDir:         os.TempDir(),
	}
	localCache := &npmi.LocalCacheOptions{
		Dir: os.TempDir(),
	}
	minioCache := &npmi.MinioCacheOptions{
		Endpoint:        "",
		AccessKeyID:     "",
		SecretAccessKey: "",
		Bucket:          "",
		UseTLS:          true,
		InsecureTLS:     false,
	}
	options.LocalCache = localCache
	options.MinioCache = minioCache

	logLevelParser := func(v string) (interface{}, error) {
		level := npmi.LogLevelFromString(v)
		if level == npmi.NoLevel {
			return nil, fmt.Errorf("invalid loglevel '%s'", v)
		}
		return level, nil
	}

	if err := env.ParseWithFuncs(options, map[reflect.Type]env.ParserFunc{
		reflect.TypeOf(npmi.LogLevel(0)): logLevelParser,
	}); err != nil {
		return nil, fmt.Errorf("could not parse env options: %+v", err)
	}

	if err := env.Parse(localCache); err != nil {
		return nil, fmt.Errorf("could not parse env options: %+v", err)
	}

	if err := env.Parse(minioCache); err != nil {
		return nil, fmt.Errorf("could not parse env options: %+v", err)
	}

	flag.BoolVar(&options.Verbose, "verbose", options.Verbose, "Verbose output, DEPRECATED\nPlease use -loglevel with 'debug' or 'trace'")
	flag.BoolVar(&options.Force, "force", options.Force, "Force (re)installation of NPM deps and update cache(s)")
	flag.BoolVar(&options.UseLocalCache, "local", options.UseLocalCache, "Use local cache")
	flag.String("loglevel", "info", "Log level. One of info|debug|trace")
	flag.BoolVar(&options.Json, "json", options.Json, "Use JSON output")
	flag.StringVar(&localCache.Dir, "local-dir", options.LocalCache.Dir, "Local cache directory")
	flag.BoolVar(&options.UseMinioCache, "minio", options.UseMinioCache, "Use Minio for caching")
	flag.StringVar(&minioCache.Endpoint, "minio-endpoint", minioCache.Endpoint, "Minio endpoint")
	flag.StringVar(&minioCache.AccessKeyID, "minio-access-key-id", minioCache.AccessKeyID, "Minio access key ID")
	flag.StringVar(&minioCache.SecretAccessKey, "minio-secret-access-key", minioCache.SecretAccessKey, "Minio secret access key")
	flag.StringVar(&minioCache.Bucket, "minio-bucket", minioCache.Bucket, "Minio Bucket")
	flag.BoolVar(&minioCache.UseTLS, "minio-tls", minioCache.UseTLS, "Use TLS to access Minio cache")
	flag.BoolVar(&minioCache.InsecureTLS, "minio-tls-insecure", minioCache.InsecureTLS, "Disable TLS certificate checks")
	flag.StringVar(&options.PrecacheCommand, "precache", options.PrecacheCommand, "Run the following shell command before caching packages")
	flag.StringVar(&options.TempDir, "temp-dir", options.TempDir, "Temporary directory for archive creation")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage, npmi.Version, npmi.Commit, npmi.CommitDate)
		flag.PrintDefaults()
	}

	flag.Parse()
	if err := parseLogLevel(options); err != nil {
		return nil, err
	}

	return options, nil
}

func parseLogLevel(options *npmi.Options) (err error) {
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "loglevel" {
			value := f.Value.String()
			options.LogLevel = npmi.LogLevelFromString(value)

			if options.LogLevel == npmi.NoLevel {
				err = fmt.Errorf("could not parse command line args: invalid loglevel '%s'", value)
			}
		}
	})

	// Allow the deprecated Verbose flag to override everything
	if options.Verbose {
		options.LogLevel = npmi.Trace
	}
	return err
}
