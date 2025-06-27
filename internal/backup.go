package internal

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/jonhadfield/githosts-utils"
	"gitlab.com/tozd/go/errors"
)

type BackupResults struct {
	StartedAt  sobaTime                 `json:"started_at"`
	FinishedAt sobaTime                 `json:"finished_at"`
	Results    *[]ProviderBackupResults `json:"results,omitempty"`
}

func execProviderBackups() {
	backupDir, _ := GetEnvOrFile(envGitBackupDir)

	if httpClient == nil {
		httpClient = getHTTPClient(os.Getenv(envSobaLogLevel))
	}

	backupResults := BackupResults{
		StartedAt: sobaTime{
			Time: time.Now(),
			f:    time.RFC3339,
		},
	}

	var providerBackupResults []ProviderBackupResults

	if bbToken, exists := GetEnvOrFile(envBitBucketAPIToken); exists && bbToken != "" {
		providerBackupResults = append(providerBackupResults, *Bitbucket(backupDir))
	}

	if giteaToken, exists := GetEnvOrFile(envGiteaToken); exists && giteaToken != "" {
		providerBackupResults = append(providerBackupResults, *Gitea(backupDir))
	}

	if ghToken, exists := GetEnvOrFile(envGitHubToken); exists && ghToken != "" {
		providerBackupResults = append(providerBackupResults, *GitHub(backupDir))
	}

	if glToken, exists := GetEnvOrFile(envGitLabToken); exists && glToken != "" {
		providerBackupResults = append(providerBackupResults, *Gitlab(backupDir))
	}

	if azureDevOpsUserName, exists := GetEnvOrFile(envAzureDevOpsUserName); exists && azureDevOpsUserName != "" {
		providerBackupResults = append(providerBackupResults, *AzureDevOps(backupDir))
	}

	logger.Println("cleaning up")

	delErr := os.RemoveAll(backupDir + pathSep + workingDIRName + pathSep)
	if delErr != nil {
		logger.Printf("failed to delete working directory: %s",
			backupDir+pathSep+workingDIRName)
	}

	backupResults.Results = &providerBackupResults
	backupResults.FinishedAt = sobaTime{
		Time: time.Now(),
		f:    time.RFC3339,
	}

	succeeded, failed := getBackupsStats(backupResults)

	switch {
	case succeeded == 0 && failed >= 0:
		logger.Println("all backups failed")
	case succeeded > 0 && failed > 0:
		logger.Println("backups completed with errors")
	default:
		logger.Println("backups complete")
	}

	notify(backupResults, succeeded, failed)

	if job != nil {
		nextRun, _ := job.NextRun()
		logger.Printf("next Run scheduled for: %s", nextRun.Format("2006-01-02 15:04:05 -0700 MST"))
	} else if failed > 0 { // if no interval is set then exit
		os.Exit(1)
	}
}

