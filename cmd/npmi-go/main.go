package main

import (
	"log"

	"github.com/hermo/npmi-go/pkg/npmi"
)

func main() {
	options, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}

	m, err := npmi.New(options)
	if err != nil {
		log.Fatal(err)
	}

	err = m.RunAllSteps()
	if err != nil {
		log.Fatal(err)
	}
}
