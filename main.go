package main

import (
	"log"
	"os"
	"strings"

	"github.com/jonhadfield/soba/githosts"
)

const (
	workingDIRName = ".working"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, "soba: ", log.Lshortfile|log.LstdFlags)
}

func main() {
	if os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("GITLAB_TOKEN") == "" {
		logger.Fatal("no tokens passed. Please set environment variables GITHUB_TOKEN and/or GITLAB_TOKEN.")
	}
	var backupDIR = os.Getenv("GIT_BACKUP_DIR")
	if backupDIR == "" {
		logger.Fatal("environment variable GIT_BACKUP_DIR must be set.")
	} else {
		_, err := os.Stat(backupDIR)
		if os.IsNotExist(err) {
			logger.Fatalf("specified backup directory \"%s\" does not exist.", backupDIR)
		}
	}
	if len(backupDIR) > 1 && strings.HasSuffix(backupDIR, "/") {
		backupDIR = backupDIR[:len(backupDIR)-1]
	}
	workingDIR := backupDIR + string(os.PathSeparator) + workingDIRName

	logger.Println("creating working directory: ", workingDIR)
	createWorkingDIRErr := createDirIfAbsent(workingDIR)
	if createWorkingDIRErr != nil {
		logger.Fatal(createWorkingDIRErr)
	}

	logger.Println("backing up GitLab repos")
	githosts.Backup("gitlab", backupDIR)

	logger.Println("backing up GitHub repos")
	githosts.Backup("github", backupDIR)

	logger.Println("backups complete")
}
