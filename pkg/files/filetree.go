package files

import (
	"io/fs"
	"os"
	"path/filepath"
)

const FileTreeCapacityReservation = 50e3

type TreeType uint8

const (
	TypeRegular TreeType = 1
	TypeLink    TreeType = 2
	TypeDir     TreeType = 3
	TypeOther   TreeType = 4
)

type TreeItem struct {
	Path     string
	Type     TreeType
	FileInfo *fs.FileInfo
}

func NewTree(path string, fi *fs.FileInfo) *TreeItem {
	return &TreeItem{
		Path:     path,
		FileInfo: fi,
		Type:     determinePathType(*fi),
	}
}

func (fi *TreeItem) IsDir() bool {
	return fi.Type == TypeDir
}

func (fi *TreeItem) IsLink() bool {
	return fi.Type == TypeLink
}

func (fi *TreeItem) IsRegular() bool {
	return fi.Type == TypeRegular
}

func (fi *TreeItem) IsOther() bool {
	return fi.Type == TypeOther
}

type Tree []TreeItem

func determinePathType(fi os.FileInfo) TreeType {
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

func CreateFileTree(src string) (*Tree, error) {
	tree := make(Tree, 0, FileTreeCapacityReservation)

	err := filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		tree = append(tree, *NewTree(path, &fi))
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &tree, err
}
