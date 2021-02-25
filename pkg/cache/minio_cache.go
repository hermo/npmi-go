package cache

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioCache represents a Minio Cache instance
type MinioCache struct {
	client *minio.Client
	bucket string
}

// NewMinioCache creates a new Minio Cache
func NewMinioCache() *MinioCache {
	return &MinioCache{bucket: "mah-bucket"}
}

// Dial connects to a Minio instance
func (cache *MinioCache) Dial(endpoint string, accessKeyID string, secretAccessKey string, useTLS bool) error {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useTLS,
	})

	if err != nil {
		return err
	}
	cache.client = minioClient

	return nil
}

// Has determines whether or not Minio contains a given key
func (cache *MinioCache) Has(key string) (bool, error) {
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
	// fmt.Println(objInfo)
	return true, nil
}

// Put stores something in the cache
// TODO: Test with inputs larger than 128 MiB
func (cache *MinioCache) Put(key string, reader io.Reader) error {
	_, err := cache.client.PutObject(
		context.Background(), cache.bucket, key, reader, -1,
		minio.PutObjectOptions{ContentType: "application/octet-stream"})

	if err != nil {
		return err
	}
	//fmt.Println("Successfully uploaded bytes: ", uploadInfo)
	return nil
}

// Get fetches something from the cache
func (cache *MinioCache) Get(key string) (io.Reader, error) {
	return cache.client.GetObject(context.Background(), cache.bucket, key, minio.GetObjectOptions{})
}
