package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hermo/npmi-go/pkg/files"
)

func getBaseDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return path.Dir(filename)
}

func Test_ExtractFilesNormal(t *testing.T) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	tarContents := []struct {
		Name    string
		Content string
		Type    byte
	}{
		{"root.txt", "rootfile", tar.TypeReg},
		{"somedir", "", tar.TypeDir},
		{"somedir/sub.txt", "subfile", tar.TypeReg},
	}

	for _, f := range tarContents {
		data := []byte(f.Content)
		hdr := tar.Header{
			Typeflag: f.Type,
			Name:     f.Name,
		}
		if f.Type == tar.TypeReg {
			hdr.Size = int64(len(data))
		}
		tw.WriteHeader(&hdr)
		if f.Type == tar.TypeReg {
			tw.Write(data)
		}
	}

	tw.Close()
	gzw.Close()

	testBaseDir, err := filepath.Abs(fmt.Sprintf("%s/../../testdata", getBaseDir()))
	if err != nil {
		t.Fatalf("Can't find test directory: %v", err)
	}

	testDir, err := os.MkdirTemp(testBaseDir, "test")
	if err != nil {
		t.Fatalf("Can't create temporary test directory: %v", err)
	}

	defer func() {
		os.Chdir(getBaseDir())
		err := os.RemoveAll(testDir)
		if err != nil {
			t.Fatalf("Could not remove temp directory: %v", err)
		}
	}()

	err = os.Chdir(testDir)
	if err != nil {
		t.Fatalf("Can't chdir to test directory: %v", err)
	}

	err = os.Mkdir("extract", 0700)
	if err != nil {
		t.Fatalf("Could not create directory: %v", err)
	}

	err = os.Chdir("extract")
	if err != nil {
		t.Fatalf("Can't chdir to test directory: %v", err)
	}

	manifest, err := Extract(&buf)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	wantManifestLen := 2
	if len(manifest) != wantManifestLen {
		t.Fatalf("Manifest length=%d,want=%d", len(manifest), wantManifestLen)
	}

	for _, f := range tarContents {
		if f.Type != tar.TypeReg {
			continue
		}
		exists, err := files.IsExistingFile(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("File %s should exists but does not", f.Name)
		}
	}
}

func Test_ExtractFilesEvil(t *testing.T) {
	evilPayloads := []struct {
		Name    string
		Content string
		Type    byte
	}{
		{"../evil.txt", "evil", tar.TypeReg},
		{"./../evil.txt", "evil", tar.TypeReg},
	}

	testBaseDir, err := filepath.Abs(fmt.Sprintf("%s/../../testdata", getBaseDir()))
	if err != nil {
		t.Fatalf("Can't find test data directory: %v", err)
	}

	for _, f := range evilPayloads {
		testDir, err := os.MkdirTemp(testBaseDir, "test")
		if err != nil {
			t.Fatalf("Can't create temporary test directory: %v", err)
		}

		defer func() {
			os.Chdir(getBaseDir())
			err := os.RemoveAll(testDir)
			if err != nil {
				t.Fatalf("Could not remove temp directory: %v", err)
			}
		}()

		err = os.Chdir(testDir)
		if err != nil {
			t.Fatalf("Can't chdir to test directory: %v", err)
		}

		err = os.Mkdir("extract", 0700)
		if err != nil {
			t.Fatalf("Could not create directory: %v", err)
		}

		err = os.Chdir("extract")
		if err != nil {
			t.Fatalf("Can't chdir to test directory: %v", err)
		}

		var buf bytes.Buffer
		gzw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gzw)

		data := []byte(f.Content)
		hdr := tar.Header{
			Typeflag: f.Type,
			Name:     f.Name,
		}
		if f.Type == tar.TypeReg {
			hdr.Size = int64(len(data))
		}
		tw.WriteHeader(&hdr)
		if f.Type == tar.TypeReg {
			tw.Write(data)
		}

		tw.Close()
		gzw.Close()

		_, err = Extract(&buf)
		if err == nil {
			t.Fatalf("Extract should have failed but did not")
		}
	}
}
