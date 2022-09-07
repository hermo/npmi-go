package cmd

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/hermo/npmi-go/pkg/npmi"
)

func Execute() {
	options, err := ParseFlags()
	if err != nil {
		// Create a default logger for handling early errors.
		log := hclog.New(&hclog.LoggerOptions{
			Name: "npmi",
		})
		log.Error("Flag parsing failed", "error", err)
		os.Exit(1)
	}

	// Create properly configured logger for the rest of the runtime duration.
	log := hclog.New(&hclog.LoggerOptions{
		Name:       "npmi",
		Level:      hclog.LevelFromString(options.LogLevel.String()),
		JSONFormat: options.Json,
		Color:      hclog.AutoColor,
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
