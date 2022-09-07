package files

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"
)

func getBaseDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return path.Dir(filename)
}

func TestFileTree_Create(t *testing.T) {
	testDir, err := filepath.Abs(fmt.Sprintf("%s/../../testdata", getBaseDir()))
	if err != nil {
		t.Fatal(err)
	}

	if err = os.Chdir(testDir); err != nil {
		t.Fatal(err)
	}

	tree, err := CreateFileTree("tree")

	if err != nil {
		t.Fatal(err)
	}

	expectedItems := &FileTree{
		FileTreeItem{Path: "tree", Type: LeafTypeDir},
		FileTreeItem{Path: "tree/root.txt", Type: LeafTypeRegular},
		FileTreeItem{Path: "tree/somedir", Type: LeafTypeDir},
		FileTreeItem{Path: "tree/somedir/link1.txt", Type: LeafTypeLink},
		FileTreeItem{Path: "tree/somedir/zero.txt", Type: LeafTypeRegular},
		FileTreeItem{Path: "tree/sub", Type: LeafTypeDir},
		FileTreeItem{Path: "tree/sub/dir", Type: LeafTypeDir},
		FileTreeItem{Path: "tree/sub/dir/sub2.txt", Type: LeafTypeRegular},
		FileTreeItem{Path: "tree/sub/dir/sub3.txt", Type: LeafTypeRegular},
		FileTreeItem{Path: "tree/sub/sub1.txt", Type: LeafTypeRegular},
	}

	if len(*tree) != len(*expectedItems) {
		t.Fatalf("Expected Tree size to be %d, got %d", len(*expectedItems), len(*tree))
	}

	// We don't assume that things will be in the same order but they need to be present.
	for _, expected := range *expectedItems {
		found := false
		for _, item := range *tree {
			if item.Path == expected.Path && item.Type == expected.Type {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing entry: %v in tree %v", expected, tree)
		}
	}
}
