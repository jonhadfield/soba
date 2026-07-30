package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/jonhadfield/githosts-utils/v2"
	"gitlab.com/tozd/go/errors"
)

type BackupResults struct {
	StartedAt  sobaTime                 `json:"started_at"`
	FinishedAt sobaTime                 `json:"finished_at"`
	Results    *[]ProviderBackupResults `json:"results,omitempty"`
}

func execProviderBackups() {
	failed := runProviderBackups()

	// If running one-shot (no scheduler) and any provider failed, surface a
	// non-zero exit. With a scheduler the next-run banner is enough.
	if job == nil && failed > 0 {
		os.Exit(1)
	}
}

func runProviderBackups() int {
	backupDir, exists := GetEnvOrFile(envGitBackupDir)
	if !exists || backupDir == "" {
		logger.Printf("environment variable %s is not set; skipping backup run", envGitBackupDir)

		return 0
	}

	if httpClient == nil {
		httpClient = getHTTPClient(os.Getenv(envSobaLogLevel))
	}

	workingDir := resolveWorkingDir(backupDir)

	// Cleanup runs on normal completion via this defer. Signal-driven
	// shutdown calls gocron's Shutdown() which waits for the current job to
	// finish, so this defer still fires before process exit on SIGINT/SIGTERM.
	defer cleanupWorkingDir(workingDir, backupDir)

	backupResults := BackupResults{
		StartedAt: sobaTime{
			Time: time.Now(),
			f:    time.RFC3339,
		},
	}

	providerBackupResults := collectProviderBackupResults(backupDir)

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
	}

	return failed
}

// collectProviderBackupResults runs a backup for each provider with complete
// credentials and returns the per-provider results.
func collectProviderBackupResults(backupDir string) []ProviderBackupResults {
	var results []ProviderBackupResults

	// BitBucket - check for API OAuthToken or OAuth2 authentication
	if bitbucketAPITokenDefined() || bitbucketOAuthDefined() {
		results = append(results, *Bitbucket(backupDir))
	}

	tokenProviders := []struct {
		envVar string
		run    func(string) *ProviderBackupResults
	}{
		{envGiteaToken, Gitea},
		{envGitHubToken, GitHub},
		{envGitLabToken, Gitlab},
		{envAzureDevOpsUserName, AzureDevOps},
		{envSourcehutToken, Sourcehut},
	}

	for _, p := range tokenProviders {
		if val, ok := GetEnvOrFile(p.envVar); ok && val != "" {
			results = append(results, *p.run(backupDir))
		}
	}

	return results
}

func bitbucketAPITokenDefined() bool {
	bbEmail, emailExists := GetEnvOrFile(envBitBucketEmail)
	bbToken, tokenExists := GetEnvOrFile(envBitBucketAPIToken)

	return emailExists && bbEmail != "" && tokenExists && bbToken != ""
}

func bitbucketOAuthDefined() bool {
	bbUser, userExists := GetEnvOrFile(envBitBucketUser)
	bbKey, keyExists := GetEnvOrFile(envBitBucketKey)
	bbSecret, secretExists := GetEnvOrFile(envBitBucketSecret)

	return userExists && bbUser != "" && keyExists && bbKey != "" && secretExists && bbSecret != ""
}

func resolveWorkingDir(backupDir string) string {
	if w := os.Getenv(envGitWorkingDir); w != "" {
		return w
	}

	return filepath.Join(backupDir, workingDIRName)
}

// cleanupWorkingDir removes the working directory after a backup run.
// It refuses to remove anything that does not resolve to a path inside
// backupDir to protect against a misconfigured GIT_WORKING_DIR wiping
// arbitrary filesystem locations.
func cleanupWorkingDir(workingDir, backupDir string) {
	logger.Println("cleaning up")

	if workingDir == "" {
		return
	}

	absWorking, err := filepath.Abs(filepath.Clean(workingDir))
	if err != nil {
		logger.Printf("failed to resolve working directory %q: %v", workingDir, err)

		return
	}

	absBackup, err := filepath.Abs(filepath.Clean(backupDir))
	if err != nil {
		logger.Printf("failed to resolve backup directory %q: %v", backupDir, err)

		return
	}

	if absWorking == absBackup {
		logger.Printf("refusing to clean working directory %q: equals backup directory", absWorking)

		return
	}

	if !strings.HasPrefix(absWorking+string(os.PathSeparator), absBackup+string(os.PathSeparator)) {
		logger.Printf("refusing to clean working directory %q: not inside backup directory %q", absWorking, absBackup)

		return
	}

	if err := os.RemoveAll(absWorking); err != nil {
		logger.Printf("failed to clean working directory %q: %v", absWorking, err)
	}
}

