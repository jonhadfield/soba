package githosts

import (
	"io"
	"os"
	"strings"
	"time"
)

const pathSep = string(os.PathSeparator)

func createDirIfAbsent(path string) error {
	return os.MkdirAll(path, 0755)
}

func getTimestamp() string {
	t := time.Now()
	return t.Format("20060102150405")
}

func stripTrailing(input string, toStrip string) string {
	if strings.HasSuffix(input, toStrip) {
		return input[:len(input)-len(toStrip)]
	}
	return input
}

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	defer func() {
		if cErr := f.Close(); cErr != nil {
			logger.Printf("warn: failed to close: %s", name)
		}
	}()

	if err != nil {
		return false, err
	}

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
