package npmi

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

var (
	nodeBinary string
	npmBinary  string
)

// DeterminePlatform determines the node platform
func DeterminePlatform() (string, error) {
	if nodeBinary == "" {
		return "", fmt.Errorf("DeterminePlatform: InitNodeBinaries not run")
	}

	cmd := exec.Command(nodeBinary, "-p", `process.version + "-" + process.platform + "-" + process.arch`)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// InitNodeBinaries makes sure that required Node.JS binaries are present
func InitNodeBinaries() error {
	var err error
	nodeBinary, err = exec.LookPath("node")
	if err != nil {
		return err
	}

	npmBinary, err = exec.LookPath("npm")
	if err != nil {
		return err
	}
	return nil
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

// HashString hashes a given string
func HashString(str string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(str)); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// InstallPackages installs packages from NPM
func InstallPackages() (stdout string, stderr string, err error) {
	return runCommand(npmBinary, "ci")
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	// TODO: Consider adding Windows support
	return runCommand("sh", "-c", commandLine)
}

func runCommand(name string, args ...string) (stdout string, stderr string, err error) {
	cmd := exec.Command(name, args...)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	stdout = strings.TrimSpace(stdoutBuf.String())
	stderr = strings.TrimSpace(stderrBuf.String())
	return
}
