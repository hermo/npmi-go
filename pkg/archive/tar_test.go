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
	"strings"
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
		{"somedir/sub_link.txt", "sub.txt", tar.TypeSymlink}, // Link created before actual file on purpose
		{"somedir/sub.txt", "subfile", tar.TypeReg},
		{"sub_link.txt", "somedir/sub.txt", tar.TypeSymlink},
		{"somedir/root_link.txt", "../root.txt", tar.TypeSymlink},
	}
	// Number of files/links expected in archive
	wantManifestLen := 5

	for _, f := range tarContents {
		data := []byte(f.Content)
		hdr := tar.Header{
			Format:   tar.FormatPAX,
			Typeflag: f.Type,
			Name:     f.Name,
		}
		if f.Type == tar.TypeReg {
			hdr.Size = int64(len(data))
		}
		if f.Type == tar.TypeSymlink {
			hdr.Linkname = f.Content
		}
		err := tw.WriteHeader(&hdr)
		if err != nil {
			t.Fatal(err)
		}
		if f.Type == tar.TypeReg {
			_, err := tw.Write(data)
			if err != nil {
				t.Fatal(err)
			}
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
		err := os.Chdir(getBaseDir())
		if err != nil {
			t.Fatal(err)
		}
		err = os.RemoveAll(testDir)
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

	if len(manifest) != wantManifestLen {
		t.Fatalf("Manifest length=%d,want=%d", len(manifest), wantManifestLen)
	}

	for _, f := range tarContents {
		if f.Type == tar.TypeDir {
			continue
		}
		exists, err := files.IsExistingFile(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("File %s should exists but does not", f.Name)
		} else {
			if f.Type == tar.TypeSymlink {
				li, err := os.Lstat(f.Name)
				if err != nil {
					t.Fatal(err)
				}
				target, err := os.Readlink(f.Name)
				if err != nil {
					t.Fatal(err)
				}
				if target != f.Content {
					t.Fatalf("Link %s points to %s, want=%s", li.Name(), f.Content, target)
				}

			}
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
		{"/evil.txt", "evil", tar.TypeReg},
		{"C:/Users/Public/evil.txt", "evil", tar.TypeReg},
		{"C:|Users/Public/evil.txt", "evil", tar.TypeReg},
		{"COM1>", "evil", tar.TypeReg},
		{"CON", "evil", tar.TypeReg},
		{"NUL", "evil", tar.TypeReg},
		{"C:\\Users\\Public\\evil2.txt", "evil2", tar.TypeReg},
		{"abs_link", "/etc/passwd", tar.TypeSymlink},
		{"outside_link", "../outside_cwd", tar.TypeSymlink},
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
			err := os.Chdir(getBaseDir())
			if err != nil {
				t.Fatal(err)
			}
			err = os.RemoveAll(testDir)
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
		if f.Type == tar.TypeSymlink {
			hdr.Linkname = f.Content
		}
		err = tw.WriteHeader(&hdr)
		if err != nil {
			t.Fatal(err)
		}
		if f.Type == tar.TypeReg {
			_, err = tw.Write(data)
			if err != nil {
				t.Fatal(err)
			}
		}

		tw.Close()
		gzw.Close()

		_, err = Extract(&buf)
		if err == nil {
			t.Errorf("Extract should have failed but did not: %s", f.Name)
		} else {
			if !strings.Contains(err.Error(), "invalid path") {
				t.Errorf("Unexpected error: %v", err)
			}
		}
	}
}
