package cache

import (
	"errors"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/minio/minio-go/v7"
)

func TestMinioCacheIsCacheMiss(t *testing.T) {
	c := &minioCache{log: hclog.NewNullLogger()}

	if !c.IsCacheMiss(minio.ErrorResponse{Code: "NoSuchKey"}) {
		t.Fatalf("expected NoSuchKey to be treated as cache miss")
	}

	if !c.IsCacheMiss(minio.ErrorResponse{Code: "NotFound"}) {
		t.Fatalf("expected NotFound to be treated as cache miss")
	}

	if !c.IsCacheMiss(minio.ErrorResponse{StatusCode: 404}) {
		t.Fatalf("expected HTTP 404 to be treated as cache miss")
	}

	if c.IsCacheMiss(minio.ErrorResponse{Code: "AccessDenied", StatusCode: 403}) {
		t.Fatalf("expected non-not-found MinIO error to not be treated as cache miss")
	}

	if c.IsCacheMiss(errors.New("boom")) {
		t.Fatalf("expected generic error to not be treated as cache miss")
	}
}
