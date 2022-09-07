package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hermo/npmi-go/pkg/files"
	"github.com/klauspost/pgzip"
)

// Create an archive file containing the contents of directory src
func Create(filename string, src string) (warnings []string, err error) {
	tree, err := files.CreateFileTree(src)
	if err != nil {
		return nil, err
	}

	archive, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("TAR: %v", err.Error())
	}

	gzw := pgzip.NewWriter(archive)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	badPath := NewBadPath(true)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	for _, item := range *tree {
		// Ignore unknown types
		if item.IsOther() {
			warnings = append(warnings, fmt.Sprintf("Ignored unknown path: %q\n", item.Path))
			continue
		}

		if item.IsLink() {
			warning, err := writeLink(&item, wd, badPath, tw)
			if err != nil {
				return nil, err
			}
			if warning != "" {
				warnings = append(warnings, warning)
			}
		}

		if item.IsRegular() {
			err := writeRegular(&item, tw)
			if err != nil {
				return nil, err
			}
		}

		if item.IsDir() {
			err := writeDir(&item, tw)
			if err != nil {
				return nil, err
			}
		}
	}
	return warnings, err
}

func writeLink(item *files.TreeItem, wd string, badPath *badpath, tw *tar.Writer) (warning string, err error) {
	link, err := os.Readlink(item.Path)
	if err != nil {
		return
	}

	pathDir := filepath.Dir(item.Path)
	linkFull := filepath.Join(wd, pathDir, link)

	if badPath.IsBad(link) {
		err = fmt.Errorf("invalid path: contains bad characters: %s", link)
		return
	}

	if strings.Index(linkFull, wd) != 0 {
		err = fmt.Errorf("invalid path: symlink points outside current directory: %s -> %s", item.Path, link)
		return
	}

	_, err = os.Stat(linkFull)
	if err != nil {
		// Handle non-existent file error
		if os.IsNotExist(err) {
			warning = fmt.Sprintf("Skipped non-existent symlink: %s -> %s\n", item.Path, link)
			err = nil
		}
	}

	if err = writeHeader(item, tw, &link); err != nil {
		return
	}
	return
}

func writeDir(item *files.TreeItem, tw *tar.Writer) error {
	return writeHeader(item, tw, nil)
}

func writeRegular(item *files.TreeItem, tw *tar.Writer) error {
	if err := writeHeader(item, tw, nil); err != nil {
		return err
	}

	f, err := os.Open(item.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(tw, f); err != nil {
		return err
	}

	return nil
}

func writeHeader(item *files.TreeItem, tw *tar.Writer, linkname *string) error {
	// create a new dir/file header
	header, err := tar.FileInfoHeader(*item.FileInfo, item.Path)
	if err != nil {
		return err
	}

	// Use PAX format for utf-8 support
	header.Format = tar.FormatPAX

	// Symlink special sauce
	if item.IsLink() {
		header.Typeflag = tar.TypeSymlink
		header.Linkname = *linkname
	}

	header.Name = filepath.ToSlash(item.Path)
	header.Linkname = filepath.ToSlash(header.Linkname)

	// write the header
	if err = tw.WriteHeader(header); err != nil {
		return err
	}
	return nil

}

type badpath struct {
	allowDoubleDot bool
}

func NewBadPath(allowDoubleDot bool) *badpath {
	return &badpath{allowDoubleDot}
}

func (bp *badpath) IsBad(path string) bool {
	if strings.ContainsAny(path, "<|>:\"*?\\") {
		return true
	}

	if !bp.allowDoubleDot {
		if strings.Contains(path, "..") {
			return true
		}
	}

	if path[0] == '/' {
		return true
	}

	if strings.HasPrefix(path, " ") {
		return true
	}

	path = strings.ToUpper(path)
	windowsDevices := []string{"CON", "PRN", "AUX", "NUL"}
	for _, s := range windowsDevices {
		if path == s {
			return true
		}
	}

	windowsDevicePrefixes := []string{"COM", "LPT"}
	for _, s := range windowsDevicePrefixes {
		if strings.HasPrefix(path, s) && len(path) > 3 {
			if path[3] >= '1' && path[3] <= '9' {
				return true
			}
		}
	}
	return false
}

// Extract all files from an archive to current directory
func Extract(reader io.Reader) ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	var manifest []string
	gzr, err := pgzip.NewReaderN(reader, 500e3, 50)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	badPath := NewBadPath(false)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		// if the header is nil, just skip it (not sure how this happens)
		if header == nil {
			fmt.Println("Nil header?")
			continue
		}

		// the target location where the dir/file should be created
		target := header.Name
		target = filepath.ToSlash(filepath.Clean(target))

		if badPath.IsBad(target) {
			return nil, fmt.Errorf("invalid path: contains bad characters")
		}

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
					return nil, err
				}
			}

			// Defer setting directory mtimes as they are bound to change when files are written to them
			defer func() {
				if err := os.Chtimes(target, time.Now(), header.FileInfo().ModTime()); err != nil {
					fmt.Fprintf(os.Stderr, "Error: Could not restore mtime for directory %s: %v", target, err)
				}
			}()

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return nil, err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
			if err = os.Chtimes(target, time.Now(), header.FileInfo().ModTime()); err != nil {
				return nil, err
			}

			if err = os.Chmod(target, header.FileInfo().Mode()); err != nil {
				return nil, err
			}

			manifest = append(manifest, target)

		case tar.TypeSymlink:
			reldest := filepath.ToSlash(header.Name)
			dest := filepath.Join(cwd, filepath.ToSlash(header.Name))
			source := filepath.ToSlash(header.Linkname)

			if source[0] == '/' {
				return nil, fmt.Errorf("invalid path: symlink with absolute path: %s -> %s", dest, source)
			}

			dir := filepath.Dir(dest)
			resolvedTarget := filepath.Join(dir, source)

			if !strings.HasPrefix(resolvedTarget, cwd) {
				return nil, fmt.Errorf("invalid path: %s -> %s points outside cwd", reldest, source)
			}

			err = syncSymLink(source, dest)
			if err != nil {
				return nil, fmt.Errorf("syncing symlink failed: %v", err)
			}

			// symlink timestamps are not preserved
			// see https://stackoverflow.com/questions/54762079/how-to-change-timestamp-for-symbol-link-using-golang
			manifest = append(manifest, target)

		default:
			return nil, fmt.Errorf("unsupported file type: %+v", header)
		}

	}
	return manifest, nil
}

// syncSymlink makes sure that a symlink exists and is pointing to the right place
func syncSymLink(source string, dest string) error {
	info, err := os.Lstat(dest)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else if info.Mode()&os.ModeSymlink != 0 {
		currentTarget, err := os.Readlink(dest)
		if err != nil {
			return err
		}

		if currentTarget == source {
			return nil
		}

		err = os.Remove(dest)
		if err != nil {
			return err
		}
	}

	return os.Symlink(source, dest)
}
