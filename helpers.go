package main

import (
	"os"
	"strings"
)

const pathSep = string(os.PathSeparator)

func stringInStrings(single string, group []string) bool {
	for _, item := range group {
		if single == item {
			return true
		}
	}
	return false
}

func stripTrailing(input, toStrip string) string {
	if strings.HasSuffix(input, toStrip) {
		return input[:len(input)-len(toStrip)]
	}
	return input
}
