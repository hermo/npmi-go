package npmi

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"
)

// isNodeInProductionMode determines whether or not Node is running in production mode
func isNodeInProductionMode() bool {
	return os.Getenv("NODE_ENV") == "production"
}

// HashFile hashes a file using SHA-256
func HashFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return hashInput(f)
}

// HashString hashes a given string using SHA-256
func HashString(str string) (string, error) {
	return hashInput(strings.NewReader(str))
}

func hashInput(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
