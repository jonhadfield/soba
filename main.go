package main

import (
	"log"
	"os"
	"strings"

	"strconv"

	"runtime"

	"time"

	"github.com/carlescere/scheduler"
	"github.com/jonhadfield/soba/githosts"
)

const (
	workingDIRName = ".working"
)

var (
	logger *log.Logger
	// overwritten at build time
	version, tag, sha, buildDate string
)

func init() {
	logger = log.New(os.Stdout, "soba: ", log.Lshortfile|log.LstdFlags)
}

func getBackupInterval() int {
	backupIntervalEnv := os.Getenv("GIT_BACKUP_INTERVAL")
	var backupInterval int
	var intervalConversionErr error
	if backupIntervalEnv != "" {
		backupInterval, intervalConversionErr = strconv.Atoi(backupIntervalEnv)
		if intervalConversionErr != nil {
			logger.Fatal("GIT_BACKUP_INTERVAL must be a number.")
		}
	}
	return backupInterval
}
func main() {
	if tag != "" && buildDate != "" {
		logger.Printf("[%s-%s] %s UTC", tag, sha, buildDate)
	} else if version != "" {
		logger.Println(version)
	}
	logger.Println("starting")
	if os.Getenv("GITHUB_TOKEN") == "" && os.Getenv("GITLAB_TOKEN") == "" {
		logger.Fatal("no tokens passed. Please set environment variables GITHUB_TOKEN and/or GITLAB_TOKEN")
	}
	var backupDIR = os.Getenv("GIT_BACKUP_DIR")
	if backupDIR == "" {
		logger.Fatal("environment variable GIT_BACKUP_DIR must be set")
	} else {
		_, err := os.Stat(backupDIR)
		if os.IsNotExist(err) {
			logger.Fatalf("specified backup directory \"%s\" does not exist", backupDIR)
		}
	}

	if len(backupDIR) > 1 && strings.HasSuffix(backupDIR, "/") {
		backupDIR = backupDIR[:len(backupDIR)-1]
	}
	workingDIR := backupDIR + string(os.PathSeparator) + workingDIRName

	logger.Println("creating working directory: ", workingDIR)
	createWorkingDIRErr := os.MkdirAll(workingDIR, 0755)
	if createWorkingDIRErr != nil {
		logger.Fatal(createWorkingDIRErr)
	}

	backupInterval := getBackupInterval()

	if backupInterval != 0 {
		logger.Printf("scheduling to run every %d hours", backupInterval)
		_, schedulerErr := scheduler.Every(backupInterval).Hours().Run(execProviderBackups)
		if schedulerErr != nil {
			logger.Fatalln(schedulerErr)
		}
		runtime.Goexit()
	} else {
		execProviderBackups()
	}
}

func execProviderBackups() {
	startTime := time.Now()
	backupDIR := os.Getenv("GIT_BACKUP_DIR")
	if os.Getenv("GITLAB_TOKEN") != "" {
		logger.Println("backing up GitLab repos")
		githosts.Backup("gitlab", backupDIR)
	}

	if os.Getenv("GITHUB_TOKEN") != "" {
		logger.Println("backing up GitHub repos")
		githosts.Backup("github", backupDIR)
	}
	logger.Println("cleaning up")
	delErr := os.RemoveAll(backupDIR + string(os.PathSeparator) + workingDIRName + string(os.PathSeparator))
	if delErr != nil {
		logger.Printf("failed to delete working directory: %s",
			backupDIR+string(os.PathSeparator)+workingDIRName)
	}
	logger.Println("backups complete")
	if backupInterval := getBackupInterval(); backupInterval > 0 {
		nextBackupTime := startTime.Add(time.Duration(backupInterval) * time.Hour)
		if nextBackupTime.Before(time.Now()) {
			logger.Fatal("error: backup took longer than scheduled interval")
		}
		logger.Printf("next run scheduled for: %v", nextBackupTime)
	}
}
