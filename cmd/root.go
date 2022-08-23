package cmd

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hermo/npmi-go/pkg/npmi"
)

func Execute() {
	log := hclog.New(&hclog.LoggerOptions{
		Name:  "npmi",
		Level: hclog.Info,
	})

	options, err := ParseFlags()
	if err != nil {
		log.Error("Flag parsing failed", "error", err)
		os.Exit(1)
	}

	m, err := npmi.New(options)
	if err != nil {
		log.Error("Initialization failed", "error", err)
		os.Exit(1)
	}

	err = m.Run()
	if err != nil {
		log.Error("Installation failed", "error", err)
		os.Exit(1)
	}
}