func displayStartupConfig() {
	if backupDIR, exists := GetEnvOrFile(envGitBackupDir); exists && backupDIR != "" {
		logger.Printf("root backup directory: %s", backupDIR)
	}

	// output github config
	if ghToken, exists := GetEnvOrFile(envGitHubToken); exists && ghToken != "" { // nolint: nestif
		if ghOrgs, orgsExists := GetEnvOrFile(envGitHubOrgs); orgsExists && strings.ToLower(ghOrgs) != "" {
			logger.Printf("GitHub Organistations: %s", strings.ToLower(ghOrgs))
		}

		if _, exists = GetEnvOrFile(envGitHubSkipUserRepos); exists && envTrue(envGitHubSkipUserRepos) {
			logger.Printf("GitHub skipping user repos: true")
		}

		var compare string
		if compare, exists = GetEnvOrFile(envGitHubCompare); exists && strings.EqualFold(compare, compareTypeRefs) {
			logger.Print("GitHub compare method: refs")
		} else {
			logger.Print("GitHub compare method: clone")
		}

		if _, exists = GetEnvOrFile(envGitHubBackupLFS); exists && envTrue(envGitHubBackupLFS) {
			logger.Printf("GitHub backup LFS: true")
		}
	}

	// output gitea config
	if giteaToken, exists := GetEnvOrFile(envGiteaToken); exists && giteaToken != "" { // nolint: nestif
		if giteaOrgs, orgsExists := GetEnvOrFile(envGiteaOrgs); orgsExists && strings.ToLower(giteaOrgs) != "" {
			logger.Printf("Gitea Organistations: %s", strings.ToLower(giteaOrgs))
		}

		if giteaBackups, backupsExists := GetEnvOrFile(envGiteaBackups); backupsExists && giteaBackups != "" {
			logger.Printf("Gitea backups to keep: %s", giteaBackups)
		}

		var compare string
		if compare, exists = GetEnvOrFile(envGiteaCompare); exists && strings.EqualFold(compare, compareTypeRefs) {
			logger.Print("Gitea compare method: refs")
		} else {
			logger.Print("Gitea compare method: clone")
		}

		if _, exists = GetEnvOrFile(envGiteaBackupLFS); exists && envTrue(envGiteaBackupLFS) {
			logger.Printf("Gitea backup LFS: true")
		}
	}

	// output gitlab config
	if glToken, exists := GetEnvOrFile(envGitLabToken); exists && glToken != "" { // nolint: nestif
		glProjectMinAccessLevel, minAccessExists := GetEnvOrFile(envGitLabMinAccessLevel)
		if !minAccessExists || glProjectMinAccessLevel == "" {
			logger.Printf("GitLab project minimum access level: %d", githosts.GitLabDefaultMinimumProjectAccessLevel)
		} else {
			logger.Printf("GitLab project minimum access level: %s", glProjectMinAccessLevel)
		}

		if glBackups, backupsExists := GetEnvOrFile(envGitLabBackups); backupsExists && glBackups != "" {
			logger.Printf("GitLab backups to keep: %s", glBackups)
		}

		compareMethod := "clone"

		var compare string
		if compare, exists = GetEnvOrFile(envGitLabCompare); exists && strings.EqualFold(compare, compareTypeRefs) {
			compareMethod = "refs"
		}

		logger.Printf("GitLab compare method: %s", compareMethod)

		if _, exists = GetEnvOrFile(envGitLabBackupLFS); exists && envTrue(envGitLabBackupLFS) {
			logger.Printf("Gitlab backup LFS: true")
		}
	}

	// output bitbucket config
	if bbUser, exists := GetEnvOrFile(envBitBucketEmail); exists && bbUser != "" {
		if bbBackups, backupsExists := GetEnvOrFile(envBitBucketBackups); backupsExists && bbBackups != "" {
			logger.Printf("BitBucket backups to keep: %s", bbBackups)
		}

		if compare, exists := GetEnvOrFile(envBitBucketCompare); exists && strings.ToLower(compare) == compareTypeRefs {
			logger.Printf("BitBucket compare method: %s", compareTypeRefs)
		} else {
			logger.Printf("BitBucket compare method: %s", compareTypeClone)
		}

		if _, exists = GetEnvOrFile(envBitBucketBackupLFS); exists && envTrue(envBitBucketBackupLFS) {
			logger.Printf("BitBucket backup LFS: true")
		}
	}

	// output azure devops config
	if azureDevOpsUserName, exists := GetEnvOrFile(envAzureDevOpsUserName); exists && azureDevOpsUserName != "" {
		if ghOrgs, orgsExists := GetEnvOrFile(envAzureDevOpsOrgs); orgsExists && strings.ToLower(ghOrgs) != "" {
			logger.Printf("Azure DevOps Organistations: %s", strings.ToLower(ghOrgs))
		}

		if compare, exists := GetEnvOrFile(envAzureDevOpsCompare); exists && strings.EqualFold(compare, compareTypeRefs) {
			logger.Print("Azure DevOps compare method: refs")
		} else {
			logger.Print("Azure DevOps compare method: clone")
		}

		if _, exists = GetEnvOrFile(envAzureDevOpsBackupLFS); exists && envTrue(envAzureDevOpsBackupLFS) {
			logger.Printf("Azure DevOps backup LFS: true")
		}
	}
}

