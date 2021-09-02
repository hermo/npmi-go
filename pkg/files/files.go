package files

import (
	"fmt"
	"os"
	"path/filepath"
)

// DirectoryExists determines if a given path exists and is a directory or not
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}
	return info.IsDir()
}

// IsExistingFile determines if a given path exists and is a file
func IsExistingFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

// RemoveFilesNotPresentInManifest compares a real directory tree with a list of
// files to keep and removes extra files
func RemoveFilesNotPresentInManifest(directory string, filesTokeep []string) (int, error) {
	numRemoved := 0

	// Convert manifest into a map
	// TODO: Just create the manifest in map for to begin with
	m := make(map[string]bool, len(filesTokeep))
	for _, f := range filesTokeep {
		m[f] = true
	}

	return numRemoved, filepath.Walk(directory, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !(fi.Mode().IsRegular() || fi.Mode()&os.ModeSymlink != 0) {
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
