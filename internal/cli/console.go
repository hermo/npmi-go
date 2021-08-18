package cli

import "fmt"

type ConsoleWriter interface {
	Println(message string)
	Printf(format string, opts ...interface{})
}

type nonVerboseConsole struct{}
type verboseConsole struct{}

func (c *nonVerboseConsole) Println(message string) {

}

func (c *nonVerboseConsole) Printf(message string, opts ...interface{}) {

}

func (c *verboseConsole) Println(message string) {
	fmt.Println(message)
}

func (c *verboseConsole) Printf(message string, opts ...interface{}) {
	fmt.Printf(message, opts...)
}

func NewConsole(verbose bool) ConsoleWriter {
	if verbose {
		return &verboseConsole{}
	} else {
		return &nonVerboseConsole{}
	}
}
