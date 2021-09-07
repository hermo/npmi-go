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
	"time"
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
	mDate := time.Date(2021, time.September, 6, 11, 27, 4, 0, time.UTC)
	zeroDate := time.Time{}

	tarContents := []struct {
		Name    string
		Content string
		Date    time.Time
		Mode    os.FileMode
		Type    byte
	}{
		{"root.txt", "rootfile", mDate, 0655, tar.TypeReg},
		{"somedir", "", mDate, 0700, tar.TypeDir},
		{"somedir/sub_link.txt", "sub.txt", zeroDate, 0655, tar.TypeSymlink}, // Link created before actual file on purpose
		{"somedir/sub.txt", "subfile", mDate, 0644, tar.TypeReg},
		{"sub_link.txt", "somedir/sub.txt", zeroDate, 0650, tar.TypeSymlink},
		{"somedir/root_link.txt", "../root.txt", zeroDate, 0666, tar.TypeSymlink},
	}
	// Number of files/links expected in archive
	wantManifestLen := 5

	for _, f := range tarContents {
		data := []byte(f.Content)
		hdr := tar.Header{
			Format:   tar.FormatPAX,
			Typeflag: f.Type,
			Name:     f.Name,
			ModTime:  f.Date,
		}
		if f.Type == tar.TypeReg {
			hdr.Size = int64(len(data))
			hdr.Mode = int64(f.Mode)
		}
		if f.Type == tar.TypeSymlink {
			hdr.Linkname = f.Content
			hdr.Mode = int64(f.Mode & os.ModePerm)
		}

		if f.Type == tar.TypeDir {
			hdr.Mode = int64(f.Mode & 0777)
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
		switch f.Type {
		case tar.TypeDir:
			fi, err := os.Stat(f.Name)
			if err != nil {
				t.Fatal(err)
			}
			if fi.IsDir() != true {
				t.Fatalf("%s should be a directory but is not", f.Name)
			}
			if (fi.Mode() & 0777) != f.Mode {
				t.Fatalf("%s mode=%v, want=%v", f.Name, fi.Mode(), f.Mode)
			}
		case tar.TypeSymlink:
			li, err := os.Lstat(f.Name)
			if err != nil {
				t.Fatal(err)
			}

			if !f.Date.IsZero() {
				t.Fatal("Symlink timestamps are not supported and need to be zero")
			}

			target, err := os.Readlink(f.Name)
			if err != nil {
				t.Fatal(err)
			}
			if target != f.Content {
				t.Fatalf("Link %s points to %s, want=%s", li.Name(), f.Content, target)
			}
		case tar.TypeReg:
			fi, err := os.Stat(f.Name)
			if err != nil {
				t.Fatal(err)
			}

			if fi.Mode() != f.Mode {
				t.Fatalf("%s mode=%v, want=%v", f.Name, fi.Mode(), f.Mode)
			}

			if fi.ModTime().UTC() != f.Date {
				t.Fatalf("%s mtime=%v, want=%v", f.Name, fi.ModTime().UTC(), f.Date)
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
