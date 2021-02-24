package npmi

import (
	"bytes"
	"io/fs"
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

func Test_hashHandle(t *testing.T) {
	var buffer bytes.Buffer
	buffer.WriteString("Hello!")
	hash, err := hashHandle(&buffer)
	if err != nil {
		t.Error(err)
	}

	expected := "334d016f755cd6dc58c53a86e183882f8ec14f52fb05345887c8a5edd42c87b7"
	if hash != expected {
		t.Errorf("Was expecting %s but got %s instead", expected, hash)
	}
}
