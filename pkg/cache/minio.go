package cache

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/hashicorp/go-hclog"
)

const s3Region = "us-east-1"

// minioCache represents an S3-backed cache that used to be implemented with Minio client
// but Minio client was deprecated, so now this uses AWS SDK
type minioCache struct {
	client          *s3.Client
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
	return &minioCache{
		endpoint:        endpoint,
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		useTLS:          useTLS,
		insecureTLS:     insecureTLS,
		bucket:          bucket,
		log:             log,
	}
}

// Dial configures a client for the S3-compatible endpoint.
func (cache *minioCache) Dial() error {
	scheme := "http"
	if cache.useTLS {
		scheme = "https"
	}

	endpointURL, err := url.Parse(fmt.Sprintf("%s://%s", scheme, cache.endpoint))
	if err != nil {
		return fmt.Errorf("invalid S3 endpoint %q: %w", cache.endpoint, err)
	}
	if endpointURL.Host == "" || endpointURL.User != nil ||
		(endpointURL.Path != "" && endpointURL.Path != "/") ||
		endpointURL.RawQuery != "" || endpointURL.Fragment != "" {
		return fmt.Errorf("invalid S3 endpoint %q", cache.endpoint)
	}
	endpointURL.Path = ""

	var transport http.RoundTripper = http.DefaultTransport
	if cache.insecureTLS {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	credentials := aws.NewCredentialsCache(aws.CredentialsProviderFunc(
		func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cache.accessKeyID,
				SecretAccessKey: cache.secretAccessKey,
				Source:          "npmi-go",
			}, nil
		},
	))

	cache.client = s3.New(s3.Options{
		BaseEndpoint: aws.String(endpointURL.String()),
		Credentials:  credentials,
		HTTPClient: &http.Client{
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		Region:                     s3Region,
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
		UsePathStyle:               true,
	})

	return nil
}

// Has determines whether or not Minio contains a given key
func (cache *minioCache) Has(key string) (bool, error) {
	log := cache.log.Named("has")
	log.Trace("start", "key", key)

	_, err := cache.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(cache.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var apiError interface{ ErrorCode() string }
		if errors.As(err, &apiError) && (apiError.ErrorCode() == "NotFound" || apiError.ErrorCode() == "NoSuchKey") {
			log.Trace("complete", "found", false)
			return false, nil
		}
		log.Error("failed", "error", err)
		return false, err
	}

	log.Trace("complete", "found", true)
	return true, nil
}

// Put stores something in the cache
func (cache *minioCache) Put(key string, reader io.Reader) error {
	log := cache.log.Named("put")
	log.Trace("start", "key", key)

	uploader := transfermanager.New(cache.client, func(options *transfermanager.Options) {
		options.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})
	_, err := uploader.UploadObject(context.Background(), &transfermanager.UploadObjectInput{
		Body:        reader,
		Bucket:      aws.String(cache.bucket),
		ContentType: aws.String("application/octet-stream"),
		Key:         aws.String(key),
	})
	if err != nil {
		log.Error("failed", "error", err)
		return err
	}

	log.Trace("complete")
	return nil
}

// Get fetches something from the cache
func (cache *minioCache) Get(key string) (io.Reader, error) {
	log := cache.log.Named("get")
	log.Trace("start", "key", key)

	output, err := cache.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(cache.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Error("failed", "error", err)
		return nil, err
	}

	log.Trace("complete")
	return output.Body, nil
}

func (cache *minioCache) String() string {
	return "minio"
}
