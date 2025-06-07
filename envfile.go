// GetEnvOrFile returns the value of the environment variable if set, otherwise if a corresponding _FILE variable is set, reads the value from the file at that path.
// If both are set, the environment variable takes precedence.
package main

import (
	"fmt"
	"os"
	"strings"
)

// GetEnvOrFile returns the value of the environment variable if set, otherwise if a corresponding _FILE variable is set, reads the value from the file at that path.
func GetEnvOrFile(envVar string) (string, bool) {
	val, exists := os.LookupEnv(envVar)
	if exists {
		if val != "" {
			return val, exists
		}

		return "", exists
	}

	fileEnv := envVar + "_FILE"

	filePath := os.Getenv(fileEnv)
	if filePath != "" {
		b, err := os.ReadFile(strings.TrimSpace(filePath))
		if err == nil {
			return strings.TrimSpace(string(b)), true
		}

		if os.IsNotExist(err) {
			fmt.Printf("File %s does not exist, returning empty string\n", filePath)
			return "", false
		} else {
			fmt.Printf("Error reading file %s: %v\n", filePath, err)
			return "", false
		}
	}

	return "", false
}
