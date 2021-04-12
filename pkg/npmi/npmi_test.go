package npmi

import (
	"bytes"
	"io/fs"
	"os"
	"testing"
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
