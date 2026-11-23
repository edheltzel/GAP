//go:build linux

package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func init() { groupAlive = linuxGroupAlive }

func linuxGroupAlive(pgid int) bool {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return syscall.Kill(-pgid, 0) == nil
	}
	for _, entry := range entries {
		name := entry.Name()
		if _, err := strconv.Atoi(name); err != nil {
			continue
		}
		stat, err := os.ReadFile("/proc/" + name + "/stat")
		if err != nil {
			continue
		}
		text := string(stat)
		end := strings.LastIndex(text, ")")
		if end < 0 || end+2 >= len(text) {
			continue
		}
		fields := strings.Fields(text[end+2:])
		if len(fields) < 3 || fields[0] == "Z" {
			continue
		}
		if group, err := strconv.Atoi(fields[2]); err == nil && group == pgid {
			return true
		}
	}
	return false
}
