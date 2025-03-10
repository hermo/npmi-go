package npmi

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hermo/npmi-go/pkg/cmd"
)

type NpmInstaller struct {
	npmBinary      string
	runner         cmd.Runner
	log            hclog.Logger
}

func NewNpmInstaller(config *Config, log hclog.Logger) *NpmInstaller {
	return &NpmInstaller{
		npmBinary:      config.npmBinary,
		runner:         config.runner,
		log:            log,
	}
}

// Run installs packages from NPM
func (i *NpmInstaller) Run() (stdout string, stderr string, err error) {
	var args = []string{"ci", "--loglevel", "error", "--progress", "false"}

	i.log.Trace("Running", "npmBinary", i.npmBinary, "args", args)
	return i.runner.RunCommand(i.npmBinary, args...)
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func (i *NpmInstaller) RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	i.log.Trace("Running shell", "commandLine", commandLine)
	return i.runner.RunShellCommand(commandLine)
}
