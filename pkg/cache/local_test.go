package cache

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestLocalCacheIsCacheMiss(t *testing.T) {
	c := &localCache{dir: t.TempDir(), log: hclog.NewNullLogger()}

	notExistErr := &os.PathError{Op: "open", Path: "missing", Err: os.ErrNotExist}
	if !c.IsCacheMiss(notExistErr) {
		t.Fatalf("expected not-exist error to be treated as cache miss")
	}

	if c.IsCacheMiss(errors.New("boom")) {
		t.Fatalf("expected generic error to not be treated as cache miss")
	}
}

func TestLocalCachePutReturnsCreateError(t *testing.T) {
	c := &localCache{dir: t.TempDir(), log: hclog.NewNullLogger()}

	err := c.Put("nested/path/file.tgz", strings.NewReader("content"))
	if err == nil {
		t.Fatalf("expected create error when parent directory does not exist")
	}
}
