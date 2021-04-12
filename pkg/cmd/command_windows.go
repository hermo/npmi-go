package cmd

// RunShellCommand executes a shell command
func (r *defaultRunner) RunShellCommand(commandLine string) (stdout string, stderr string, err error) {
	return r.RunCommand(`C:\windows\system32\cmd.exe`, "/C", commandLine)
}
