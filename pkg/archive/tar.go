package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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
func ExtractArchive(dst string, reader io.Reader) ([]string, error) {
	return untar(dst, reader)
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

		// TODO: Test if this is unnecessary
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

// untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func untar(dst string, r io.Reader) ([]string, error) {
	var manifest []string
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

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
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return nil, err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			manifest = append(manifest, target)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return nil, err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
	return manifest, nil
}

// CleanTree compares a real directory tree with a manifest and removes extra files
func CleanTree(src string, manifest []string) (int, error) {

	numRemoved := 0

	// Convert manifest into a map
	// TODO: Just create the manifest in map for to begin with
	m := make(map[string]bool, len(manifest))
	for _, f := range manifest {
		m[f] = true
	}

	return numRemoved, filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		// TODO: Add link support
		if !fi.Mode().IsRegular() {
			return nil
		}

		// Delete files not present in manifest
		if !m[file] {
			fmt.Printf("DEL: %s\n", file)
			err = os.Remove(file)
			if err != nil {
				return err
			}
			numRemoved++
		}
		return nil
	})
}
