package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func cleanup(path string) error {
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

// Archive creates an archive file containing the contents of src
func Archive(filename string, src string) error {
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

// ExtractArchive uncompresses stuff
func ExtractArchive(reader io.Reader) error {
	return nil
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

	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		// TODO: Add link support
		if !fi.IsDir() && !fi.Mode().IsRegular() {
			return nil
		}

		// Clean up file path and convert directory separators to slashes for TAR
		file = filepath.ToSlash(filepath.Clean(file))

		// Directory entries should have a trailing slash
		if fi.IsDir() && file[len(file)-1:] != "/" {
			file = file + "/"
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		header.Name = file

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// No further work required for directories
		if fi.IsDir() {
			return nil
		}

		// Add file to archive
		f, err := os.Open(file)
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