func displayStartupConfig() {
	if backupDIR, exists := GetEnvOrFile(envGitBackupDir); exists && backupDIR != "" {
		logger.Printf("root backup directory: %s", backupDIR)
	}

	displayGitHubStartupConfig()
	displayGiteaStartupConfig()
	displayGitLabStartupConfig()
	displayBitBucketStartupConfig()
	displayAzureDevOpsStartupConfig()
}

// logProviderOrgs logs the configured organisations for a provider, if any.
func logProviderOrgs(label, envVar string) {
	if orgs, exists := GetEnvOrFile(envVar); exists && strings.ToLower(orgs) != "" {
		logger.Printf("%s Organistations: %s", label, strings.ToLower(orgs))
	}
}

// logProviderBackupsToKeep logs the configured backup retention for a provider, if set.
func logProviderBackupsToKeep(label, envVar string) {
	if backups, exists := GetEnvOrFile(envVar); exists && backups != "" {
		logger.Printf("%s backups to keep: %s", label, backups)
	}
}

// logProviderCompareMethod logs whether a provider compares by refs or clone.
func logProviderCompareMethod(label, envVar string) {
	method := compareTypeClone
	if compare, exists := GetEnvOrFile(envVar); exists && strings.EqualFold(compare, compareTypeRefs) {
		method = compareTypeRefs
	}

	logger.Printf("%s compare method: %s", label, method)
}

// logProviderBackupLFS logs whether LFS backup is enabled for a provider.
func logProviderBackupLFS(label, envVar string) {
	if _, exists := GetEnvOrFile(envVar); exists && envTrue(envVar) {
		logger.Printf("%s backup LFS: true", label)
	}
}

func displayGitHubStartupConfig() {
	if ghToken, exists := GetEnvOrFile(envGitHubToken); !exists || ghToken == "" {
		return
	}

	logProviderOrgs("GitHub", envGitHubOrgs)

	if _, exists := GetEnvOrFile(envGitHubSkipUserRepos); exists && envTrue(envGitHubSkipUserRepos) {
		logger.Printf("GitHub skipping user repos: true")
	}

	logProviderCompareMethod("GitHub", envGitHubCompare)
	logProviderBackupLFS("GitHub", envGitHubBackupLFS)
}

func displayGiteaStartupConfig() {
	if giteaToken, exists := GetEnvOrFile(envGiteaToken); !exists || giteaToken == "" {
		return
	}

	logProviderOrgs("Gitea", envGiteaOrgs)
	logProviderBackupsToKeep("Gitea", envGiteaBackups)
	logProviderCompareMethod("Gitea", envGiteaCompare)
	logProviderBackupLFS("Gitea", envGiteaBackupLFS)
}

func displayGitLabStartupConfig() {
	if glToken, exists := GetEnvOrFile(envGitLabToken); !exists || glToken == "" {
		return
	}

	glProjectMinAccessLevel, minAccessExists := GetEnvOrFile(envGitLabMinAccessLevel)
	if !minAccessExists || glProjectMinAccessLevel == "" {
		logger.Printf("GitLab project minimum access level: %d", githosts.GitLabDefaultMinimumProjectAccessLevel)
	} else {
		logger.Printf("GitLab project minimum access level: %s", glProjectMinAccessLevel)
	}

	logProviderBackupsToKeep("GitLab", envGitLabBackups)
	logProviderCompareMethod("GitLab", envGitLabCompare)
	logProviderBackupLFS("Gitlab", envGitLabBackupLFS)
}

func displayBitBucketStartupConfig() {
	if bbUser, exists := GetEnvOrFile(envBitBucketEmail); !exists || bbUser == "" {
		return
	}

	logProviderBackupsToKeep("BitBucket", envBitBucketBackups)
	logProviderCompareMethod("BitBucket", envBitBucketCompare)
	logProviderBackupLFS("BitBucket", envBitBucketBackupLFS)
}

