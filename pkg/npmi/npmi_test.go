package npmi

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/hermo/npmi-go/pkg/cmd"
)

func Test_HashFile(t *testing.T) {
	_, err := HashFile("nonexistant.file")
	if err == nil {
		t.Error("Was expecting a file not found error")
	} else {
		if _, ok := err.(*fs.PathError); !ok {
			t.Errorf("Was expecting fs.PathError but got %T", err)
		}
	}
}

func Test_hashInput(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteString("Hello!")
	hash, err := hashInput(&buffer)
	if err != nil {
		t.Error(err)
	}

	expected := "334d016f755cd6dc58c53a86e183882f8ec14f52fb05345887c8a5edd42c87b7"
	if hash != expected {
		t.Errorf("Was expecting %s but got %s instead", expected, hash)
	}
}

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
			if got := isNodeInProductionMode(); got != tt.want {
				t.Errorf("isNodeInProductionMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
			os.Setenv("NODE_ENV", tt.nodeEnv)
			runner = tt.runner
			got, err := DeterminePlatformKey()
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
			path, err := GetDirectoryInFakePath(tt.path)
			if err != nil {
				t.Errorf("InitNodeBinaries() setup error = %v", err)
			}
			os.Setenv("PATH", path)
			if err := LocateRequiredBinaries(); (err != nil) != tt.wantErr {
				t.Errorf("InitNodeBinaries() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"Empty string", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", false},
		{"Hello string", "hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HashString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HashString() = %v, want %v", got, tt.want)
			}
		})
	}
}
