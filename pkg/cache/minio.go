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
	_, err := cache.client.PutObject(
		context.Background(), cache.bucket, key, reader, -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})

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

func (cache *minioCache) String() string {
	return "minio"
}
