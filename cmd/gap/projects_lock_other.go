//go:build !unix

package main

func lockPath(string) (func(), error) {
	return func() {}, nil
}

func lockProjects(string) (func(), error) {
	return func() {}, nil
}
