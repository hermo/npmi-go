package archive

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/klauspost/pgzip"
)

type TarOptions struct {
	AllowAbsolutePaths   bool
	AllowDoubleDotPaths  bool
	AllowLinksOutsideCwd bool
}

// Create an archive file containing the contents of directory src
func Create(filename string, src string, options *TarOptions) (warnings []string, err error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("TAR: %v", err.Error())
	}

	gzw := pgzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	badPath := NewBadPath(options.AllowDoubleDotPaths, options.AllowAbsolutePaths)

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	// walk path
	err = filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		var link string

		// return on any error
		if err != nil {
			return err
		}

		pathType := determinePathType(fi)
		// Ignore unknown types
		if pathType == TypeOther {
			warnings = append(warnings, fmt.Sprintf("Ignored unknown path: %q\n", path))
			return nil
		}

		if pathType == TypeLink {
			link, err = os.Readlink(path)
			if err != nil {
				return err
			}

			pathDir := filepath.Dir(path)
			var linkFull string

			if filepath.IsAbs(link) {
				linkFull = link
			} else {
				linkFull = filepath.Join(wd, pathDir, link)
			}

			if badPath.IsBad(link) {
				return fmt.Errorf("invalid path: contains bad characters: %s", link)
			}

			if strings.Index(linkFull, wd) != 0 {
				err := fmt.Errorf("invalid path: symlink points outside current directory: %s -> %s", path, link)
				if !options.AllowLinksOutsideCwd {
					return err
				} else {
					warnings = append(warnings, fmt.Sprintf("%v", err))
				}
			}

			_, err := os.Stat(linkFull)
			if err != nil {
				if os.IsNotExist(err) {
					warnings = append(warnings, fmt.Sprintf("Skipped non-existent symlink: %s -> %s\n", path, link))
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
	return warnings, err
}

type badpath struct {
	allowDoubleDot           bool
	allowAbsolutePaths       bool
	disallowedFirstCharRegex *regexp.Regexp
}

func NewBadPath(allowDoubleDot bool, allowAbsolutePaths bool) *badpath {
	return &badpath{allowDoubleDot, allowAbsolutePaths, regexp.MustCompile(`^[\x00-\x1F\s!"#$%&'()*+,\-:;<=>?@[\]^_\x60{|}~]`)}
}

func (bp *badpath) IsBad(path string) bool {
	if !bp.allowAbsolutePaths && filepath.IsAbs(path) {
		return true
	}

	if strings.ContainsAny(path, "<|>:\"*?\\") {
		return true
	}

	if bp.disallowedFirstCharRegex.MatchString(path) {
		return true
	}

	if !bp.allowDoubleDot && strings.Contains(path, "..") {
		return true
	}

	return false
}

// Extract all files from an archive to current directory
func Extract(reader io.Reader, options *TarOptions) (manifest []string, warnings []string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}

	gzr, err := pgzip.NewReaderN(reader, 500e3, 50)
	if err != nil {
		return nil, nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	badPath := NewBadPath(false, false)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, nil, err
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
			return nil, nil, fmt.Errorf("invalid path: contains bad characters")
		}

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, header.FileInfo().Mode()); err != nil {
					return nil, nil, err
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
				return nil, nil, err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return nil, nil, err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
			if err = os.Chtimes(target, time.Now(), header.FileInfo().ModTime()); err != nil {
				return nil, nil, err
			}

			if err = os.Chmod(target, header.FileInfo().Mode()); err != nil {
				return nil, nil, err
			}

			manifest = append(manifest, target)

		case tar.TypeSymlink:
			reldest := filepath.ToSlash(header.Name)
			dest := filepath.Join(cwd, filepath.ToSlash(header.Name))
			source := filepath.ToSlash(header.Linkname)

			if source[0] == '/' {
				err = fmt.Errorf("invalid path: symlink with absolute path: %s -> %s", dest, source)

				if !options.AllowAbsolutePaths {
					return nil, nil, err
				} else {
					warnings = append(warnings, fmt.Sprintf("%v", err))
				}
			}

			dir := filepath.Dir(dest)
			resolvedTarget := filepath.Join(dir, source)

			if !strings.HasPrefix(resolvedTarget, cwd) {
				err = fmt.Errorf("invalid path: %s -> %s points outside cwd", reldest, source)
				if !options.AllowLinksOutsideCwd {
					return nil, nil, err
				} else {
					warnings = append(warnings, fmt.Sprintf("%v", err))
				}
			}

			err = syncSymLink(source, dest)
			if err != nil {
				return nil, nil, fmt.Errorf("syncing symlink failed: %v", err)
			}

			// symlink timestamps are not preserved
			// see https://stackoverflow.com/questions/54762079/how-to-change-timestamp-for-symbol-link-using-golang
			manifest = append(manifest, target)

		default:
			return nil, nil, fmt.Errorf("unsupported file type: %+v", header)
		}

	}
	return manifest, warnings, nil
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
