package files

import "os"

// IsExistingDir determines if a given path exists and is a directory or not
func IsExistingDir(path string) bool {
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

// isExistingDir determines if a given path exists and is a directory or not
func isExistingDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}
	return info.IsDir()
}

// isFile determines if a given path exists and is a file
func isFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}
