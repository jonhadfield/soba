package internal

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// GetEnvOrFile returns the value of the environment variable if set, otherwise
// if a corresponding _FILE variable is set, reads the value from the file at
// that path. Files larger than maxEnvFileSize are rejected to prevent
// accidentally loading huge or device-backed paths into memory.
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
	if filePath == "" {
		return "", false
	}

	cleanPath := filepath.Clean(strings.TrimSpace(filePath))

	f, err := os.Open(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Printf("file %s does not exist", filePath)
		} else {
			logger.Printf("error opening file %s: %v", filePath, err)
		}

		return "", false
	}
	defer func() { _ = f.Close() }()

	b, err := io.ReadAll(io.LimitReader(f, maxEnvFileSize+1))
	if err != nil {
		logger.Printf("error reading file %s: %v", filePath, err)

		return "", false
	}

	if int64(len(b)) > maxEnvFileSize {
		logger.Printf("file %s exceeds maximum size of %d bytes", filePath, maxEnvFileSize)

		return "", false
	}

	return strings.TrimSpace(string(b)), true
}
