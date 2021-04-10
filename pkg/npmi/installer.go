package npmi

import (
	"os/exec"

	"github.com/hermo/npmi-go/pkg/cmd"
)

type Installer struct {
	nodeBinary string
	npmBinary  string
	Runner     cmd.Runner
}

func NewInstaller() *Installer {
	return &Installer{
		Runner: cmd.NewRunner(),
	}
}

// DeterminePlatformKey determines the Node.js runtime platform and mode
func (n *Installer) DeterminePlatformKey() (string, error) {
	env, stdErr, err := n.Runner.RunCommand(n.nodeBinary, "-p", `process.version + "-" + process.platform + "-" + process.arch`)
	if err != nil {
		return stdErr, err
	}

	if isNodeInProductionMode() {
		env += "-prod"
	} else {
		env += "-dev"
	}

	return env, nil
}

// LocateRequiredBinaries makes sure that required Node.js binaries are present
func (n *Installer) LocateRequiredBinaries() error {
	var err error
	n.nodeBinary, err = exec.LookPath("node")
	if err != nil {
		return err
	}

	n.npmBinary, err = exec.LookPath("npm")
	if err != nil {
		return err
	}
	return nil
}

// InstallPackages installs packages from NPM
func (n *Installer) InstallPackages() (stdout string, stderr string, err error) {
	if isNodeInProductionMode() {
		return n.Runner.RunCommand(n.npmBinary, "ci", "--production", "--loglevel", "error", "--progress", "false")
	} else {
		return n.Runner.RunCommand(n.npmBinary, "ci", "--dev", "--loglevel", "error", "--progress", "false")
	}
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func (n *Installer) RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	return n.Runner.RunShellCommand(commandLine)
}
