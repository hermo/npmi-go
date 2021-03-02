package npmi

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/hermo/npmi-go/pkg/cmd"
)

var (
	nodeBinary string
	npmBinary  string
)

// DeterminePlatform determines the Node.js runtime platform and mode
func DeterminePlatform() (string, error) {
	if nodeBinary == "" {
		return "", fmt.Errorf("DeterminePlatform: InitNodeBinaries not run")
	}

	env, _, err := cmd.RunCommand(nodeBinary, "-p", `process.version + "-" + process.platform + "-" + process.arch`)
	if err != nil {
		return "", err
	}

	// NODE_ENV determines what kind of deps are being installed
	if os.Getenv("NODE_ENV") == "production" {
		env += "-prod"
	} else {
		env += "-dev"
	}

	return env, nil
}

// InitNodeBinaries makes sure that required Node.js binaries are present
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

// InstallPackages installs packages from NPM
func InstallPackages() (stdout string, stderr string, err error) {
	return cmd.RunCommand(npmBinary, "ci", "--dev", "--loglevel", "error", "--progress", "false")
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	return cmd.RunShellCommand(commandLine)
}
