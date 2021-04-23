package npmi

// SpyRunner is a cmd.Runner for testing purposes
type SpyRunner struct {
	Stdout string
	Stderr string
	Error  error
}

func (r *SpyRunner) RunCommand(name string, args ...string) (stdout string, stderr string, err error) {
	return r.Stdout, r.Stderr, r.Error
}

func (r *SpyRunner) RunShellCommand(commandLine string) (stdout string, stderr string, err error) {
	return r.Stdout, r.Stderr, r.Error
}
