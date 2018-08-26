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

func stripTrailingLineBreak(input string) string {
	if strings.HasSuffix(input, "\n") {
		return input[:len(input)-2]
	}
	return input
}
