package cmd

// RunShellCommand executes a shell command
func RunShellCommand(commandLine string) (stdout string, stderr string, err error) {
	return RunCommand("sh", "-c", commandLine)
}
