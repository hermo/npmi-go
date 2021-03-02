package cmd

import (
	"bytes"
	"os/exec"
	"strings"
)

// RunCommand runs a command
func RunCommand(name string, args ...string) (stdout string, stderr string, err error) {
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
