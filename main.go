package main

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
	"github.com/pkg/errors"
)

const (
	appName                                = "soba"
	workingDIRName                         = ".working"
	workingDIRMode                         = 0o755
	defaultBackupsToRetain                 = 2
	defaultGitLabMinimumProjectAccessLevel = 20
	defaultEarlyErrorBackOffSeconds        = 5

	defaultHTTPClientRequestTimeout = 300 * time.Second

	// general constants
	pathSep        = string(os.PathSeparator)
	minutesPerHour = 60

	// retry settings
	httpRetryWaitMax    = 120 * time.Second
	httpRetryWaitMin    = 60 * time.Second
	httpRetryMax        = 2
	webhookRetryWaitMin = 1 * time.Second
	webhookRetryWaitMax = 3 * time.Second
	webhookRetryMax     = 3

	// http
	maxIdleConns    = 10
	idleConnTimeout = 30 * time.Second

	// env vars
	envPath                = "PATH"
	envSobaLogLevel        = "SOBA_LOG"
	envSobaWebHookURL      = "SOBA_WEBHOOK_URL"
	envSobaWebHookFormat   = "SOBA_WEBHOOK_FORMAT"
	envGitBackupInterval   = "GIT_BACKUP_INTERVAL"
	envGitBackupCron       = "GIT_BACKUP_CRON"
	envGitBackupDir        = "GIT_BACKUP_DIR"
	envGitRequestTimeout   = "GIT_REQUEST_TIMEOUT"
	envGitHubAPIURL        = "GITHUB_APIURL"
	envGitHubBackups       = "GITHUB_BACKUPS"
	envAzureDevOpsOrgs     = "AZURE_DEVOPS_ORGS"
	envAzureDevOpsUserName = "AZURE_DEVOPS_USERNAME"
	envAzureDevOpsPAT      = "AZURE_DEVOPS_PAT"
	envAzureDevOpsCompare  = "AZURE_DEVOPS_COMPARE"
	envAzureDevOpsBackups  = "AZURE_DEVOPS_BACKUPS"
	// nolint:gosec
	envGitHubToken          = "GITHUB_TOKEN"
	envGitHubOrgs           = "GITHUB_ORGS"
	envGitHubSkipUserRepos  = "GITHUB_SKIP_USER_REPOS"
	envGitHubLimitUserOwned = "GITHUB_LIMIT_USER_OWNED"
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
	providerNameAzureDevOps = "AzureDevOps"
	providerNameBitBucket   = "BitBucket"
	providerNameGitHub      = "GitHub"
	providerNameGitLab      = "GitLab"
	providerNameGitea       = "Gitea"

	// compare types
	compareTypeRefs  = "refs"
	compareTypeClone = "clone"
)

var (
	logger *log.Logger
	// overwritten at build time.
	version, tag, sha, buildDate string

	httpClient *retryablehttp.Client

	enabledProviderAuth = map[string][]string{
		providerNameAzureDevOps: {
			envAzureDevOpsUserName,
			envAzureDevOpsPAT,
		},
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
		providerNameAzureDevOps,
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

	// output azure devops config
	if azureDevOpsUserName := os.Getenv(envAzureDevOpsUserName); azureDevOpsUserName != "" {
		if ghOrgs := strings.ToLower(os.Getenv(envAzureDevOpsOrgs)); ghOrgs != "" {
			logger.Printf("Azure DevOps Organistations: %s", ghOrgs)
		}

		if strings.EqualFold(os.Getenv(envAzureDevOpsCompare), compareTypeRefs) {
			logger.Print("Azure DevOps compare method: refs")
		} else {
			logger.Print("Azure DevOps compare method: clone")
		}
	}
}

var job gocron.Job

func run() error {
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

	if _, githubOrgsKeyExists := os.LookupEnv(envGitHubOrgs); githubOrgsKeyExists {
		if _, githubTokenExists := os.LookupEnv(envGitHubToken); !githubTokenExists {
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
		logger.Printf("scheduling to run every %s", formatIntervalDuration(backupInterval))

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
		logger.Printf("scheduling to run with cron '%s'", backupCron)

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

type ProviderBackupResults struct {
	Provider string                        `json:"provider"`
	Results  githosts.ProviderBackupResult `json:"results"`
}

type BackupResults struct {
	StartedAt  sobaTime                 `json:"started_at"`
	FinishedAt sobaTime                 `json:"finished_at"`
	Results    *[]ProviderBackupResults `json:"results,omitempty"`
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

func execProviderBackups() {
	backupDir := os.Getenv(envGitBackupDir)

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

	if os.Getenv(envBitBucketUser) != "" {
		providerBackupResults = append(providerBackupResults, *Bitbucket(backupDir))
	}

	if os.Getenv(envGiteaToken) != "" {
		providerBackupResults = append(providerBackupResults, *Gitea(backupDir))
	}

	if os.Getenv(envGitHubToken) != "" {
		providerBackupResults = append(providerBackupResults, *GitHub(backupDir))
	}

	if os.Getenv(envGitLabToken) != "" {
		providerBackupResults = append(providerBackupResults, *Gitlab(backupDir))
	}

	if os.Getenv(envAzureDevOpsUserName) != "" {
		providerBackupResults = append(providerBackupResults, *AzureDevOps(backupDir))
	}

	logger.Println("cleaning up")

	// startFileRemovals := time.Now()
	delErr := os.RemoveAll(backupDir + pathSep + workingDIRName + pathSep)
	if delErr != nil {
		logger.Printf("failed to delete working directory: %s",
			backupDir+pathSep+workingDIRName)
	}

	// logger.Printf("file removals took %s", time.Since(startFileRemovals).String())

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
		logger.Printf("next run scheduled for: %s", nextRun.Format("2006-01-02 15:04:05 -0700 MST"))
	} else if failed > 0 { // if no interval is set then exit
		os.Exit(1)
	}
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
	path, _ := lookPath("git")

	return path
}
