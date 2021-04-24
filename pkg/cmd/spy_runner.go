package cmd

import "fmt"

type RunCommandCall struct {
	Name string
	Args []string
}

func (c *RunCommandCall) String() string {
	return fmt.Sprintf("Name: %+v, Args: %+v", c.Name, c.Args)
}

type RunShellCommandCall struct {
	CommandLine string
}

func (c *RunShellCommandCall) String() string {
	return fmt.Sprintf("CommandLine: %+v", c.CommandLine)
}

// SpyRunner is a cmd.Runner for testing purposes
type SpyRunner struct {
	Stdout               string
	Stderr               string
	Error                error
	RunCommandCalls      []*RunCommandCall
	RunShellCommandCalls []*RunShellCommandCall
}

func (r *SpyRunner) RunCommand(name string, args ...string) (stdout string, stderr string, err error) {
	r.RunCommandCalls = append(r.RunCommandCalls, &RunCommandCall{
		Name: name,
		Args: args,
	})
	return r.Stdout, r.Stderr, r.Error
}

func (r *SpyRunner) RunShellCommand(commandLine string) (stdout string, stderr string, err error) {
	r.RunShellCommandCalls = append(r.RunShellCommandCalls, &RunShellCommandCall{
		CommandLine: commandLine,
	})
	return r.Stdout, r.Stderr, r.Error
}
