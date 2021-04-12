package cache

import (
	"context"
	"io"

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
	bucket          string
}

// NewMinioCache creates a new Minio Cache
func NewMinioCache(endpoint string, accessKeyID string, secretAccessKey string, bucket string, useTLS bool) *minioCache {
	return &minioCache{nil, endpoint, accessKeyID, secretAccessKey, useTLS, bucket}
}

// Dial connects to a Minio instance
func (cache *minioCache) Dial() error {
	minioClient, err := minio.New(cache.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cache.accessKeyID, cache.secretAccessKey, ""),
		Secure: cache.useTLS,
	})

	if err != nil {
		return err
	}
	cache.client = minioClient

	return nil
}

// Has determines whether or not Minio contains a given key
func (cache *minioCache) Has(key string) (bool, error) {
	_, err := cache.client.StatObject(context.Background(), cache.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		// Handle NoSuchKey error from Minio
		if serr, ok := err.(minio.ErrorResponse); ok {
			if serr.Code == "NoSuchKey" {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

// Put stores something in the cache
// TODO: Test with inputs larger than 128 MiB
func (cache *minioCache) Put(key string, reader io.Reader) error {
	_, err := cache.client.PutObject(
		context.Background(), cache.bucket, key, reader, -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})

	if err != nil {
		return err
	}
	return nil
}

// Get fetches something from the cache
func (cache *minioCache) Get(key string) (io.Reader, error) {
	return cache.client.GetObject(context.Background(), cache.bucket, key, minio.GetObjectOptions{})
}

func (cache *minioCache) String() string {
	return "minio"
}
