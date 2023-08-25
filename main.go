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
	appName                                = "soba"
	workingDIRName                         = ".working"
	workingDIRMode                         = 0o755
	defaultBackupsToRetain                 = 2
	defaultGitLabMinimumProjectAccessLevel = 20

	pathSep = string(os.PathSeparator)

	// env vars
	envSobaLogLevel      = "SOBA_LOG"
	envGitBackupInterval = "GIT_BACKUP_INTERVAL"
	envGitBackupDir      = "GIT_BACKUP_DIR"
	envGitHubAPIURL      = "GITHUB_APIURL"
	envGitHubBackups     = "GITHUB_BACKUPS"
	// nolint:gosec
	envGitHubToken          = "GITHUB_TOKEN"
	envGitHubOrgs           = "GITHUB_ORGS"
	envGitHubSkipUserRepos  = "GITHUB_SKIP_USER_REPOS"
	envGitHubCompare        = "GITHUB_COMPARE"
	envGitLabBackups        = "GITLAB_BACKUPS"
	envGitLabMinAccessLevel = "GITLAB_PROJECT_MIN_ACCESS_LEVEL"
	envGitLabToken          = "GITLAB_TOKEN"
	envGitLabAPIURL         = "GITLAB_APIURL"
	envGitLabCompare        = "GITLAB_COMPARE"
	envBitBucketUser        = "BITBUCKET_USER"
	envBitBucketKey         = "BITBUCKET_KEY"
	envBitBucketSecret      = "BITBUCKET_SECRET"
	envBitBucketAPIURL      = "BITBUCKET_APIURL"
	envBitBucketCompare     = "BITBUCKET_COMPARE"
	envBitBucketBackups     = "BITBUCKET_BACKUPS"
	envGiteaToken           = "GITEA_TOKEN"
	envGiteaAPIURL          = "GITEA_APIURL"
	envGiteaBackups         = "GITEA_BACKUPS"
	envGiteaCompare         = "GITEA_COMPARE"
	envGiteaOrgs            = "GITEA_ORGS"

	// provider names
	providerNameBitBucket = "BitBucket"
	providerNameGitHub    = "GitHub"
	providerNameGitLab    = "GitLab"
	providerNameGitea     = "Gitea"

	// compare types
	compareTypeRefs  = "refs"
	compareTypeClone = "clone"
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
		providerNameGitea: {
			envGiteaAPIURL,
			envGiteaToken,
		},
	}
	justTokenProviders = []string{
		providerNameGitHub,
		providerNameGitLab,
		providerNameGitea,
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

	hours, isHour := isInt(backupIntervalEnv)

	switch {
	case isHour:
		// an int represents hours
		return hours * 60
	case strings.HasSuffix(backupIntervalEnv, "h"):
		// a string ending in h represents hours
		hours, isHour = isInt(backupIntervalEnv[:len(backupIntervalEnv)-1])
		if isHour {
			return hours * 60
		}
	case strings.HasSuffix(backupIntervalEnv, "m"):
		// a string ending in m represents minutes
		minutes, isMinute := isInt(backupIntervalEnv[:len(backupIntervalEnv)-1])
		if isMinute {
			return minutes
		}
	}

	return 0
}

func getLogLevel() int {
	sobaLogLevelEnv := os.Getenv(envSobaLogLevel)

	var sobaLogLevel int

	var intervalConversionErr error

	if sobaLogLevelEnv != "" {
		sobaLogLevel, intervalConversionErr = strconv.Atoi(sobaLogLevelEnv)
		if intervalConversionErr != nil {
			logger.Fatalf("%s must be a number.", envSobaLogLevel)
		}
	}

	return sobaLogLevel
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
		logger.Println("version", version)
	}

	if err := run(); err != nil {
		logger.Fatal(err)
	}
}

func envTrue(envVar string) bool {
	res, err := strconv.ParseBool(os.Getenv(envVar))
	if err != nil {
		return false
	}

	return res
}

