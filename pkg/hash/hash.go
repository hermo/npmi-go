package hash

import (
	"crypto/sha256"
	b64 "encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hermo/npmi-go/pkg/files"
	"github.com/zeebo/blake3"
)

// File hashes a file using SHA-256
func File(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return hashInput(f)
}

// String hashes a given string using SHA-256
func String(str string) (string, error) {
	return hashInput(strings.NewReader(str))
}

func hashInput(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

type HashedTreeItem struct {
	Item files.TreeItem
	Hash Hash
}

type Hash []byte
type HashedTree []HashedTreeItem

func (h Hash) String() string {
	return b64.StdEncoding.EncodeToString(h)
}

func HashTree(tree *files.Tree) (*HashedTree, error) {
	hTree := make(HashedTree, 0, len(*tree))

	for _, item := range *tree {
		if !item.IsRegular() {
			continue
		}
		f, err := os.Open(item.Path)
		if err != nil {
			return nil, err
		}

		h := blake3.New()

		if _, err = io.Copy(h, f); err != nil {
			f.Close()
			return nil, err
		}
		f.Close()
		hTree = append(hTree, HashedTreeItem{
			Item: item,
			Hash: h.Sum(nil),
		})
	}
	return &hTree, nil
}
