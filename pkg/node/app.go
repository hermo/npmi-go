package node

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

// DeterminePlatform determines the node platform
func DeterminePlatform() {
	nodeBinary, err := exec.LookPath("node")
	if err != nil {
		log.Fatal("Can't find a node binary in path")
	}

	cmd := exec.Command(nodeBinary, "-p", `process.version + "-" + process.platform + "-" + process.arch`)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err = cmd.Run(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Node env: %s\n", out.String())
}
