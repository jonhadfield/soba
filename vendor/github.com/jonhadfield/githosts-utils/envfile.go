// getEnvOrFile returns the value of the environment variable if set, otherwise if a corresponding _FILE variable is set, reads the value from the file at that path.
// If both are set, the environment variable takes precedence.
package githosts

import (
	"fmt"
	"os"
	"strings"
)

// getEnvOrFile returns the value of the environment variable if set, otherwise if a corresponding _FILE variable is set, reads the value from the file at that path.
func getEnvOrFile(envVar string) string {
	ev := os.Environ()
	for k, v := range ev {
		fmt.Printf("Environment variable %d: %s=%s\n", k, v, os.Getenv(v))
	}

	val := os.Getenv(envVar)
	fmt.Printf("getEnvOrFile: envVar=%s, value=%s\n", envVar, val)
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