func displayAzureDevOpsStartupConfig() {
	if azureDevOpsUserName, exists := GetEnvOrFile(envAzureDevOpsUserName); !exists || azureDevOpsUserName == "" {
		return
	}

	logProviderOrgs(providerLabelAzureDevOps, envAzureDevOpsOrgs)
	logProviderCompareMethod(providerLabelAzureDevOps, envAzureDevOpsCompare)
	logProviderBackupLFS(providerLabelAzureDevOps, envAzureDevOpsBackupLFS)
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

// checkProvider validates a single provider's credentials. It returns the
// number of fully-configured provider entries it found (0 or 1) and any
// partial-configuration errors so the caller can surface them rather than
// terminating the process from a library function.
func checkProvider(provider string) (int, error) {
	var outputErrs strings.Builder

	var count int

	if slices.Contains(justTokenProviders, provider) {
		count += checkJustTokenProvider(provider, &outputErrs)
	}

	if slices.Contains(userAndPasswordProviders, provider) {
		count += checkUserAndPasswordProvider(provider, &outputErrs)
	}

	if outputErrs.Len() > 0 {
		return count, errors.New(outputErrs.String())
	}

	return count, nil
}

// checkJustTokenProvider counts the provider's non-empty auth parameters,
// recording any that are defined but blank.
func checkJustTokenProvider(provider string, outputErrs *strings.Builder) int {
	var count int

	for _, param := range enabledProviderAuth[provider] {
		val, exists := GetEnvOrFile(param)
		if !exists {
			continue
		}

		if strings.Trim(val, " ") == "" {
			_, _ = fmt.Fprintf(outputErrs, "%s parameter '%s' is not defined.\n", provider, param)
		} else {
			count++
		}
	}

	return count
}

// checkUserAndPasswordProvider returns 1 if all of the provider's auth
// parameters are set, recording the missing ones when only some are.
func checkUserAndPasswordProvider(provider string, outputErrs *strings.Builder) int {
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
				_, _ = fmt.Fprintf(outputErrs, "%s parameter '%s' is not defined.\n", provider, param)
			}
		}
	}

	if foundCount == totalCount {
		return 1
	}

	return 0
}

func Run() error {
	gitExecPath := gitInstallPath()
	if gitExecPath == "" {
		return errors.New("git not found in PATH")
	}

	displayStartupConfig()

	logger.Println("using git executable:", gitExecPath)
	logGitVersion(gitExecPath)

	if err := logRequestTimeout(); err != nil {
		return err
	}

	backupDIR, err := validateStartupConfig()
	if err != nil {
		return err
	}

	if err = createWorkingDir(backupDIR); err != nil {
		return err
	}

	return scheduleBackups()
}

// logRequestTimeout logs the request timeout in use, returning an error if a
// defined timeout is invalid.
func logRequestTimeout() error {
	ok, reqTimeout, err := getRequestTimeout()
	if err != nil {
		return err
	}

	if ok {
		logger.Printf("using defined request timeout: %s", reqTimeout.String())
	} else {
		logger.Printf("using default request timeout: %s", reqTimeout.String())
	}

	return nil
}

// validateStartupConfig checks the environment holds a usable backup
// directory and provider configuration, returning the backup directory.
func validateStartupConfig() (string, error) {
	backupDIR, backupDIRKeyExists := os.LookupEnv(envGitBackupDir)
	if !backupDIRKeyExists || backupDIR == "" {
		return "", fmt.Errorf("environment variable %s must be set", envGitBackupDir)
	}

	_, ghOrgsExists := GetEnvOrFile(envGitHubOrgs)
	_, githubTokenExists := GetEnvOrFile(envGitHubToken)

	if ghOrgsExists && !githubTokenExists {
		return "", fmt.Errorf("environment variable %s must be set if %s is set", envGitHubToken, envGitHubOrgs)
	}

	backupDIR = strings.TrimSuffix(backupDIR, "\n")

	if _, statErr := os.Stat(backupDIR); statErr != nil {
		if os.IsNotExist(statErr) {
			return "", errors.Wrap(statErr, fmt.Sprintf("specified backup directory \"%s\" does not exist", backupDIR))
		}

		return "", errors.Wrap(statErr, fmt.Sprintf("cannot access backup directory \"%s\"", backupDIR))
	}

	if err := checkProvidersDefined(); err != nil {
		return "", errors.Wrap(err, "provider configuration invalid")
	}

	return backupDIR, nil
}

