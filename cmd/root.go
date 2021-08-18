package cmd

import (
	"log"

	"github.com/hermo/npmi-go/pkg/npmi"
)

func Execute() {
	options, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	m, err := npmi.New(options)
	if err != nil {
		log.Fatal(err)
	}

	err = m.Run()
	if err != nil {
		log.Fatal(err)
	}
}
