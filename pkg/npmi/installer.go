package npmi

type Installer struct {
	config *Config
}

func NewInstaller(config *Config) *Installer {
	return &Installer{
		config: config,
	}
}

// Run installs packages from NPM
func (n *Installer) Run() (stdout string, stderr string, err error) {
	if n.config.productionMode {
		return n.config.runner.RunCommand(n.config.npmBinary, "ci", "--production", "--loglevel", "error", "--progress", "false")
	} else {
		return n.config.runner.RunCommand(n.config.npmBinary, "ci", "--dev", "--loglevel", "error", "--progress", "false")
	}
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func (n *Installer) RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	return n.config.runner.RunShellCommand(commandLine)
}