func createWorkingDir(backupDIR string) error {
	// Check if GIT_WORKING_DIR is set, otherwise use default
	workingDIR := os.Getenv(envGitWorkingDir)
	if workingDIR == "" {
		workingDIR = filepath.Join(backupDIR, workingDIRName)
	}

	logger.Println("creating working directory:", workingDIR)

	if mkErr := os.MkdirAll(filepath.Clean(workingDIR), workingDIRMode); mkErr != nil {
		return errors.Wrap(mkErr, fmt.Sprintf("failed to create working directory %q", workingDIR))
	}

	return nil
}

// scheduleBackups runs backups on the configured interval or cron schedule,
// blocking until shutdown; with neither configured it runs a single backup.
func scheduleBackups() error {
	backupInterval := getBackupInterval()
	backupCron := os.Getenv(envGitBackupCron)

	s, err := gocron.NewScheduler()
	if err != nil {
		return errors.Wrap(err, "failed to create scheduler")
	}

	switch {
	case backupInterval != 0:
		logger.Printf("scheduling to Run every %s", formatIntervalDuration(backupInterval))

		return runScheduledJob(s,
			gocron.DurationJob(time.Duration(backupInterval)*time.Minute),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			gocron.WithStartAt(gocron.WithStartImmediately()),
		)
	case backupCron != "":
		logger.Printf("scheduling to Run with cron '%s'", backupCron)

		return runScheduledJob(s,
			gocron.CronJob(backupCron, false),
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
		)
	default:
		execProviderBackups()
	}

	return nil
}

// runScheduledJob registers the backup task with the scheduler, starts it and
// blocks until shutdown.
func runScheduledJob(s gocron.Scheduler, definition gocron.JobDefinition, options ...gocron.JobOption) error {
	var err error

	job, err = s.NewJob(
		definition,
		gocron.NewTask(execProviderBackups),
		options...,
	)
	if err != nil {
		return errors.Wrap(err, "failed to create job")
	}

	s.Start()
	waitForShutdown(s)

	return nil
}

// waitForShutdown blocks until SIGINT or SIGTERM is received and then
// shuts the scheduler down gracefully so any running job finishes (and
// its deferred cleanup runs) before the process exits.
func waitForShutdown(s gocron.Scheduler) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Printf("received %s; shutting down scheduler", sig)

	if err := s.Shutdown(); err != nil {
		logger.Printf("scheduler shutdown error: %v", err)
	}
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

// logGitVersion records the resolved git version at startup. Failure is
// non-fatal — gitInstallPath already verified the binary exists, but a
// version probe can fail in restricted environments (sandboxes, PATH
// shimming, etc.) and should not abort the run.
func logGitVersion(gitExecPath string) {
	ctx, cancel := context.WithTimeout(context.Background(), gitVersionProbeTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, gitExecPath, "--version").Output()
	if err != nil {
		logger.Printf("could not determine git version: %v", err)

		return
	}

	logger.Printf("git version: %s", strings.TrimSpace(string(out)))
}

func init() {
	logger = log.New(os.Stdout, fmt.Sprintf("%s: ", AppName), log.Lshortfile|log.LstdFlags)
}

func getLogLevel() int {
	sobaLogLevelEnv := os.Getenv(envSobaLogLevel)

	if sobaLogLevelEnv == "" {
		return 0
	}

	sobaLogLevel, err := strconv.Atoi(sobaLogLevelEnv)
	if err != nil {
		logger.Printf("%s must be a number; defaulting to 0", envSobaLogLevel)

		return 0
	}

	return sobaLogLevel
}

func checkProvidersDefined() error {
	var count int

	var errBuilder strings.Builder

	bitbucketAPITokenComplete := bitbucketAPITokenDefined()

	for provider := range enabledProviderAuth {
		switch provider {
		case providerNameBitBucketAPIToken:
			if bitbucketAPITokenComplete {
				count++
			}
		case providerNameBitBucketOAuth:
			// Only count if the API OAuthToken method isn't already complete.
			if bitbucketOAuthDefined() && !bitbucketAPITokenComplete {
				count++
			}
		default:
			n, err := checkProvider(provider)
			count += n

			if err != nil {
				errBuilder.WriteString(err.Error())
			}
		}
	}

	if errBuilder.Len() > 0 {
		return errors.New(errBuilder.String())
	}

	if count == 0 {
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
