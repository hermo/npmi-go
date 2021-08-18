package hash

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

// File hashes a file using SHA-256
func File(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return hashInput(f)
}

// String hashes a given string using SHA-256
func String(str string) (string, error) {
	return hashInput(strings.NewReader(str))
}

func hashInput(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