func getBackupInterval() int {
	backupIntervalEnv := os.Getenv(envGitBackupInterval)

	hours, isHour := isInt(backupIntervalEnv)

	switch {
	case isHour:
		// an int represents hours
		return hours * minutesPerHour
	case strings.HasSuffix(backupIntervalEnv, "h"):
		// a string ending in h represents hours
		hours, isHour = isInt(backupIntervalEnv[:len(backupIntervalEnv)-1])
		if isHour {
			return hours * minutesPerHour
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

func checkProviderFactory(provider string) func() {
	retFunc := func() {
		var outputErrs strings.Builder
		// tokenOnlyProviders
		if slices.Contains(justTokenProviders, provider) {
			for _, param := range enabledProviderAuth[provider] {
				val, exists := GetEnvOrFile(param)
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
		if slices.Contains(userAndPasswordProviders, provider) { // nolint: nestif
			var foundCount, totalCount int
			for _, param := range enabledProviderAuth[provider] {
				totalCount++

				val, exists := GetEnvOrFile(param)
				if exists && strings.Trim(val, " ") != "" {
					foundCount++
				}
			}

			if foundCount > 0 && foundCount < totalCount {
				for _, param := range enabledProviderAuth[provider] {
					val, exists := GetEnvOrFile(param)
					if !exists || strings.Trim(val, " ") == "" {
						_, _ = fmt.Fprintf(&outputErrs, "%s parameter '%s' is not defined.\n", provider, param)
					}
				}
			}

			if foundCount == totalCount {
				numUserDefinedProviders++
			}
		}

		if outputErrs.Len() > 0 {
			logger.Fatalln(outputErrs.String())
		}
	}

	return retFunc
}

func Run() error {
	gitExecPath := gitInstallPath()
	if gitExecPath == "" {
		return errors.New("git not found in PATH")
	}

	displayStartupConfig()

	logger.Println("using git executable:", gitExecPath)

	ok, reqTimeout, err := getRequestTimeout()
	if err != nil {
		return err
	}

	if ok {
		logger.Printf("using defined request timeout: %s", reqTimeout.String())
	} else {
		logger.Printf("using default request timeout: %s", reqTimeout.String())
	}

	backupDIR, backupDIRKeyExists := os.LookupEnv(envGitBackupDir)
	if !backupDIRKeyExists || backupDIR == "" {
		return fmt.Errorf("environment variable %s must be set", envGitBackupDir)
	}

	_, ghOrgsExists := GetEnvOrFile(envGitHubOrgs)
	_, githubTokenExists := GetEnvOrFile(envGitHubToken)

	if ghOrgsExists {
		if !githubTokenExists {
			return fmt.Errorf("environment variable %s must be set if %s is set", envGitHubToken, envGitHubOrgs)
		}
	}

	backupDIR = strings.TrimSuffix(backupDIR, "\n")

	_, err = os.Stat(backupDIR)
	if os.IsNotExist(err) {
		return errors.Wrap(err, fmt.Sprintf("specified backup directory \"%s\" does not exist", backupDIR))
	}

	if err = checkProvidersDefined(); err != nil {
		logger.Fatal("no providers defined")
	}

	workingDIR := filepath.Join(backupDIR, workingDIRName)

	logger.Println("creating working directory:", workingDIR)
	createWorkingDIRErr := os.MkdirAll(workingDIR, workingDIRMode)

	if createWorkingDIRErr != nil {
		logger.Fatal(createWorkingDIRErr)
	}

	backupInterval := getBackupInterval()
	backupCron := os.Getenv(envGitBackupCron)

	var s gocron.Scheduler

	s, err = gocron.NewScheduler()
	if err != nil {
		return errors.Wrap(err, "failed to create scheduler")
	}

	switch {
	case backupInterval != 0:
		logger.Printf("scheduling to Run every %s", formatIntervalDuration(backupInterval))

		job, err = s.NewJob(
			gocron.DurationJob(
				time.Duration(backupInterval)*time.Minute,
			),
			gocron.NewTask(
				execProviderBackups,
			),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			gocron.WithStartAt(gocron.WithStartImmediately()),
		)
		if err != nil {
			return errors.Wrap(err, "failed to create job")
		}

		s.Start()

		select {}
	case backupCron != "":
		logger.Printf("scheduling to Run with cron '%s'", backupCron)

		job, err = s.NewJob(
			gocron.CronJob(
				backupCron,
				false,
			),
			gocron.NewTask(
				execProviderBackups,
			),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
		)
		if err != nil {
			return errors.Wrap(err, "failed to create job")
		}

		s.Start()

		select {}
	default:
		execProviderBackups()
	}

	return nil
}

type ProviderBackupResults struct {
	Provider string                        `json:"provider"`
	Results  githosts.ProviderBackupResult `json:"results"`
}

func getHTTPClient(logLevel string) *retryablehttp.Client {
	tr := &http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: true,
		MaxIdleConns:       maxIdleConns,
		IdleConnTimeout:    idleConnTimeout,
		ForceAttemptHTTP2:  false,
	}

	rc := retryablehttp.NewClient()

	_, reqTimeout, _ := getRequestTimeout()

	rc.HTTPClient = &http.Client{
		Transport: tr,
		Timeout:   reqTimeout,
	}

	if !strings.EqualFold(logLevel, "debug") {
		rc.Logger = nil
	}

	rc.RetryWaitMax = httpRetryWaitMax
	rc.RetryWaitMin = httpRetryWaitMin
	rc.RetryMax = httpRetryMax

	return rc
}

func getProjectMinimumAccessLevel() int {
	return getEnvIntDefault(envGitLabMinAccessLevel, defaultGitLabMinimumProjectAccessLevel)
}

func getBackupsToRetain(envVar string) int {
	return getEnvIntDefault(envVar, defaultBackupsToRetain)
}

func isInt(i string) (int, bool) {
	if val, err := strconv.Atoi(i); err == nil {
		return val, true
	}

	return 0, false
}

// getEnvIntDefault returns an integer value from the specified environment
// variable, or the provided default if the variable is unset or invalid.
func getEnvIntDefault(envVar string, def int) int {
	val := os.Getenv(envVar)
	if val == "" {
		logger.Printf("environment variable %s not set, using default of %d", envVar, def)

		return def
	}

	i, err := strconv.Atoi(val)
	if err != nil {
		logger.Printf("error converting environment variable %s to int so defaulting to: %d", envVar, def)

		return def
	}

	return i
}

var lookPath = exec.LookPath

func gitInstallPath() string {
	p, err := lookPath("git")
	if err != nil {
		logger.Printf("git not found: %v", err)
		return ""
	}

	return p
}

func init() {
	logger = log.New(os.Stdout, fmt.Sprintf("%s: ", AppName), log.Lshortfile|log.LstdFlags)
}

func getLogLevel() int {
	sobaLogLevelEnv := os.Getenv(envSobaLogLevel)

	if sobaLogLevelEnv != "" {
		sobaLogLevel, err := strconv.Atoi(sobaLogLevelEnv)
		if err != nil {
			logger.Fatalf("%s must be a number.", envSobaLogLevel)
		}

		return sobaLogLevel
	}

	return 0
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

func envTrue(envVar string) bool {
	val := os.Getenv(envVar)
	if val == "" {
		return false
	}

	if strings.EqualFold(val, "yes") || strings.EqualFold(val, "y") {
		return true
	}

	res, err := strconv.ParseBool(os.Getenv(envVar))
	if err != nil {
		return false
	}

	return res
}

var job gocron.Job

func formatIntervalDuration(m int) string {
	if m == 0 {
		return ""
	}

	if m%minutesPerHour == 0 {
		return fmt.Sprintf("%dh", m/minutesPerHour)
	}

	return time.Duration(int64(m) * int64(time.Minute)).String()
}

func getRequestTimeout() (bool, time.Duration, error) {
	eReqTimeout := os.Getenv(envGitRequestTimeout)

	if eReqTimeout != "" {
		reqTimeoutInt, err := strconv.Atoi(eReqTimeout)
		if err != nil {
			return false, defaultHTTPClientRequestTimeout, fmt.Errorf("%s value \"%s\" should be the maximum seconds to wait for a response, defined as an integer", envGitRequestTimeout, eReqTimeout)
		}

		return true, time.Duration(reqTimeoutInt) * time.Second, nil
	}

	return false, defaultHTTPClientRequestTimeout, nil
}

func getOrgsListFromEnvVar(envVar string) []string {
	orgsList := os.Getenv(envVar)

	if orgsList == "" {
		return []string{}
	}

	return strings.Split(orgsList, ",")
}
