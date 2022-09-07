package files

import (
	"os"
	"path/filepath"
)

type FileTreeType uint8

const (
	LeafTypeRegular FileTreeType = 1
	LeafTypeLink    FileTreeType = 2
	LeafTypeDir     FileTreeType = 2
	LeafTypeOther   FileTreeType = 2
)

type FileTreeItem struct {
	Path string
	Type FileTreeType
}

type FileTree []FileTreeItem

type FileType int

const (
	TypeRegular FileType = iota
	TypeLink
	TypeDir
	TypeOther
)

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
			Path: path,
			Type: determinePathType(fi),
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &tree, err
}
