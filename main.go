package main

import (
	"log"
	"os"
	"strings"

	"strconv"

	"time"

	"fmt"

	"runtime"

	"github.com/carlescere/scheduler"
	"github.com/jonhadfield/githosts-utils"
	"github.com/pkg/errors"
)

const (
	workingDIRName = ".working"
)

var (
	logger *log.Logger
	// overwritten at build time
	version, tag, sha, buildDate string

	enabledProviderAuth = map[string][]string{
		"GitHub": {
			"GITHUB_TOKEN",
		},
		"GitLab": {
			"GITLAB_TOKEN",
		},
		"BitBucket": {
			"BITBUCKET_USER",
			"BITBUCKET_APP_PASSWORD",
		},
	}
	justTokenProviders = []string{
		"GitHub",
		"GitLab",
	}
	userAndPasswordProviders = []string{
		"BitBucket",
	}
	numUserDefinedProviders int64
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

func checkProviderFactory(provider string) func() {
	retFunc := func() {
		var outputErrs strings.Builder
		//tokenOnlyProviders
		if stringInStrings(provider, justTokenProviders) {
			for _, param := range enabledProviderAuth[provider] {
				val, exists := os.LookupEnv(param)
				if exists {
					if strings.Trim(val, " ") == "" {
						fmt.Fprintf(&outputErrs, "%s parameter '%s' is not defined.\n", provider, param)
					} else {
						numUserDefinedProviders++
					}
				}

			}
		}
		//userAndPasswordProviders
		if stringInStrings(provider, userAndPasswordProviders) {
			var firstParamFound bool
			for _, param := range enabledProviderAuth[provider] {
				val, exists := os.LookupEnv(param)
				if firstParamFound && !exists {
					fmt.Fprintf(&outputErrs, "both parameters for '%s' are required.\n", provider)
				}
				if exists {
					firstParamFound = true
					if val == "" {
						fmt.Fprintf(&outputErrs, "%s parameter '%s' is not defined.\n", provider, param)
					} else {
						numUserDefinedProviders++
					}
				}
			}
		}
		if outputErrs.Len() > 0 {
			logger.Fatalln(outputErrs.String())
		}
	}
	return retFunc
}

func checkProvidersDefined() error {
	for provider := range enabledProviderAuth {
		checkProviderFactory(provider)()
	}
	if numUserDefinedProviders == 0 {
		return errors.New("no providers defined")
	}
	return nil
}

func main() {
	if tag != "" && buildDate != "" {
		logger.Printf("[%s-%s] %s UTC", tag, sha, buildDate)
	} else if version != "" {
		logger.Println(version)
	}
	err := run()
	if err != nil {
		logger.Fatal(err)
	}
}

func run() error {
	logger.Println("starting")
	var backupDIR string
	var backupDIRKeyExists bool
	backupDIR, backupDIRKeyExists = os.LookupEnv("GIT_BACKUP_DIR")
	if !backupDIRKeyExists || backupDIR == "" {
		return fmt.Errorf("environment variable GIT_BACKUP_DIR must be set")
	}
	backupDIR = stripTrailingLineBreak(backupDIR)
	_, err := os.Stat(backupDIR)
	if os.IsNotExist(err) {
		return fmt.Errorf("specified backup directory \"%s\" does not exist", backupDIR)
	}

	if checkProvidersDefined() != nil {
		logger.Fatal("no providers defined")
	}
	if len(backupDIR) > 1 && strings.HasSuffix(backupDIR, "/") {
		backupDIR = backupDIR[:len(backupDIR)-1]
	}
	workingDIR := backupDIR + pathSep + workingDIRName

	logger.Println("creating working directory:", workingDIR)
	createWorkingDIRErr := os.MkdirAll(workingDIR, 0755)
	if createWorkingDIRErr != nil {
		logger.Fatal(createWorkingDIRErr)
	}

	backupInterval := getBackupInterval()

	if backupInterval != 0 {
		hourOutput := "hour"
		if backupInterval > 1 {
			hourOutput = "hours"
		}
		logger.Printf("scheduling to run every %d %s", backupInterval, hourOutput)

		_, schedulerErr := scheduler.Every(time.Duration(backupInterval)).Hours().Run(execProviderBackups)
		if schedulerErr != nil {
			return schedulerErr
		}
		runtime.Goexit()
	} else {
		execProviderBackups()
	}
	return nil
}

func execProviderBackups() {
	startTime := time.Now()
	backupDIR := os.Getenv("GIT_BACKUP_DIR")
	if os.Getenv("BITBUCKET_USER") != "" {
		logger.Println("backing up BitBucket repos")
		githosts.Backup("bitbucket", backupDIR)
	}

	if os.Getenv("GITLAB_TOKEN") != "" {
		logger.Println("backing up GitLab repos")
		githosts.Backup("gitlab", backupDIR)
	}

	if os.Getenv("GITHUB_TOKEN") != "" {
		logger.Println("backing up GitHub repos")
		githosts.Backup("github", backupDIR)
	}

	logger.Println("cleaning up")
	delErr := os.RemoveAll(backupDIR + pathSep + workingDIRName + pathSep)
	if delErr != nil {
		logger.Printf("failed to delete working directory: %s",
			backupDIR+pathSep+workingDIRName)
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
