package main

import (
	"log"
	"os"
	"strings"

	"strconv"

	"github.com/jonhadfield/soba/githosts"
	"github.com/whiteshtef/clockwork"
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

func validateStringTime(input string) bool {
	startTimeParts := strings.Split(input, ":")
	if len(startTimeParts) == 2 {
		if _, hrConvErr := strconv.Atoi(startTimeParts[0]); hrConvErr != nil {
			return false
		}
		if _, minConvErr := strconv.Atoi(startTimeParts[1]); minConvErr != nil {
			return false
		}
		return true
	}
	return false
}

func main() {
	if tag != "" && buildDate != "" {
		logger.Printf("[%s-%s] %s UTC", tag, sha, buildDate)
	} else if version != "" {
		logger.Println(version)
	}
	logger.Println("starting")
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
	backupIntervalEnv := os.Getenv("GIT_BACKUP_INTERVAL")
	backupStartTimeEnv := os.Getenv("GIT_BACKUP_START_TIME")
	var backupInterval int
	var intervalConversionErr error
	if backupIntervalEnv != "" {
		backupInterval, intervalConversionErr = strconv.Atoi(backupIntervalEnv)
		if intervalConversionErr != nil {
			logger.Fatal("GIT_BACKUP_INTERVAL must be a number.")
		}
	}
	var backupStartTime string
	if backupStartTimeEnv != "" {
		if validateStringTime(backupStartTimeEnv) {
			backupStartTime = backupStartTimeEnv
		} else {
			logger.Fatal("GIT_BACKUP_START_TIME is invalid. Please use HH:MM format.")
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

	if backupStartTime != "" || backupInterval != 0 {
		scheduler := clockwork.NewScheduler()
		// if start time only, then schedule to run once
		if backupStartTime != "" && backupInterval == 0 {
			scheduler.Schedule().At(backupStartTime).Do(execProviderBackups)
		}
		// if interval only, then schedule and start now
		if backupStartTime == "" && backupInterval > 0 {
			scheduler.Schedule().Every(backupInterval).Hours().Do(execProviderBackups)

		}
		// if start time and interval then schedule
		if backupStartTime == "" && backupInterval > 0 {
			scheduler.Schedule().Every(backupInterval).Hours().At(backupStartTime).Do(execProviderBackups)
		}
		scheduler.Run()
	} else {
		execProviderBackups()
	}
}

func execProviderBackups() {
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
}
