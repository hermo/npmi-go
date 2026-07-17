package npmi

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type testCacherWithoutClassifier struct{}

func (c *testCacherWithoutClassifier) Has(key string) (bool, error) { return false, nil }
func (c *testCacherWithoutClassifier) Put(key string, reader io.Reader) error {
	return nil
}
func (c *testCacherWithoutClassifier) Get(key string) (io.Reader, error) {
	return strings.NewReader(""), nil
}

type testCacherWithClassifier struct{
	miss bool
}

func (c *testCacherWithClassifier) Has(key string) (bool, error) { return false, nil }
func (c *testCacherWithClassifier) Put(key string, reader io.Reader) error {
	return nil
}
func (c *testCacherWithClassifier) Get(key string) (io.Reader, error) {
	return strings.NewReader(""), nil
}
func (c *testCacherWithClassifier) IsCacheMiss(err error) bool { return c.miss }

func TestIsCacheMissUsesClassifierWhenAvailable(t *testing.T) {
	err := errors.New("test error")

	if !isCacheMiss(&testCacherWithClassifier{miss: true}, err) {
		t.Fatalf("expected classifier result to be used")
	}

	if isCacheMiss(&testCacherWithClassifier{miss: false}, err) {
		t.Fatalf("expected classifier false result to be used")
	}
}

func TestIsCacheMissReturnsFalseWithoutClassifier(t *testing.T) {
	err := errors.New("test error")
	if isCacheMiss(&testCacherWithoutClassifier{}, err) {
		t.Fatalf("expected false when cacher does not implement classifier")
	}
}
