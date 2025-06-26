package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jonhadfield/soba/internal"
)

var (
	// overwritten at build time.
	version, tag, sha, buildDate string
	logger                       *log.Logger
)

func init() {
	logger = log.New(os.Stdout, fmt.Sprintf("%s: ", internal.AppName), log.Lshortfile|log.LstdFlags)
}

func main() {
	if tag != "" && buildDate != "" {
		logger.Printf("[%s-%s] %s UTC", tag, sha, buildDate)
	} else if version != "" {
		logger.Println("version", version)
	}

	if err := internal.Run(); err != nil {
		logger.Fatal(err)
	}
}
