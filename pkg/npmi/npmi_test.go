package npmi

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hermo/npmi-go/pkg/cmd"
	"github.com/hermo/npmi-go/pkg/files"
)

func Test_isNodeInProductionMode(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"", false},
		{"prod", false},
		{"development", false},
		{"production", true},
		{"PRODUCTION", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("NODE_ENV", tt.name)
			if got := defaultProductionModeDeterminator(); got != tt.want {
				t.Errorf("isNodeInProductionMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNpmiCanBeCreated(t *testing.T) {
	testDataDir, err := filepath.Abs("../../testdata/cache")
	if err != nil {
		t.Fatalf("test cache dir not present: %v", err)
	}
	options := &Options{
		LocalCache: &LocalCacheOptions{
			Dir: testDataDir,
		},
		UseLocalCache: true,
	}
	builder := NewConfigBuilder()
	builder.WithRunner(&cmd.SpyRunner{
		Stdout: "v11.16.3-darwin-x64",
		Stderr: "",
		Error:  nil,
	})
	config, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewWithConfig(options, config)
	if err != nil {
		t.Fatal(err)
	}
}

func TestNpmiCanBeRun(t *testing.T) {
	err := os.Chdir("../../testdata")
	if err != nil {
		t.Fatalf("could not chdir to testdata: %v", err)
	}

	testDataCacheDir, err := filepath.Abs("cache")
	if err != nil {
		t.Fatalf("testdata/cache dir not present: %v", err)
	}

	// Ensure node_modules exists to avoid error from post-install sanity check
	if !files.DirectoryExists("node_modules") {
		if err = os.Mkdir("node_modules", 0600); err != nil {
			t.Fatalf("node_modules not present and mkdir failed: %v", err)
		}
	}

	options := &Options{
		LogLevel: Debug,
		LocalCache: &LocalCacheOptions{
			Dir: testDataCacheDir,
		},
		UseLocalCache: true,
	}
	builder := NewConfigBuilder()
	runner := &cmd.SpyRunner{
		Stdout: "temp-v11.16.3-darwin-x64",
		Stderr: "",
		Error:  nil,
	}
	builder.WithRunner(runner)
	config, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	config.nodeBinary = "/bin/node"
	config.npmBinary = "/bin/npm"
	m, err := NewWithConfig(options, config)
	if err != nil {
		t.Fatal(err)
	}

	err = m.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanCacheDir(t, testDataCacheDir)
	fmt.Printf("RunCommand calls: %v\n", runner.RunCommandCalls)
	fmt.Printf("RunShellCommandCalls: %v\n", runner.RunShellCommandCalls)

	numRunCommandCalls := len(runner.RunCommandCalls)
	if numRunCommandCalls != 2 {
		t.Errorf("Expected %d RunCommand calls, got %d", 2, numRunCommandCalls)
	}

	numRunShellCommandCalls := len(runner.RunShellCommandCalls)
	if numRunShellCommandCalls != 0 {
		t.Errorf("Expected %d RunShellCommandCalls, got %d", 0, numRunShellCommandCalls)
	}
}

func cleanCacheDir(t *testing.T, cacheDir string) {
	fp := path.Join(cacheDir, "temp-*")
	matches, err := filepath.Glob(fp)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range matches {
		fmt.Printf("F: %s\n", file)
		if err := os.Remove(file); err != nil {
			t.Fatal(err)
		}
	}
}
