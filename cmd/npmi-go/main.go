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

	main, err := npmi.New(options)
	if err != nil {
		log.Fatal(err)
	}

	err = main.Run()
	if err != nil {
		log.Fatal(err)
	}
}
