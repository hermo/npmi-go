package cmd

// RunShellCommand executes a shell command
func (r *defaultRunner) RunShellCommand(commandLine string) (stdout string, stderr string, err error) {
	return r.RunCommand("sh", "-c", commandLine)
}
