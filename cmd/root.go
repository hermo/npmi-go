package cmd

import (
	"log"
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hermo/npmi-go/pkg/npmi"
)

func Execute() {
	options, err := ParseFlags()
	if err != nil {
		log.Fatalf("Flag parsing failed: %s", err)
		os.Exit(1)
	}

	log := hclog.New(&hclog.LoggerOptions{
		Name:       "npmi",
		Level:      hclog.LevelFromString(options.LogLevel.String()),
		JSONFormat: options.Json,
	})

	m, err := npmi.New(options, log)
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
