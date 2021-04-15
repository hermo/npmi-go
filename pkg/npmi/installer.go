package npmi

type Installer struct {
	config *NodeConfig
}

func NewInstaller(config *NodeConfig) *Installer {
	return &Installer{
		config: config,
	}
}

// Run installs packages from NPM
func (n *Installer) Run() (stdout string, stderr string, err error) {
	if n.config.ProductionMode {
		return n.config.Runner.RunCommand(n.config.NpmBinary, "ci", "--production", "--loglevel", "error", "--progress", "false")
	} else {
		return n.config.Runner.RunCommand(n.config.NpmBinary, "ci", "--dev", "--loglevel", "error", "--progress", "false")
	}
}

// RunPrecacheCommand runs a given command before inserting freshly installed NPM deps into cache
func (n *Installer) RunPrecacheCommand(commandLine string) (stdout string, stderr string, err error) {
	return n.config.Runner.RunShellCommand(commandLine)
}
