//go:build !windows

package main

import (
	"io/fs"
	"os"
	"slices"
	"strings"
	"syscall"
)

func readDirOnly(root *os.Root, name string) ([]fs.DirEntry, error) {
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	slices.SortFunc(entries, func(a, b fs.DirEntry) int { return strings.Compare(a.Name(), b.Name()) })
	return entries, err
}