func displayStartupConfig() {
	if backupDIR := os.Getenv(envGitBackupDir); backupDIR != "" {
		logger.Printf("root backup directory: %s", backupDIR)
	}

	// output github config
	if ghToken := os.Getenv(envGitHubToken); ghToken != "" {
		if ghOrgs := strings.ToLower(os.Getenv(envGitHubOrgs)); ghOrgs != "" {
			logger.Printf("GitHub Organistations: %s", ghOrgs)
		}

		if envTrue(envGitHubSkipUserRepos) {
			logger.Printf("GitHub skipping user repos: true")
		}

		if strings.EqualFold(os.Getenv(envGitHubCompare), compareTypeRefs) {
			logger.Print("GitHub compare method: refs")
		} else {
			logger.Print("GitHub compare method: clone")
		}
	}

	// output gitea config
	if giteaToken := os.Getenv(envGiteaToken); giteaToken != "" {
		if giteaOrgs := strings.ToLower(os.Getenv(envGiteaOrgs)); giteaOrgs != "" {
			logger.Printf("Gitea Organistations: %s", giteaOrgs)
		}

		if giteaBackups := os.Getenv(envGiteaBackups); giteaBackups != "" {
			logger.Printf("Gitea backups to keep: %s", giteaBackups)
		}

		if strings.EqualFold(os.Getenv(envGiteaCompare), compareTypeRefs) {
			logger.Print("Gitea compare method: refs")
		} else {
			logger.Print("Gitea compare method: clone")
		}
	}

	// output gitlab config
	if glToken := os.Getenv(envGitLabToken); glToken != "" {
		if glProjectMinAccessLevel := os.Getenv(envGitLabMinAccessLevel); glProjectMinAccessLevel != "" {
			logger.Printf("GitLab project minimum access level: %s", glProjectMinAccessLevel)
		} else {
			logger.Printf("GitLab project minimum access level: %d", githosts.GitLabDefaultMinimumProjectAccessLevel)
		}

		if glBackups := os.Getenv(envGitLabBackups); glBackups != "" {
			logger.Printf("GitLab backups to keep: %s", glBackups)
		}

		if strings.EqualFold(os.Getenv(envGitLabCompare), compareTypeRefs) {
			logger.Print("GitLab compare method: refs")
		} else {
			logger.Print("GitLab compare method: clone")
		}
	}

	// output bitbucket config
	if bbUser := os.Getenv(envBitBucketUser); bbUser != "" {
		if bbBackups := os.Getenv(envBitBucketBackups); bbBackups != "" {
			logger.Printf("BitBucket backups to keep: %s", bbBackups)
		}

		if strings.ToLower(os.Getenv(envBitBucketCompare)) == compareTypeRefs {
			logger.Printf("BitBucket compare method: %s", compareTypeRefs)
		} else {
			logger.Printf("BitBucket compare method: %s", compareTypeClone)
		}
	}
}

func run() error {
	displayStartupConfig()

	var backupDIR string

	var backupDIRKeyExists bool

	backupDIR, backupDIRKeyExists = os.LookupEnv(envGitBackupDir)
	if !backupDIRKeyExists || backupDIR == "" {
		return fmt.Errorf("environment variable %s must be set", envGitBackupDir)
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
		logger.Printf("scheduling to run every %s", formatIntervalDuration(backupInterval))

		_, err = scheduler.Every(int(time.Duration(backupInterval))).Minutes().Run(execProviderBackups)
		if err != nil {
			return errors.Wrapf(err, "scheduler failed")
		}

		runtime.Goexit()
	} else {
		execProviderBackups()
	}

	return nil
}

func formatIntervalDuration(m int) string {
	if m == 0 {
		return ""
	}

	if m%60 == 0 {
		return fmt.Sprintf("%dh", m/60)
	}

	return time.Duration(int64(m) * int64(time.Minute)).String()
}

func getOrgsListFromEnvVar(envVar string) []string {
	orgsList := os.Getenv(envVar)

	if orgsList == "" {
		return []string{}
	}

	return strings.Split(orgsList, ",")
}

