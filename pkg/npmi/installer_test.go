package npmi

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hermo/npmi-go/pkg/cmd"
)

// SpyRunner is a cmd.Runner for testing purposes
type SpyRunner struct {
	Stdout string
	Stderr string
	Error  error
}

func (r *SpyRunner) RunCommand(name string, args ...string) (stdout string, stderr string, err error) {
	return r.Stdout, r.Stderr, r.Error
}

func (r *SpyRunner) RunShellCommand(commandLine string) (stdout string, stderr string, err error) {
	return r.Stdout, r.Stderr, r.Error
}

func TestDeterminePlatformKey(t *testing.T) {

	tests := []struct {
		name    string
		nodeEnv string
		runner  cmd.Runner
		want    string
		wantErr bool
	}{
		{
			name:    "Darwin Node development",
			nodeEnv: "development",
			runner:  &SpyRunner{"v12.16.3-darwin-x64", "", nil},
			want:    "v12.16.3-darwin-x64-dev",
			wantErr: false,
		},
		{
			name:    "Linux Node production",
			nodeEnv: "production",
			runner:  &SpyRunner{"v12.16.0-linux-x64", "", nil},
			want:    "v12.16.0-linux-x64-prod",
			wantErr: false,
		},
		{
			name:    "Missing Node.js",
			nodeEnv: "development",
			runner:  &SpyRunner{"", "", errors.New("fork/exec /usr/bin/node: no such file or directory")},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewInstaller()
			os.Setenv("NODE_ENV", tt.nodeEnv)
			n.Runner = tt.runner
			got, err := n.DeterminePlatformKey()
			if (err != nil) != tt.wantErr {
				t.Errorf("DeterminePlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DeterminePlatform() = %v, want %v", got, tt.want)
			}
		})
	}
}

func getFakePathRoot() string {
	return "../../test/fake_path"
}

func GetDirectoryInFakePath(name string) (string, error) {
	path := path.Join(getFakePathRoot(), name)
	path, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("Invalid Fake path: %v", err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("Can't stat Fake path: %v", err)
	}
	if stat.IsDir() == false {
		return "", fmt.Errorf("Fake path is not a directory: %v", err)
	}
	return path, nil
}

func TestLocateRequiredBinaries(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"Both binaries in path", "both", false},
		{"NPM only", "npm_only", true},
		{"Node only", "node_only", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewInstaller()
			path, err := GetDirectoryInFakePath(tt.path)
			if err != nil {
				t.Errorf("InitNodeBinaries() setup error = %v", err)
			}
			os.Setenv("PATH", path)
			if err := n.LocateRequiredBinaries(); (err != nil) != tt.wantErr {
				t.Errorf("InitNodeBinaries() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
