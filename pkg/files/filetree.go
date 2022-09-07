package files

import (
	"io/fs"
	"os"
	"path/filepath"
)

type FileTreeType uint8

const (
	LeafTypeRegular FileTreeType = 1
	LeafTypeLink    FileTreeType = 2
	LeafTypeDir     FileTreeType = 3
	LeafTypeOther   FileTreeType = 4
)

type FileTreeItem struct {
	Path     string
	Type     FileTreeType
	FileInfo *fs.FileInfo
}

type FileTree []FileTreeItem

func determinePathType(fi os.FileInfo) FileTreeType {
	if fi.Mode().IsRegular() {
		return LeafTypeRegular
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return LeafTypeLink
	}
	if fi.IsDir() {
		return LeafTypeDir
	}
	return LeafTypeOther
}

func CreateFileTree(src string) (*FileTree, error) {
	// walk path

	tree := make(FileTree, 0, 50e3)
	err := filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		// return on any error
		if err != nil {
			return err
		}
		tree = append(tree, FileTreeItem{
			Path:     path,
			Type:     determinePathType(fi),
			FileInfo: &fi,
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &tree, err
}
