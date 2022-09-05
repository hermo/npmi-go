package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/pgzip"
)

// Create an archive file containing the contents of directory src
func Create(filename string, src string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	err = createTarGz(src, f)
	if err != nil {
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
	windowDevices := []string{"CON", "PRN", "AUX", "NUL"}
	for _, s := range windowDevices {
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

// createTarGz writes a given directory tree to a Gzipped TAR
func createTarGz(src string, writers ...io.Writer) error {

	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("TAR: %v", err.Error())
	}

	mw := io.MultiWriter(writers...)

	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	badPath := NewBadPath(true)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// walk path
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		var link string

		// return on any error
		if err != nil {
			return err
		}

		pathType := determinePathType(fi)
		// Ignore unknown types
		if pathType == TypeOther {
			fmt.Printf("WARN: Ignoring unknown path: %q\n", path)
			return nil
		}

		if pathType == TypeLink {
			link, err = os.Readlink(path)
			if err != nil {
				return err
			}

			pathDir := filepath.Dir(path)
			linkFull := filepath.Join(wd, pathDir, link)

			//			fmt.Printf("LINK: src %s (D %s) -> tgt %s (%s)\n", path, pathDir, link, linkFull)

			if badPath.IsBad(link) {
				return fmt.Errorf("invalid path: contains bad characters: %s", link)
			}

			if strings.Index(linkFull, wd) != 0 {
				return fmt.Errorf("invalid path: symlink points outside current directory: %s -> %s", path, link)
			}

			_, err := os.Stat(linkFull)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Printf("WARN: Skipping non-existent symlink: %s -> %s\n", path, link)
					return nil

				} else {
					return err
				}
			}
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, link)
		if err != nil {
			return err
		}

		// Use PAX format for utf-8 support
		header.Format = tar.FormatPAX

		if pathType == TypeLink {
			header.Typeflag = tar.TypeSymlink
			header.Linkname = link
		}

		header.Name = filepath.ToSlash(path)
		header.Linkname = filepath.ToSlash(header.Linkname)

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// No further work required for directories
		if pathType != TypeRegular {
			return nil
		}

		// Add file to archive
		f, err := os.Open(header.Name)
		if err != nil {
			return err
		}

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}

type FileType int

const (
	TypeRegular FileType = iota
	TypeLink
	TypeDir
	TypeOther
)

func determinePathType(fi os.FileInfo) FileType {
	if fi.Mode().IsRegular() {
		return TypeRegular
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return TypeLink
	}
	if fi.IsDir() {
		return TypeDir
	}
	return TypeOther
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
