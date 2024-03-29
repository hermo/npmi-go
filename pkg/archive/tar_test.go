package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
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

	wantWarnings := 0

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

	testDir, err := prepareTestDir()
	if err != nil {
		t.Fatalf("Can't create temporary test directory: %v", err)
	}

	defer removeTestDir(testDir)

	err = os.Mkdir("extract", 0700)
	if err != nil {
		t.Fatalf("Could not create directory: %v", err)
	}

	err = os.Chdir("extract")
	if err != nil {
		t.Fatalf("Can't chdir to test directory: %v", err)
	}

	options := TarOptions{
		AllowAbsolutePaths:   false,
		AllowDoubleDotPaths:  false,
		AllowLinksOutsideCwd: false,
	}

	manifest, warnings, err := Extract(&buf, &options)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	if len(warnings) != wantWarnings {
		t.Fatalf("Expected %d warnings, got %d", wantWarnings, len(warnings))
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

			if fi.ModTime().UTC() != f.Date {
				t.Fatalf("%s mtime=%v, want=%v", f.Name, fi.ModTime().UTC(), f.Date)
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
	tests := []struct {
		Name           string
		Content        string
		Type           byte
		WantedWarnings int
	}{
		{"../evil.txt", "evil", tar.TypeReg, 0},
		{"./../evil.txt", "evil", tar.TypeReg, 0},
		{"/evil.txt", "evil", tar.TypeReg, 0},
		{"C:/Users/Public/evil.txt", "evil", tar.TypeReg, 0},
		{"C:|Users/Public/evil.txt", "evil", tar.TypeReg, 0},
		{"C:\\Users\\Public\\evil2.txt", "evil2", tar.TypeReg, 0},
		{"abs_link", "/etc/passwd", tar.TypeSymlink, 0},
		{"outside_link", "../outside_cwd", tar.TypeSymlink, 0},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			testDir, err := prepareTestDir()
			if err != nil {
				t.Fatalf("Can't create temporary test directory: %v", err)
			}

			defer removeTestDir(testDir)

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

			data := []byte(tt.Content)
			hdr := tar.Header{
				Typeflag: tt.Type,
				Name:     tt.Name,
			}
			if tt.Type == tar.TypeReg {
				hdr.Size = int64(len(data))
			}
			if tt.Type == tar.TypeSymlink {
				hdr.Linkname = tt.Content
			}
			err = tw.WriteHeader(&hdr)
			if err != nil {
				t.Fatal(err)
			}
			if tt.Type == tar.TypeReg {
				_, err = tw.Write(data)
				if err != nil {
					t.Fatal(err)
				}
			}

			tw.Close()
			gzw.Close()

			options := TarOptions{
				AllowAbsolutePaths:   false,
				AllowDoubleDotPaths:  false,
				AllowLinksOutsideCwd: false,
			}

			_, warnings, err := Extract(&buf, &options)
			if len(warnings) != tt.WantedWarnings {
				t.Errorf("Expected %d warnings, got %d", tt.WantedWarnings, len(warnings))
			}

			if err == nil {
				t.Errorf("Extract should have failed but did not: %s", tt.Name)
			} else {
				if !strings.Contains(err.Error(), "invalid path") {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// Some tests for disallowing bad symlinks
func Test_CreateArchiveSymlinks(t *testing.T) {
	tests := []struct {
		Source       string
		Target       string
		IsEvil       bool
		WarningCount int
	}{
		// Normal cases
		{"hello2.txt", "hello.txt", false, 1},
		{"hello_subdir_1.txt", "subdir/hello.txt", false, 1},
		{"hello_subdir_2.txt", "subdir/.foo/hello.txt", false, 1},
		{"subdir/hello_parent_1.txt", "../hello.txt", false, 1},
		// Evil cases that are allowed by default for compatibility 🤦‍♂️
		{"abs_1.txt", "/hello.txt", false, 2},
		{"abs_2.txt", "/etc/passwd", false, 1},
		{"subdir/evil_parent_0.txt", "../../hello.txt", false, 2},
		{"evil_parent_1.txt", "../evil.txt", false, 2},
		{"evil_parent_2.txt", "./../evil.txt", false, 2},
		// Still evil
		{"evil_abs_win_1.txt", "C:/Users/Public/evil.txt", true, 0},
		{"evil_abs_win_2.txt", "C:|Users/Public/evil.txt", true, 0},
		{"evil_abs_win_3.txt", "C:\\Users\\Public\\evil2.txt", true, 0},
	}

	for _, tt := range tests {
		var testType string
		if tt.IsEvil {
			testType = "EVIL"
		} else {
			testType = "NORMAL"
		}
		testName := fmt.Sprintf("%s/%s", testType, tt.Source)

		t.Run(testName, func(t *testing.T) {
			testDir, err := prepareTestDir()
			if err != nil {
				t.Fatalf("Can't create temporary test directory: %v", err)
			}

			defer removeTestDir(testDir)

			err = os.Mkdir("subdir", 0750)
			if err != nil {
				t.Fatalf("Can't create test subdir: %v", err)
			}

			err = os.Symlink(tt.Target, tt.Source)
			if err != nil {
				t.Fatalf("Can't create test symlink: %v", err)
			}

			options := TarOptions{
				AllowAbsolutePaths:   true,
				AllowDoubleDotPaths:  true,
				AllowLinksOutsideCwd: true,
			}
			warnings, err := Create("temp.tgz", ".", &options)

			if len(warnings) != tt.WarningCount {
				t.Errorf("Expected %d warnings, got %d", tt.WarningCount, len(warnings))
				fmt.Printf("[%s] Warnings: %v\n", testName, warnings)
			}

			if tt.IsEvil {
				if err == nil {
					t.Fatalf("Evil symlink (%s -> %s) should have failed but did not\n", tt.Source, tt.Target)
				} else {
					if !strings.Contains(err.Error(), "invalid path") {
						t.Fatalf("Unexpected error when creating evil symlink (%s -> %s): %v", tt.Source, tt.Target, err)
					}
				}
			} else {
				if err != nil {
					if strings.Contains(err.Error(), "invalid path") {
						t.Fatalf("Normal symlink (%s -> %s) failed: %v", tt.Source, tt.Target, err)
					} else {
						t.Fatalf("Unexpected error when creating normal symlink (%s -> %s): %v", tt.Source, tt.Target, err)
					}
				}
			}
		})
	}
}

func Test_CreateArchive_DisallowSymlinksOutsideCwd(t *testing.T) {
	tests := []struct {
		Source       string
		Target       string
		IsEvil       bool
		WarningCount int
	}{
		// Normal cases
		{"hello2.txt", "hello.txt", false, 1},
		{"hello_subdir_1.txt", "subdir/hello.txt", false, 1},
		{"hello_subdir_2.txt", "subdir/.foo/hello.txt", false, 1},
		{"subdir/hello_parent_1.txt", "../hello.txt", false, 1},
		// Evil cases
		{"subdir/evil_parent_0.txt", "../../hello.txt", true, 0},
		{"evil_parent_1.txt", "../evil.txt", true, 0},
		{"evil_parent_2.txt", "./../evil.txt", true, 0},
		{"evil_abs_1.txt", "/evil.txt", true, 0},
		{"evil_abs_2.txt", "/etc/passwd", true, 0},
		{"evil_abs_win_1.txt", "C:/Users/Public/evil.txt", true, 0},
		{"evil_abs_win_2.txt", "C:|Users/Public/evil.txt", true, 0},
		{"evil_abs_win_3.txt", "C:\\Users\\Public\\evil2.txt", true, 0},
	}

	for _, tt := range tests {
		var testType string
		if tt.IsEvil {
			testType = "EVIL"
		} else {
			testType = "NORMAL"
		}
		testName := fmt.Sprintf("%s/%s", testType, tt.Source)

		t.Run(testName, func(t *testing.T) {
			testDir, err := prepareTestDir()
			if err != nil {
				t.Fatalf("Can't create temporary test directory: %v", err)
			}

			defer removeTestDir(testDir)

			err = os.Mkdir("subdir", 0750)
			if err != nil {
				t.Fatalf("Can't create test subdir: %v", err)
			}

			err = os.Symlink(tt.Target, tt.Source)
			if err != nil {
				t.Fatalf("Can't create test symlink: %v", err)
			}

			options := TarOptions{
				AllowAbsolutePaths:   false,
				AllowDoubleDotPaths:  true,
				AllowLinksOutsideCwd: false,
			}
			warnings, err := Create("temp.tgz", ".", &options)

			if len(warnings) != tt.WarningCount {
				t.Errorf("Expected %d warnings, got only %d", tt.WarningCount, len(warnings))
			}

			if tt.IsEvil {
				if err == nil {
					t.Fatalf("Evil symlink (%s -> %s) should have failed but did not\n", tt.Source, tt.Target)
				} else {
					if !strings.Contains(err.Error(), "invalid path") {
						t.Fatalf("Unexpected error when creating evil symlink (%s -> %s): %v", tt.Source, tt.Target, err)
					}
				}
			} else {
				if err != nil {
					if strings.Contains(err.Error(), "invalid path") {
						t.Fatalf("Normal symlink (%s -> %s) failed: %v", tt.Source, tt.Target, err)
					} else {
						t.Fatalf("Unexpected error when creating normal symlink (%s -> %s): %v", tt.Source, tt.Target, err)
					}
				}
			}
		})
	}
}

// Create a temporary directory and chdir into it
func prepareTestDir() (string, error) {
	testBaseDir, err := filepath.Abs(fmt.Sprintf("%s/../../testdata", getBaseDir()))
	if err != nil {
		return "", fmt.Errorf("can't find test directory: %v", err)
	}
	testDir, err := os.MkdirTemp(testBaseDir, "test")
	if err != nil {
		return "", fmt.Errorf("can't create temporary test directory: %v", err)
	}

	err = os.Chdir(testDir)
	if err != nil {
		return "", fmt.Errorf("can't chdir to test directory: %v", err)
	}

	return testDir, nil
}

// Remove test directory after testing
func removeTestDir(testDir string) {
	err := os.Chdir(getBaseDir())
	if err != nil {
		return
	}
	err = os.RemoveAll(testDir)
	if err != nil {
		fmt.Printf("ERROR: could not remove temp directory: %v", err)
	}
}

func BenchmarkExtract(b *testing.B) {
	testDir, err := prepareTestDir()
	if err != nil {
		b.Fatalf("Can't create temporary test directory: %v", err)
	}

	defer removeTestDir(testDir)

	err = os.Mkdir("extract", 0700)
	if err != nil {
		b.Fatalf("Could not create directory: %v", err)
	}

	err = os.Chdir("extract")
	if err != nil {
		b.Fatalf("Can't chdir to test directory: %v", err)
	}
	testArchive, err := filepath.Abs(fmt.Sprintf("%s/../../bench/extract/test.tgz", getBaseDir()))
	if err != nil {
		b.Fatal(err)
	}

	f, err := os.Open(testArchive)
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()

	options := TarOptions{
		AllowAbsolutePaths:   false,
		AllowDoubleDotPaths:  false,
		AllowLinksOutsideCwd: false,
	}

	for i := 0; i < b.N; i++ {
		f.Seek(0, io.SeekStart)

		_, _, err := Extract(f, &options)
		if err != nil {
			b.Fatalf("Extract failed: %v", err)
		}
	}
}

func TestBadPath(t *testing.T) {
	tests := []struct {
		allowDoubleDot     bool
		allowAbsolutePaths bool
		Path               string
		Expected           bool
	}{
		// Evil inputs, double dots not allowed
		{false, false, "/evil1.txt", true},
		{false, false, `\evil1.txt`, true},
		{false, false, "evil11..txt", true},
		{false, false, "../evil2.txt", true},
		{false, false, "C:/Users/Public/evil3.txt", true},
		{false, false, "C:|Users/Public/evil4.txt", true},
		{false, false, "<", true},
		{false, false, "<foo", true},
		{false, false, " <foo2", true},
		{false, false, "bar>", true},
		{false, false, "win\\separator", true},
		{false, false, "\tbadpath", true},
		{false, false, "\x01badpath", true},
		{false, false, "!badpath", true},
		{false, false, "@badpath", true},
		{false, false, ":evil.txt", true},
		{false, false, ";evil.txt", true},
		{false, false, "!evil.txt", true},
		{false, false, "@evil.txt", true},
		{false, false, "#evil.txt", true},
		{false, false, "$evil.txt", true},
		{false, false, "%evil.txt", true},
		{false, false, "^evil.txt", true},
		{false, false, "&evil.txt", true},
		{false, false, "*evil.txt", true},
		{false, false, "(evil.txt", true},
		{false, false, ")evil.txt", true},
		{false, false, "+evil.txt", true},
		{false, false, "=evil.txt", true},
		{false, false, "{evil.txt", true},
		{false, false, "}evil.txt", true},
		{false, false, "[evil.txt", true},
		{false, false, "]evil.txt", true},
		{false, false, "|evil.txt", true},
		{false, false, "\\evil.txt", true},
		{false, false, "/evil.txt", true},
		{false, false, "?evil.txt", true},
		{false, false, "<evil.txt", true},
		{false, false, ">evil.txt", true},
		{false, false, ",evil.txt", true},
		{false, false, "`evil.txt", true},
		{false, false, "~evil.txt", true},
		{false, false, " evil.txt", true},
		{false, false, "\tevil.txt", true},
		{false, false, "\x01evil.txt", true},
		{false, false, "\x02evil.txt", true},
		{false, false, "\x03evil.txt", true},
		{false, false, "\x04evil.txt", true},
		{false, false, "\x05evil.txt", true},
		{false, false, "\x06evil.txt", true},
		{false, false, "\x07evil.txt", true},
		{false, false, "\x08evil.txt", true},
		{false, false, "\x09evil.txt", true},
		{false, false, "\x0aevil.txt", true},
		{false, false, "\x0bevil.txt", true},
		{false, false, "\x0cevil.txt", true},
		{false, false, "\x0devil.txt", true},
		{false, false, "\x0eevil.txt", true},
		{false, false, "\x0fevil.txt", true},
		{false, false, "\x10evil.txt", true},
		{false, false, "\x11evil.txt", true},
		{false, false, "\x12evil.txt", true},
		{false, false, "\x13evil.txt", true},
		{false, false, "\x14evil.txt", true},
		{false, false, "\x15evil.txt", true},
		{false, false, "\x16evil.txt", true},
		{false, false, "\x17evil.txt", true},
		{false, false, "\x18evil.txt", true},
		{false, false, "\x19evil.txt", true},
		{false, false, "\x1aevil.txt", true},
		{false, false, "\x1bevil.txt", true},
		{false, false, "\x1cevil.txt", true},
		{false, false, "\x1devil.txt", true},
		{false, false, "\x1eevil.txt", true},
		{false, false, "\x1fevil.txt", true},
		{false, false, " Spaceman", true},
		{false, false, "C:\\Users\\Public\\evil5.txt", true},

		// Evil inputs, double dots allowed
		{true, false, "/../evil_double_dots_6.txt", true},
		{true, false, "/../evil_double_dots_61..txt", true},

		// Good inputs, double dots disallowed
		{false, false, "kissa7.txt", false},
		{false, false, "foo/bar//double_dots71.txt", false},
		{false, false, "Hello dolly", false},

		// Good inputs, double dots allowed
		{true, false, "../double_dots8.txt", false},
		{true, false, "double_dots9..txt", false},

		// Good inputs, double dots disallowed, absolute paths allowed
		{false, true, "/kissa.txt", false},

		// Evil inputs, double dots disallowed, absolute paths allowed
		{false, true, "../kissa.txt", true},
		{false, true, "/../kissa.txt", true},

		// Good inputs, double dots allowed, absolute paths allowed
		{true, true, "/kissa.txt", false},
		{true, true, "../kissa.txt", false},
		{true, true, "/../kissa.txt", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("allowDoubleDots:%v,Path:%s", tt.allowDoubleDot, tt.Path), func(t *testing.T) {
			bp := NewBadPath(tt.allowDoubleDot, tt.allowAbsolutePaths)
			if bp.IsBad(tt.Path) != tt.Expected {
				t.Errorf("IsBad(%s) did not return %v", tt.Path, tt.Expected)
			}
		})
	}
}

func BenchmarkBadPath(b *testing.B) {
	bp := NewBadPath(false, false)
	path := "node_modules/foo_bar/baz.js/somepath"
	for i := 0; i < b.N; i++ {
		bp.IsBad(path)
	}
}

func BenchmarkCreate(b *testing.B) {
	testDir, err := prepareTestDir()
	if err != nil {
		b.Fatalf("Can't create temporary test directory: %v", err)
	}

	defer removeTestDir(testDir)

	err = os.Mkdir("compress", 0700)
	if err != nil {
		b.Fatalf("Could not create directory: %v", err)
	}

	err = os.Chdir("compress")
	if err != nil {
		b.Fatalf("Can't chdir to test directory: %v", err)
	}
	testArchive, err := filepath.Abs(fmt.Sprintf("%s/../../bench/extract/test.tgz", getBaseDir()))
	if err != nil {
		b.Fatal(err)
	}

	f, err := os.Open(testArchive)
	if err != nil {
		b.Fatal(err)
	}
	options := TarOptions{
		AllowAbsolutePaths:   false,
		AllowDoubleDotPaths:  false,
		AllowLinksOutsideCwd: false,
	}

	_, _, err = Extract(f, &options)
	if err != nil {
		b.Fatalf("Extract failed: %v", err)
	}
	f.Close()

	for i := 0; i < b.N; i++ {
		warnings, err := Create("compressed.tgz", "node_modules", &options)
		if err != nil {
			b.Fatalf("Create failed: %v", err)
		}

		if len(warnings) > 100 {
			b.Errorf("Warnings: %v", warnings)
		}

		if err = os.Remove("compressed.tgz"); err != nil {
			b.Fatalf("Remove failed: %v", err)
		}
	}
}
