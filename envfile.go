// getEnvOrFile returns the value of the environment variable if set, otherwise if a corresponding _FILE variable is set, reads the value from the file at that path.
// If both are set, the environment variable takes precedence.
package main

import (
	"os"
	"strings"
)

// getEnvOrFile returns the value of the environment variable if set, otherwise if a corresponding _FILE variable is set, reads the value from the file at that path.
func getEnvOrFile(envVar string) string {
	val := os.Getenv(envVar)
	if val != "" {
		return val
	}

	fileEnv := envVar + "_FILE"

	filePath := os.Getenv(fileEnv)
	if filePath != "" {
		b, err := os.ReadFile(strings.TrimSpace(filePath))
		if err == nil {
			return strings.TrimSpace(string(b))
		}
	}

	return ""
}
