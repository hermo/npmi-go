package npmi

import "github.com/hermo/npmi-go/pkg/cmd"

type NpmInstaller struct {
	npmBinary      string
	productionMode bool
	runner         cmd.Runner
}

func NewNpmInstaller(config *Config) *NpmInstaller {
	return &NpmInstaller{
		npmBinary:      config.npmBinary,
		productionMode: config.productionMode,
		runner:         config.runner,
	}
}

// Run installs packages from NPM
func (i *NpmInstaller) Run() (stdout string, stderr string, err error) {
	if i.productionMode {
		return i.runner.RunCommand(i.npmBinary, "ci", "--production", "--loglevel", "error", "--progress", "false")
	} else {
		return i.runner.RunCommand(i.npmBinary, "ci", "--dev", "--loglevel", "error", "--progress", "false")
	}
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func (i *NpmInstaller) RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	return i.runner.RunShellCommand(commandLine)
}
