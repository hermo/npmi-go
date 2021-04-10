package npmi

import (
	"os"
	"os/exec"

	"github.com/hermo/npmi-go/pkg/cmd"
)

type NodeConfig struct {
	NodeBinary     string
	NpmBinary      string
	ProductionMode bool
	Runner         cmd.Runner
}

func NewNodeConfig() *NodeConfig {
	return &NodeConfig{
		Runner:         cmd.NewRunner(),
		ProductionMode: isNodeInProductionMode(),
	}
}

// Run makes sure that required Node.js binaries are present
func (n *NodeConfig) Run() error {
	var err error
	n.NodeBinary, err = exec.LookPath("node")
	if err != nil {
		return err
	}

	n.NpmBinary, err = exec.LookPath("npm")
	if err != nil {
		return err
	}
	return nil
}

// GetPlatform determines the Node.js runtime platform and mode
func (n *NodeConfig) GetPlatform() (string, error) {
	env, stdErr, err := n.Runner.RunCommand(n.NodeBinary, "-p", `process.version + "-" + process.platform + "-" + process.arch`)
	if err != nil {
		return stdErr, err
	}

	if n.ProductionMode {
		env += "-prod"
	} else {
		env += "-dev"
	}

	return env, nil
}

type Installer struct {
	config *NodeConfig
	Runner cmd.Runner
}

func NewInstaller(config *NodeConfig) *Installer {
	return &Installer{
		config: config,
		Runner: cmd.NewRunner(),
	}
}

// isNodeInProductionMode determines whether or not Node is running in production mode
func isNodeInProductionMode() bool {
	return os.Getenv("NODE_ENV") == "production"
}

// Run installs packages from NPM
func (n *Installer) Run() (stdout string, stderr string, err error) {
	if n.config.ProductionMode {
		return n.Runner.RunCommand(n.config.NpmBinary, "ci", "--production", "--loglevel", "error", "--progress", "false")
	} else {
		return n.Runner.RunCommand(n.config.NpmBinary, "ci", "--dev", "--loglevel", "error", "--progress", "false")
	}
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func (n *Installer) RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	return n.Runner.RunShellCommand(commandLine)
}
