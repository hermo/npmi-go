package npmi

import (
	"testing"

	"github.com/hermo/npmi-go/pkg/cmd"
)

func TestInstaller_Run(t *testing.T) {

	runner := &cmd.SpyRunner{
		Stdout: "v11.16.3-darwin-x64",
		Stderr: "",
		Error:  nil,
	}
	nc := &Config{
		runner:         runner,
		productionMode: false,
	}
	sut := NewInstaller(nc)

	_, _, err := sut.Run()
	if err != nil {
		t.Errorf("Should not have errored: %v", err)
	}
}
