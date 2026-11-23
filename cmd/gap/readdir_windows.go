//go:build windows

package main

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
)

func readDirOnly(root *os.Root, name string) ([]fs.DirEntry, error) {
	info, err := root.Stat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "readdir", Path: name, Err: errors.New("not a directory")}
	}
	f, err := root.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, err
}
