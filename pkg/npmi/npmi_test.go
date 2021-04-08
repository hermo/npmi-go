package npmi

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
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

func TestDeterminePlatform(t *testing.T) {

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
			got, err := DeterminePlatform()
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
