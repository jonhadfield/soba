package githosts

import (
	"io"
	"os"
	"time"
)

func createDirIfAbsent(path string) error {
	return os.MkdirAll(path, 0755)
}

func getTimestamp() string {
	t := time.Now()
	return t.Format("20060102150405")
}

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
