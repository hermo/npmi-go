package cmd

import (
	"bytes"
	"os/exec"
	"strings"
)

// Runner can run external commands, possibly shell commands
type Runner interface {
	RunCommand(name string, args ...string) (stdout string, stderr string, err error)
	RunShellCommand(commandLine string) (stdout string, stderr string, err error)
}

type DefaultRunner struct{}

func NewRunner() Runner {
	return &DefaultRunner{}
}

// RunCommand runs a command
func (r *DefaultRunner) RunCommand(name string, args ...string) (stdout string, stderr string, err error) {
	cmd := exec.Command(name, args...)
	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	stdout = strings.TrimSpace(stdoutBuf.String())
	stderr = strings.TrimSpace(stderrBuf.String())
	return
}
