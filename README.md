# npmi-go
npmi-go caches the contents of node_modules directory in a tarball stored
locally or in a Minio instance. The Node runtime environment and a hash of
package-lock.json is used as the cache key.

The cache key is something like
`v12.16.3-darwin-x64-78c49bbaba2e4002e313e55018716d9a673fa99f1e676afcb03df0a902f4883f`.

# Not for production (yet)
Note that npmi-go is work-in-progress and should not be used in production.

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

Start a temporary Minio instance using Docker:

`docker run --rm --name minio -p 9000:9000 minio/minio server /data`

See the output to determine the instance address and default credentials.

Log in to Minio and create a bucket (`npmi` in this example).

Create a dummy NPM project:

```
mkdir npmi-test && cd npmi-test
npm init -y
npm add is-odd
```

Now run npmi-go using minio:
```
npmi-go   -verbose \
          -minio=1 \
          -minio-endpoint=localhost:9000 \
          -minio-bucket=npmi \
          -minio-access-key-id=minioadmin \
          -minio-secret-access-key=minioadmin \
          -minio-tls=0 \
          -local=0
```

Note that we disable the local cache for testing purposes.

```npmi-go start
Lookup start, looking for key v12.16.3-darwin-x64-xxxx
Lookup(minio).Has start
Lookup(minio).Has complete: MISS
Lookup complete
Install start
Install(npm).InstallPackages start
Install(npm).InstallPackages complete: success: added 2 packages in 0.04s
Install complete
Archive start
Archive creating modules-v12.16.3-darwin-x64-xxxx.tar.gz
Archive complete
Cache start
Cache(minio).Open start
Cache(minio).Open complete
Cache(minio).Put start
Cache(minio).Put complete
Cache complete
npmi-go complete
Post-Archive: Removed temporary archive modules-v12.16.3-darwin-x64-xxxx.tar.gz
```

Now run the same `npmi-go` command again. The output will be something like:

```
npmi-go start
Lookup start, looking for key v12.16.3-darwin-x64-xxxx
Lookup(minio).Has start
Lookup(minio).Has complete: HIT
Lookup(minio).Get start
Lookup(minio).Get complete
Lookup(minio).Extract start
Cleanup start
Cleanup complete, 0 extraneous files removed
Lookup(minio).Extract complete
Lookup complete
npmi-go complete
```

See the `-minio*` options in usage for more info.
# Usage

```
npmi-go v0.0.0-SNAPSHOT-xxxx, commit xxxx, built at XXXX.
npmi-go installs NPM packages from a cache to speed up repeating installations.

Supported caches:
-  local		Data is cached locally in a directory.
-  minio		Data is cached to a (shared) Minio instance.

When using both caches, the local one is accessed first.

USAGE:
 npmi-go [OPTIONS]

OPTIONS:
  -force
    	Force (re)installation of NPM deps and update cache(s)
  -local
    	Use local cache (default true)
  -local-dir string
    	Local cache directory (default "$TEMP")
  -minio
    	Use Minio for caching
  -minio-access-key-id string
    	Minio access key ID
  -minio-bucket string
    	Minio Bucket
  -minio-endpoint string
    	Minio endpoint
  -minio-secret-access-key string
    	Minio secret access key
  -minio-tls
    	Use TLS to access Minio cache (default true)
  -precache string
    	Run the following shell command before caching packages
  -verbose
    	Verbose output
```

## Configuration with .npmirc

npmi-go does not currently support a config file.

## Configuration with environment variables

npmi-go does not currently support env variables.

# Known Issues

## The difference between dev and prod dependencies is not honored
The cache key does not currently take into account what kind of deps are installed.

## package-lock.json sync is not checked
npmi-go does not check is a package-lock.json file is in sync with package.lock.

## post-installation side effects outside node_modules/ will be ignored
Any post-installation script of NPM will NOT get run when installing
from cache. This includes at least the following:
- install
- postinstall
