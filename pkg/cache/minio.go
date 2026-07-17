package cache

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"

	"github.com/hashicorp/go-hclog"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const maxSinglePutObjectSize = int64(5 * 1024 * 1024 * 1024)

// minioCache represents a Minio Cache instance
type minioCache struct {
	client          *minio.Client
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	useTLS          bool
	insecureTLS     bool
	bucket          string
	log             hclog.Logger
}

// NewMinioCache creates a new Minio Cache
func NewMinioCache(endpoint string, accessKeyID string, secretAccessKey string, bucket string, useTLS bool, insecureTLS bool, log hclog.Logger) *minioCache {
	return &minioCache{nil, endpoint, accessKeyID, secretAccessKey, useTLS, insecureTLS, bucket, log}
}

// Dial connects to a Minio instance
func (cache *minioCache) Dial() error {
	var transport http.RoundTripper
	if cache.insecureTLS {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		transport = http.DefaultTransport
	}

	minioClient, err := minio.New(cache.endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(cache.accessKeyID, cache.secretAccessKey, ""),
		Secure:    cache.useTLS,
		Transport: transport,
	})

	if err != nil {
		return err
	}
	cache.client = minioClient

	return nil
}

// Has determines whether or not Minio contains a given key
func (cache *minioCache) Has(key string) (bool, error) {
	log := cache.log.Named("has")
	log.Trace("start", "key", key)
	_, err := cache.client.StatObject(context.Background(), cache.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		// Handle NoSuchKey error from Minio
		if serr, ok := err.(minio.ErrorResponse); ok {
			if serr.Code == "NoSuchKey" {
				log.Trace("complete", "found", false)
				return false, nil
			}
		}
		log.Error("failed", "error", err)
		return false, err
	}
	log.Trace("complete", "found", true)
	return true, nil
}

// Put stores something in the cache
// TODO: Test with inputs larger than 128 MiB
func (cache *minioCache) Put(key string, reader io.Reader) error {
	log := cache.log.Named("put")
	log.Trace("start", "key", key)

	size := int64(-1)
	if seeker, ok := reader.(io.Seeker); ok {
		current, err := seeker.Seek(0, io.SeekCurrent)
		if err == nil {
			end, err := seeker.Seek(0, io.SeekEnd)
			if err == nil {
				size = end - current
			}
			_, _ = seeker.Seek(current, io.SeekStart)
		}
	}

	options := minio.PutObjectOptions{ContentType: "application/octet-stream"}
	if size >= 0 && size <= maxSinglePutObjectSize {
		// Prefer single PUT for known small/medium objects to avoid multipart edge cases in some S3-compatible backends.
		options.DisableMultipart = true
	}

	_, err := cache.client.PutObject(
		context.Background(), cache.bucket, key, reader, size, options)

	if err != nil {
		log.Error("failed", "error", err)
		return err
	}
	log.Trace("complete")
	return nil
}

// Get fetches something from the cache
func (cache *minioCache) Get(key string) (io.Reader, error) {
	log := cache.log.Named("put")
	log.Trace("start", "key", key)
	return cache.client.GetObject(context.Background(), cache.bucket, key, minio.GetObjectOptions{})
}

// IsCacheMiss maps MinIO/S3 not-found responses to cache misses.
func (cache *minioCache) IsCacheMiss(err error) bool {
	response := minio.ToErrorResponse(err)
	return response.Code == "NoSuchKey" || response.Code == "NotFound" || response.StatusCode == http.StatusNotFound
}

func (cache *minioCache) String() string {
	return "minio"
}
