//go:build windows

package readdir

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
)

func Only(root *os.Root, name string) ([]fs.DirEntry, error) {
	info, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: name, Err: errors.New("not a directory")}
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, err
}
