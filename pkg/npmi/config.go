package npmi

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hermo/npmi-go/pkg/cmd"
)

type Config struct {
	nodeBinary     string
	npmBinary      string
	Platform       string
	productionMode bool
	runner         cmd.Runner
}

type configBuilder struct {
	nodeBinary                 string
	npmBinary                  string
	shouldFindBinariesInPath   bool
	productionModeDeterminator func() bool
	runner                     cmd.Runner
}

func NewConfigBuilder() *configBuilder {
	return &configBuilder{
		productionModeDeterminator: defaultProductionModeDeterminator,
		runner:                     cmd.NewRunner(),
	}
}

func (b *configBuilder) WithNodeAndNpmFromPath() {
	b.shouldFindBinariesInPath = true
}

func (b *configBuilder) WithNodeBinary(nodeBinary string) {
	b.nodeBinary = nodeBinary
}

func (b *configBuilder) WithNpmBinary(npmBinary string) {
	b.npmBinary = npmBinary
}

func (b *configBuilder) WithRunner(runner cmd.Runner) {
	b.runner = runner
}

func (b *configBuilder) WithProductionModeDeterminatorFunc(determinator func() bool) {
	b.productionModeDeterminator = determinator
}

func (b *configBuilder) Build() (*Config, error) {
	if b.shouldFindBinariesInPath {
		var err error
		b.nodeBinary, b.npmBinary, err = findNodeBinariesInPath()
		return nil, err
	}
	productionMode := b.productionModeDeterminator()
	platform, err := getPlatform(b.runner, b.nodeBinary, productionMode)
	if err != nil {
		return nil, err
	}
	return &Config{
		nodeBinary: b.nodeBinary,
		npmBinary:  b.npmBinary,
		runner:     b.runner,
		Platform:   platform,
	}, nil
}

func findNodeBinariesInPath() (nodePath string, npmPath string, err error) {
	nodePath, err = exec.LookPath("node")
	if err != nil {
		return "", "", err
	}

	npmPath, err = exec.LookPath("npm")
	if err != nil {
		return "", "", err
	}
	return
}

func getPlatform(runner cmd.Runner, nodeBinary string, productionMode bool) (string, error) {
	platform, err := determineNodeVersion(runner, nodeBinary)
	if err != nil {
		return "", err
	}

	if productionMode {
		platform += "-prod"
	} else {
		platform += "-dev"
	}

	return platform, nil
}

func determineNodeVersion(runner cmd.Runner, nodeBinary string) (string, error) {
	version, stdErr, err := runner.RunCommand(nodeBinary, "-p", `process.version + "-" + process.platform + "-" + process.arch`)
	if err != nil {
		return stdErr, fmt.Errorf("can't run node from \"%s\": %v", nodeBinary, err)
	}
	return version, nil
}

// defaultProductionModeDeterminator determines whether or not Node is running in production mode
func defaultProductionModeDeterminator() bool {
	return os.Getenv("NODE_ENV") == "production"
}
