package cmd

// RunShellCommand executes a shell command
func RunShellCommand(commandLine string) (stdout string, stderr string, err error) {
	return RunCommand(`C:\windows\system32\cmd.exe`, "/C", commandLine)
}
