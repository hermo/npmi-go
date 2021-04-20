package npmi

import (
	"testing"
)

func TestInstaller_Run(t *testing.T) {

	runner := &SpyRunner{"v11.16.3-darwin-x64", "", nil}
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
