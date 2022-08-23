package npmi

import (
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/hermo/npmi-go/pkg/cmd"
)

func TestNpmInstaller_Run(t *testing.T) {

	runner := &cmd.SpyRunner{
		Stdout: "v11.16.3-darwin-x64",
		Stderr: "",
		Error:  nil,
	}
	nc := &Config{
		runner:         runner,
		productionMode: false,
	}

	log := hclog.New(&hclog.LoggerOptions{
		Name:  "npmi",
		Level: hclog.Info,
	})
	sut := NewNpmInstaller(nc, log)

	_, _, err := sut.Run()
	if err != nil {
		t.Errorf("Should not have errored: %v", err)
	}
}
