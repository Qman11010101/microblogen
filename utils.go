package main

import "os"

// PathExists checks if a file exists at the given path.
func PathExists(name string) bool {
	_, err := os.Stat(name)
	return !os.IsNotExist(err)
}
