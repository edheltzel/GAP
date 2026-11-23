//go:build !unix

package main

func lockProjects(string) (func(), error) {
	return func() {}, nil
}
