[![CodeQL](https://github.com/hermo/npmi-go/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/hermo/npmi-go/actions/workflows/codeql-analysis.yml)

# npmi-go

npmi-go caches the contents of node_modules directory in a tarball stored
locally or in a Minio instance. The Node runtime environment and a hash of
package-lock.json is used as the cache key.

The cache key is something like
`v12.16.3-darwin-x64-dev-78c49bbaba2e4002e313e55018716d9a673fa99f1e676afcb03df0a902f4883f`.

![Diagram describing how npmi-go works](npmi-go.svg)

# Not for production (yet)

Note that npmi-go is work-in-progress and should not be used in production.

# Installation

- Download a prebuilt binary from the [releases page](https://github.com/hermo/npmi-go/releases/latest)

# Building

## Quick & dirty

Just run `go build` in the root directory to build `npmi-go` binary.

## Goreleaser

Dev builds for your system only may be built by using goreleaser:

```
goreleaser build --snapshot --clean --single-target
```

The output will appear under `dist/`.

Creating a symlink with a shorter name is recommended for testing:
`ln -s dist/npmi-go_linux_amd64/npmi-go`

# Releasing

1. Create a git tag with the desired version

```
git tag v0.4.0
```

2. Create a release build to see if everything is setup correctly

```
goreleaser release --clean --skip=publish
```

3. If everything seems OK, release to Github by running

```
goreleaser release --clean
```

# Supported Caches

## Local

The local cache is a directory containing tarballs.

It is enabled by default and caches to the system temp directory.

To disable the local cache use the flag `-local=0`.

See the `-local*` options in usage for more info.

## Minio

A [Minio](https://min.io/) instance may also be used as a cache backend.

Using Minio allows several clients to share the same cache. CI systems often
install the same deps over and over again and using a shared cache will
reduce bandwidth, IO and CPU usage significantly.

### Security notice

Note that the contents of a tarball in the cache are not checked in any way
and only trusted systems should be allowed to access the shared cache.

### Testing with Minio

1. Start a temporary Minio instance using Docker:

```
docker run --rm --name minio \
-p 9000:9000 -p 9001:9001 \
-e "MINIO_ROOT_USER=minio" \
-e "MINIO_ROOT_PASSWORD=password" \
-v "$PWD/test/minio/certs:/root/.minio/certs" \
minio/minio \
server /data --console-address ":9001"
```

2. Edit `test/minio/config.json` and change the IP to match whatever Minio reports.
3. Create a bucket called `npmi` using Minio Client:

```
docker run -v "$PWD/test/minio/config.json:/root/.mc/config.json" minio/mc --insecure mb minio/npmi
```

4. Use the dummy project under `testdata/` and run npmi-go using minio:

```
cd testdata
../npmi-go -loglevel trace \
-minio=1 \
-minio-endpoint=localhost:9000 \
-minio-bucket=npmi \
-minio-access-key-id=minio \
-minio-secret-access-key=password \
-minio-tls-insecure \
-local=0
```

Note that we disable the local cache for testing purposes.

```
2024-02-14T13:13:16.176+0200 [INFO]  npmi: Starting installation: version=dev
2024-02-14T13:13:16.176+0200 [TRACE] npmi.cache: start: cacheKey=v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26
2024-02-14T13:13:16.176+0200 [TRACE] npmi.cache.minio.lookup: start
2024-02-14T13:13:16.176+0200 [TRACE] npmi.cache.minio.has: start: key=v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26
2024-02-14T13:13:16.181+0200 [TRACE] npmi.cache.minio.has: complete: found=false
2024-02-14T13:13:16.181+0200 [TRACE] npmi.cache.minio.lookup: complete
2024-02-14T13:13:16.181+0200 [DEBUG] npmi.cache.minio.lookup: cache MISS
2024-02-14T13:13:16.181+0200 [TRACE] npmi.cache: complete
2024-02-14T13:13:16.181+0200 [TRACE] npmi.install: start
2024-02-14T13:13:16.181+0200 [TRACE] npmi.installPackages: start
2024-02-14T13:13:16.181+0200 [TRACE] npmi.npmInstaller: Running: npmBinary=/usr/bin/npm args=["ci", "--dev", "--loglevel", "error", "--progress", "false"]
2024-02-14T13:13:16.814+0200 [TRACE] npmi.installPackages: complete: stdout="added 2 packages, and audited 3 packages in 492ms\n\nfound 0 vulnerabilities"
2024-02-14T13:13:16.814+0200 [TRACE] npmi.installPackages: complete
2024-02-14T13:13:16.814+0200 [TRACE] npmi.createArchive: start
2024-02-14T13:13:16.814+0200 [DEBUG] npmi.createArchive: Creating archive: path=/tmp/modules-v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26.tar.gz
2024-02-14T13:13:16.821+0200 [TRACE] npmi.createArchive: complete
2024-02-14T13:13:16.821+0200 [TRACE] npmi.cacheArchive: start
2024-02-14T13:13:16.821+0200 [TRACE] npmi.cacheArchive.minio: start
2024-02-14T13:13:16.821+0200 [TRACE] npmi.cache.minio.put: start: key=v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26
2024-02-14T13:13:16.848+0200 [TRACE] npmi.cache.minio.put: complete
2024-02-14T13:13:16.848+0200 [TRACE] npmi.cacheArchive.minio: complete
2024-02-14T13:13:16.848+0200 [TRACE] npmi.cacheArchive: complete
2024-02-14T13:13:16.848+0200 [DEBUG] npmi.createArchive: Removed temporary archive: path=/tmp/modules-v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26.tar.gz
2024-02-14T13:13:16.848+0200 [TRACE] npmi: complete
2024-02-14T13:13:16.848+0200 [INFO]  npmi: Installation complete
```

Now run the same `npmi-go` command again. The output will be something like:

```
2024-02-14T13:14:37.014+0200 [INFO]  npmi: Starting installation: version=dev
2024-02-14T13:14:37.014+0200 [TRACE] npmi.cache: start: cacheKey=v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26
2024-02-14T13:14:37.014+0200 [TRACE] npmi.cache.minio.lookup: start
2024-02-14T13:14:37.014+0200 [TRACE] npmi.cache.minio.has: start: key=v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26
2024-02-14T13:14:37.021+0200 [TRACE] npmi.cache.minio.has: complete: found=true
2024-02-14T13:14:37.021+0200 [TRACE] npmi.cache.minio.lookup: complete
2024-02-14T13:14:37.021+0200 [DEBUG] npmi.cache.minio.lookup: cache HIT
2024-02-14T13:14:37.021+0200 [TRACE] npmi.cache.minio.fetch: start
2024-02-14T13:14:37.021+0200 [TRACE] npmi.cache.minio.put: start: key=v20.10.0-linux-x64-dev-b7782c38ef77fe9874c3c30e7a0ba49ce90c8a3e394df9b775db4e502ec19f26
2024-02-14T13:14:37.021+0200 [TRACE] npmi.cache.minio.fetch: complete
2024-02-14T13:14:37.021+0200 [TRACE] npmi.cache.minio.extract: start
2024-02-14T13:14:37.030+0200 [TRACE] npmi.cache.minio.extract.cleanup: start
2024-02-14T13:14:37.030+0200 [TRACE] npmi.cache.minio.extract.cleanup: complete: numFilesRemoved=0 filesRemoved=[]
2024-02-14T13:14:37.030+0200 [TRACE] npmi.cache.minio.extract: complete
2024-02-14T13:14:37.030+0200 [DEBUG] npmi.cache.minio: packages successfully installed from cache
2024-02-14T13:14:37.030+0200 [TRACE] npmi.cache: complete
2024-02-14T13:14:37.030+0200 [TRACE] npmi: complete
2024-02-14T13:14:37.030+0200 [INFO]  npmi: Installation complete
```

See the `-minio*` options in usage for more info.

# Usage

```
npmi-go dev, commit none, built at unknown.
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

Tar file security hardening:
  NPMI_TAR_ABSOLUTE_PATHS           Allow absolute paths in tar archives (Default: true)
  NPMI_TAR_DOUBLE_DOT_PATHS         Allow double dot paths in tar archives (Default: true)
  NPMI_TAR_LINKS_OUTSIDE_CWD  Allow links outside of the current working directory (Default: true)

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
  -force
        Force (re)installation of NPM deps and update cache(s)
  -json
        Use JSON output
  -local
        Use local cache
  -local-dir string
        Local cache directory (default "/tmp")
  -loglevel string
        Log level. One of info|debug|trace (default "info")
  -minio
        Use Minio for caching (default true)
  -minio-access-key-id string
        Minio access key ID (default "EJjhQWkij3PlGwxVcN0n")
  -minio-bucket string
        Minio Bucket (default "npmi-go")
  -minio-endpoint string
        Minio endpoint (default "minio-npmi-go.ci.dev.verkkokauppa.com")
  -minio-secret-access-key string
        Minio secret access key (default "K1BlFkl1In8VsewpQzjX")
  -minio-tls
        Use TLS to access Minio cache (default true)
  -minio-tls-insecure
        Disable TLS certificate checks
  -precache string
        Run the following shell command before caching packages
  -tar-absolute-paths
        Allow absolute paths in tar archives (default true)
  -tar-double-dot-paths
        Allow double dot paths in tar archives (default true)
  -tar-links-outside-cwd
        Allow links outside of the current working directory (default true)
  -temp-dir string
        Temporary directory for archive creation (default "/tmp")
  -verbose
        Verbose output, DEPRECATED
        Please use -loglevel with 'debug' or 'trace'
```

## Configuration with .npmirc

npmi-go does not currently support a config file.

## Configuration with environment variables

The environment variables described before are used as defaults when present.

# Known Issues

## package-lock.json sync is not checked

npmi-go does not check is a package-lock.json file is in sync with package.lock.

## post-installation side effects outside node_modules/ will be ignored

Any post-installation script of NPM will NOT get run when installing
from cache. This includes at least the following:

- install
- postinstall
