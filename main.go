package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/carlescere/scheduler"
	"github.com/jonhadfield/githosts-utils"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
)

const (
	appName        = "soba"
	workingDIRName = ".working"
	workingDIRMode = 0o755

	pathSep = string(os.PathSeparator)

	// env vars
	envGitBackupInterval = "GIT_BACKUP_INTERVAL"
	envGitBackupDir      = "GIT_BACKUP_DIR"
	envGitHubToken       = "GITHUB_TOKEN"
	envGitHubOrgs        = "GITHUB_ORGS"
	envGitHubCompare     = "GITHUB_COMPARE"
	envGitLabToken       = "GITLAB_TOKEN"
	envGitLabAPIURL      = "GITLAB_APIURL"
	envGitLabCompare     = "GITLAB_COMPARE"
	envBitBucketUser     = "BITBUCKET_USER"
	envBitBucketKey      = "BITBUCKET_KEY"
	envBitBucketSecret   = "BITBUCKET_SECRET"
	envBitBucketAPIURL   = "BITBUCKET_APIURL"
	envBitBucketCompare  = "BITBUCKET_COMPARE"

	// provider names
	providerNameGitHub    = "GitHub"
	providerNameGitLab    = "GitLab"
	providerNameBitBucket = "BitBucket"
)

var (
	logger *log.Logger
	// overwritten at build time.
	version, tag, sha, buildDate string

	enabledProviderAuth = map[string][]string{
		providerNameGitHub: {
			envGitHubToken,
		},
		providerNameGitLab: {
			envGitLabToken,
		},
		providerNameBitBucket: {
			envBitBucketUser,
			envBitBucketKey,
			envBitBucketSecret,
		},
	}
	justTokenProviders = []string{
		providerNameGitHub,
		providerNameGitLab,
	}
	userAndPasswordProviders = []string{
		providerNameBitBucket,
	}
	numUserDefinedProviders int64
)

func init() {
	logger = log.New(os.Stdout, fmt.Sprintf("%s: ", appName), log.Lshortfile|log.LstdFlags)
}

func getBackupInterval() int {
	backupIntervalEnv := os.Getenv(envGitBackupInterval)

	var backupInterval int

	var intervalConversionErr error

	if backupIntervalEnv != "" {
		backupInterval, intervalConversionErr = strconv.Atoi(backupIntervalEnv)
		if intervalConversionErr != nil {
			logger.Fatalf("%s must be a number.", envGitBackupInterval)
		}
	}

	return backupInterval
}

func checkProviderFactory(provider string) func() {
	retFunc := func() {
		var outputErrs strings.Builder
		// tokenOnlyProviders
		if slices.Contains(justTokenProviders, provider) {
			for _, param := range enabledProviderAuth[provider] {
				val, exists := os.LookupEnv(param)
				if exists {
					if strings.Trim(val, " ") == "" {
						_, _ = fmt.Fprintf(&outputErrs, "%s parameter '%s' is not defined.\n", provider, param)
					} else {
						numUserDefinedProviders++
					}
				}
			}
		}

		// userAndPasswordProviders
		if slices.Contains(userAndPasswordProviders, provider) {
			var firstParamFound bool

			for _, param := range enabledProviderAuth[provider] {
				val, exists := os.LookupEnv(param)
				if firstParamFound && !exists {
					_, _ = fmt.Fprintf(&outputErrs, "all parameters for '%s' are required.\n", provider)
				}

				if exists {
					firstParamFound = true

					if val == "" {
						_, _ = fmt.Fprintf(&outputErrs, "%s parameter '%s' is not defined.\n", provider, param)
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

	if err := run(); err != nil {
		logger.Fatal(err)
	}
}

func run() error {
	logger.Println("starting")

	var backupDIR string

	var backupDIRKeyExists bool

	backupDIR, backupDIRKeyExists = os.LookupEnv(envGitBackupDir)
	if !backupDIRKeyExists || backupDIR == "" {
		return errors.New("environment variable GIT_BACKUP_DIR must be set")
	}

	if _, githubOrgsKeyExists := os.LookupEnv(envGitHubOrgs); githubOrgsKeyExists {
		if _, githubTokenExists := os.LookupEnv(envGitHubToken); !githubTokenExists {
			return fmt.Errorf("environment variable %s must be set if %s is set", envGitHubToken, envGitHubOrgs)
		}
	}

	backupDIR = stripTrailingLineBreak(backupDIR)

	_, err := os.Stat(backupDIR)
	if os.IsNotExist(err) {
		return errors.Wrap(err, fmt.Sprintf("specified backup directory \"%s\" does not exist", backupDIR))
	}

	if err = checkProvidersDefined(); err != nil {
		logger.Fatal("no providers defined")
	}

	if len(backupDIR) > 1 && strings.HasSuffix(backupDIR, "/") {
		backupDIR = backupDIR[:len(backupDIR)-1]
	}

	workingDIR := backupDIR + pathSep + workingDIRName

	logger.Println("creating working directory:", workingDIR)
	createWorkingDIRErr := os.MkdirAll(workingDIR, workingDIRMode)

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

		_, err = scheduler.Every(int(time.Duration(backupInterval))).Hours().Run(execProviderBackups)
		if err != nil {
			return errors.Wrapf(err, "scheduler failed")
		}

		runtime.Goexit()
	} else {
		execProviderBackups()
	}

	return nil
}

func execProviderBackups() {
	var err error

	startTime := time.Now()
	backupDIR := os.Getenv(envGitBackupDir)

	if os.Getenv(envBitBucketUser) != "" {
		logger.Println("backing up BitBucket repos")

		err = githosts.Backup("bitbucket", backupDIR, os.Getenv(envBitBucketAPIURL), os.Getenv(envBitBucketCompare))
		if err != nil {
			logger.Fatal(err)
		}
	}

	if os.Getenv(envGitLabToken) != "" {
		logger.Println("backing up GitLab repos")

		err = githosts.Backup("gitlab", backupDIR, os.Getenv(envGitLabAPIURL), os.Getenv(envGitLabCompare))
		if err != nil {
			logger.Fatal(err)
		}
	}

	if os.Getenv(envGitHubToken) != "" {
		logger.Println("backing up GitHub repos")
		err = githosts.Backup("github", backupDIR, os.Getenv("GITHUB_APIURL"), os.Getenv(envGitHubCompare))
		if err != nil {
			logger.Fatal(err)
		}
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

func stripTrailingLineBreak(input string) string {
	if strings.HasSuffix(input, "\n") {
		return input[:len(input)-2]
	}

	return input
}
