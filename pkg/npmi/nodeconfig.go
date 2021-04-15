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

// isNodeInProductionMode determines whether or not Node is running in production mode
func isNodeInProductionMode() bool {
	return os.Getenv("NODE_ENV") == "production"
}
