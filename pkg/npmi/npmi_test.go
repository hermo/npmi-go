package npmi

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hashicorp/go-hclog"
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

	log := hclog.New(&hclog.LoggerOptions{
		Name:  "npmi",
		Level: hclog.Info,
	})
	_, err = NewWithConfig(options, config, log)
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
		UseLocalCache:      true,
		TarDoubleDotPaths:  true,
		TarAbsolutePaths:   true,
		TarLinksOutsideCwd: true,
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
	log := hclog.New(&hclog.LoggerOptions{
		Name:  "npmi",
		Level: hclog.Info,
	})
	m, err := NewWithConfig(options, config, log)
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

// TestNpmBehaviorThroughWrapper makes sure that npm behaves as expected when
// run through the wrapper, especially NODE_ENV handling.
func TestNpmBehaviorThroughWrapper(t *testing.T) {
	tests := []struct {
		name           string
		nodeEnv        string
		wantCapitalize bool
	}{
		{"production_mode", "production", false},
		{"development_mode", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get project root path
			_, filename, _, _ := runtime.Caller(0)
			testFilePath := filepath.Dir(filename)
			testDataDir := filepath.Join(testFilePath, "../../testdata")

			// Verify testdata exists
			if !files.DirectoryExists(testDataDir) {
				t.Fatalf("testdata directory not found at: %s", testDataDir)
			}

			// Setup environment
			origEnv := os.Getenv("NODE_ENV")
			os.Setenv("NODE_ENV", tt.nodeEnv)
			defer os.Setenv("NODE_ENV", origEnv)

			// Change to testdata directory
			err := os.Chdir(testDataDir)
			if err != nil {
				t.Fatalf("could not chdir to testdata: %v (verified path: %s)", err, testDataDir)
			}

			// Clean up node_modules
			os.RemoveAll("node_modules")
			defer os.RemoveAll("node_modules")

			// Build config with real binaries from PATH
			builder := NewConfigBuilder()
			builder.WithNodeAndNpmFromPath()
			config, err := builder.Build()
			if err != nil {
				t.Fatalf("Build failed: %v", err)
			}

			// Create and run installer
			installer := NewNpmInstaller(config, hclog.NewNullLogger())
			_, _, err = installer.Run()
			if err != nil {
				t.Fatalf("Run failed: %v", err)
			}

			// Verify dependencies
			checkDependency(t, "capitalize", tt.wantCapitalize)
			checkDependency(t, "left-pad", true)
			checkDependency(t, "uuid", true)
		})
	}
}

func checkDependency(t *testing.T, pkg string, want bool) {
	exists := files.DirectoryExists(filepath.Join("node_modules", pkg))
	if exists != want {
		t.Errorf("Dependency %s exists = %v, want %v", pkg, exists, want)
	}
}