func execProviderBackups() {
	var err error

	startTime := time.Now()

	backupDir := os.Getenv(envGitBackupDir)

	if os.Getenv(envBitBucketUser) != "" {
		logger.Println("backing up BitBucket repos")

		var bitbucketHost *githosts.BitbucketHost

		bitbucketHost, err = githosts.NewBitBucketHost(githosts.NewBitBucketHostInput{
			Caller:           appName,
			APIURL:           os.Getenv(envBitBucketAPIURL),
			DiffRemoteMethod: os.Getenv(envBitBucketCompare),
			BackupDir:        backupDir,
			User:             os.Getenv(envBitBucketUser),
			Key:              os.Getenv(envBitBucketKey),
			Secret:           os.Getenv(envBitBucketSecret),
			BackupsToRetain:  getBackupsToRetain(envBitBucketBackups),
			LogLevel:         getLogLevel(),
		})
		if err != nil {
			logger.Fatal(err)
		}

		bitbucketHost.Backup()
	}

	if os.Getenv(envGiteaToken) != "" {
		logger.Println("backing up Gitea repos")

		var giteaHost *githosts.GiteaHost

		giteaHost, err = githosts.NewGiteaHost(githosts.NewGiteaHostInput{
			Caller:           appName,
			APIURL:           os.Getenv(envGiteaAPIURL),
			DiffRemoteMethod: os.Getenv(envGiteaCompare),
			BackupDir:        backupDir,
			Token:            os.Getenv(envGiteaToken),
			Orgs:             getOrgsListFromEnvVar(envGiteaOrgs),
			BackupsToRetain:  getBackupsToRetain(envGiteaBackups),
			LogLevel:         getLogLevel(),
		})
		if err != nil {
			logger.Fatal(err)
		}

		giteaHost.Backup()
	}

	if os.Getenv(envGitHubToken) != "" {
		logger.Println("backing up GitHub repos")

		var githubHost *githosts.GitHubHost

		githubHost, err = githosts.NewGitHubHost(githosts.NewGitHubHostInput{
			Caller:           appName,
			APIURL:           os.Getenv(envGitHubAPIURL),
			DiffRemoteMethod: os.Getenv(envGitHubCompare),
			BackupDir:        backupDir,
			Token:            os.Getenv(envGitHubToken),
			Orgs:             getOrgsListFromEnvVar(envGitHubOrgs),
			BackupsToRetain:  getBackupsToRetain(envGitHubBackups),
			SkipUserRepos:    envTrue(envGitHubSkipUserRepos),
			LogLevel:         getLogLevel(),
		})
		if err != nil {
			logger.Fatal(err)
		}

		githubHost.Backup()
	}

	if os.Getenv(envGitLabToken) != "" {
		logger.Println("backing up GitLab repos")

		var gitlabHost *githosts.GitLabHost

		gitlabHost, err = githosts.NewGitLabHost(githosts.NewGitLabHostInput{
			Caller:                appName,
			APIURL:                os.Getenv(envGitLabAPIURL),
			DiffRemoteMethod:      os.Getenv(envGitLabCompare),
			BackupDir:             backupDir,
			Token:                 os.Getenv(envGitLabToken),
			BackupsToRetain:       getBackupsToRetain(envGitLabBackups),
			ProjectMinAccessLevel: getProjectMinimumAccessLevel(),
			LogLevel:              getLogLevel(),
		})
		if err != nil {
			logger.Fatal(err)
		}

		gitlabHost.Backup()
	}

	logger.Println("cleaning up")

	// startFileRemovals := time.Now()
	delErr := os.RemoveAll(backupDir + pathSep + workingDIRName + pathSep)
	if delErr != nil {
		logger.Printf("failed to delete working directory: %s",
			backupDir+pathSep+workingDIRName)
	}

	// TODO: use a debug flag to enable this
	// logger.Printf("file removals took %s", time.Since(startFileRemovals).String())

	logger.Println("backups complete")

	if backupInterval := getBackupInterval(); backupInterval > 0 {
		nextBackupTime := startTime.Add(time.Duration(backupInterval) * time.Minute)
		if nextBackupTime.Before(time.Now()) {
			logger.Fatal("error: backup took longer than scheduled interval")
		}

		logger.Printf("next run scheduled for: %s", nextBackupTime.Format("2006-01-02 15:04:05 -0700 MST"))
	}
}

func stripTrailingLineBreak(input string) string {
	if strings.HasSuffix(input, "\n") {
		return input[:len(input)-2]
	}

	return input
}

func getProjectMinimumAccessLevel() int {
	if os.Getenv(envGitLabMinAccessLevel) == "" {
		logger.Printf("environment variable %s not set, using default of %d", envGitLabMinAccessLevel, defaultGitLabMinimumProjectAccessLevel)

		return defaultGitLabMinimumProjectAccessLevel
	}

	minimumProjectAccessLevel, err := strconv.Atoi(os.Getenv(envGitLabMinAccessLevel))
	if err != nil {
		logger.Printf("error converting environment variable %s to int so defaulting to: %d", envGitLabMinAccessLevel, defaultGitLabMinimumProjectAccessLevel)

		return defaultGitLabMinimumProjectAccessLevel
	}

	return minimumProjectAccessLevel
}

func getBackupsToRetain(envVar string) int {
	if os.Getenv(envVar) == "" {
		logger.Printf("environment variable %s not set, using default of %d", envVar, defaultBackupsToRetain)

		return defaultBackupsToRetain
	}

	backupsToKeep, err := strconv.Atoi(os.Getenv(envVar))
	if err != nil {
		logger.Printf("error converting environment variable %s to int so defaulting to: %d", envVar, defaultBackupsToRetain)

		return defaultBackupsToRetain
	}

	return backupsToKeep
}

func isInt(i string) (int, bool) {
	if val, err := strconv.Atoi(i); err == nil {
		return val, true
	}

	return 0, false
}
