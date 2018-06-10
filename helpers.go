package main

import "os"

func createDirIfAbsent(path string) error {
	return os.MkdirAll(path, 0755)
}
