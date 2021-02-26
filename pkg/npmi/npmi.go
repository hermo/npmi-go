package npmi

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// DeterminePlatform determines the node platform
func DeterminePlatform() (string, error) {
	nodeBinary, err := exec.LookPath("node")
	if err != nil {
		log.Fatal("Can't find a node binary in path")
	}

	cmd := exec.Command(nodeBinary, "-p", `process.version + "-" + process.platform + "-" + process.arch`)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err = cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// HashFile attempts to hash a file using SHA-256
func HashFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return hashHandle(f)
}

// hashHandle does the actual hashing given a file handle
func hashHandle(handle io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, handle); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
