package archive

import (
	"os"

	"github.com/mholt/archiver/v3"
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

// CompressModules compresses stuff
func CompressModules() (string, error) {

	filename := "test.tar.gz"

	err := cleanup(filename)
	if err != nil {
		return "", err
	}

	err = archiver.Archive([]string{"pkg"}, filename)
	if err != nil {
		return "", err
	}
	return filename, nil
}
