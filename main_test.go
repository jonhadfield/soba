package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jonhadfield/githosts-utils"

	"github.com/stretchr/testify/require"
)

var sobaEnvVarKeys = []string{
	envPath, envGitBackupDir, envGitHubToken, envGitHubBackups, envGitLabToken, envGitLabBackups, envGitLabAPIURL,
	envGitHubCompare, envGitLabCompare, envBitBucketCompare,
	envBitBucketUser, envBitBucketKey, envBitBucketSecret, envBitBucketBackups,
	envGiteaAPIURL, envGiteaToken, envGiteaOrgs, envGiteaCompare, envGiteaBackups,
	envAzureDevOpsUserName, envAzureDevOpsPAT, envAzureDevOpsOrgs, envAzureDevOpsCompare, envAzureDevOpsBackups,
}

const (
	goSobaOrg                  = "go-soba"
	sobaOrgOne                 = "soba-org-one"
	sobaOrgTwo                 = "soba-org-two"
	skipGitHubTestMissingToken = "Skipping GitHub test as %s is missing" //nolint:gosec
)

func TestGetBackupInterval(t *testing.T) {
	os.Setenv(envGitBackupInterval, "1h")
	require.Equal(t, 60, getBackupInterval())

	os.Setenv(envGitBackupInterval, "1")
	require.Equal(t, 60, getBackupInterval())

	os.Setenv(envGitBackupInterval, "100h")
	require.Equal(t, 6000, getBackupInterval())

	os.Setenv(envGitBackupInterval, "100m")
	require.Equal(t, 100, getBackupInterval())

	os.Setenv(envGitBackupInterval, "0")
	require.Equal(t, 0, getBackupInterval())
}

func TestGetProjectMinimumAccessLevel(t *testing.T) {
	os.Setenv(envGitLabMinAccessLevel, "30")
	require.Equal(t, 30, getProjectMinimumAccessLevel())

	os.Setenv(envGitLabMinAccessLevel, "invalid")
	require.Equal(t, defaultGitLabMinimumProjectAccessLevel, getProjectMinimumAccessLevel())

	os.Unsetenv(envGitLabMinAccessLevel)
	require.Equal(t, defaultGitLabMinimumProjectAccessLevel, getProjectMinimumAccessLevel())
}

func TestGetBackupsToRetain(t *testing.T) {
	os.Setenv(envGitHubBackups, "5")
	require.Equal(t, 5, getBackupsToRetain(envGitHubBackups))

	os.Setenv(envGitHubBackups, "invalid")
	require.Equal(t, defaultBackupsToRetain, getBackupsToRetain(envGitHubBackups))

	os.Unsetenv(envGitHubBackups)
	require.Equal(t, defaultBackupsToRetain, getBackupsToRetain(envGitHubBackups))
}

func TestIsInt(t *testing.T) {
	val, ok := isInt("123")
	require.True(t, ok)
	require.Equal(t, 123, val)

	val, ok = isInt("invalid")
	require.False(t, ok)
	require.Equal(t, 0, val)
}

func TestGitInstalled(t *testing.T) {
	// succeed
	gitPath := gitInstallPath()
	require.NotEmpty(t, gitPath)

	// mock exec.LookPath function to return an error
	lookPath = func(file string) (string, error) { //nolint:revive
		return "", errors.New("command not found")
	}
	defer func() { lookPath = exec.LookPath }()

	gitPath = gitInstallPath()
	require.Empty(t, gitPath)
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("removeContents: %w", err)
	}

	defer func() {
		if err = d.Close(); err != nil {
			panic(err)
		}
	}()

	names, err := d.Readdirnames(-1)
	if err != nil {
		return fmt.Errorf("removeContents %w", err)
	}

	for _, name := range names {
		if err = os.RemoveAll(filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("removeContents %w", err)
		}
	}

	return nil
}

func preflight() {
	var err error

	// prepare clean backup directory
	bud := os.Getenv(envGitBackupDir)

	// if path not provided, create one
	if bud == "" {
		bud, err = os.MkdirTemp(os.TempDir(), "sobabackup-*")
		if err != nil {
			panic(err)
		}

		_ = os.Setenv(envGitBackupDir, bud)

		return
	}

	_, err = os.Stat(bud)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(bud, 0o700); err != nil {
			panic(err)
		}

		return
	}

	// clean out existing backup directory
	if err = removeContents(bud); err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	preflight()

	code := m.Run()

	os.Exit(code)
}

func resetGlobals() {
	// reset global var
	numUserDefinedProviders = 0
}

func backupEnvironmentVariables() map[string]string {
	m := make(map[string]string)

	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			m[e[:i]] = e[i+1:]
		}
	}

	return m
}

func restoreEnvironmentVariables(input map[string]string) {
	for key, val := range input {
		_ = os.Setenv(key, val)
	}
}

func unsetEnvVarsExcept(exceptionList []string) {
	for _, sobaVar := range sobaEnvVarKeys {
		if !slices.Contains(exceptionList, sobaVar) {
			_ = os.Unsetenv(sobaVar)
		}
	}
}

func resetBackups() {
	backupDir := os.Getenv(envGitBackupDir)
	if backupDir == "" {
		return
	}

	if err := removeContents(backupDir); err != nil {
		panic(err)
	}
}

func TestGitHubEnvs(t *testing.T) {
	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)
	require.NoError(t, os.Unsetenv(envGitHubToken))
	require.NoError(t, os.Unsetenv(envGitLabToken))
	require.NoError(t, os.Setenv(envGitHubOrgs, "example,example2"))

	err := run()

	require.NoError(t, os.Unsetenv(envGitHubOrgs))

	require.Error(t, err)
	require.Contains(t, err.Error(), fmt.Sprintf("%s must be set if %s is set", envGitHubToken, envGitHubOrgs))
}

func TestInvalidBundleIsMovedWithRefCompare(t *testing.T) {
	// set comparison to use refs, rather than bundle
	require.NoError(t, os.Setenv(envGitHubCompare, compareTypeRefs))

	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	// create invalid bundle
	backupDir := os.Getenv(envGitBackupDir)
	dfDir := path.Join(backupDir, "github.com", goSobaOrg, "repo0")
	require.NoError(t, os.MkdirAll(dfDir, 0o755))

	dfName := "repo0.20200401111111.bundle"
	dfPath := path.Join(dfDir, dfName)

	_, err := os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
	require.NoError(t, err)
	require.NoError(t, os.Setenv(envGitHubBackups, "1"))
	// run
	require.NoError(t, run())
	// check only one bundle remains
	files, err := os.ReadDir(dfDir)
	require.NoError(t, err)

	dfRenamed := "repo0.20200401111111.bundle.invalid"

	var originalFound int

	var renamedFound int

	for _, f := range files {
		require.NotEqual(t, f.Name(), dfName, fmt.Sprintf("unexpected bundle: %s", f.Name()))

		if dfName == f.Name() {
			originalFound++
		}

		if dfRenamed == f.Name() {
			renamedFound++
		}
	}

	require.Zero(t, originalFound)

	require.Equal(t, 1, renamedFound)
}

func TestAzureDevOpsRepositoryBackupWithBackupsToKeepAsOne(t *testing.T) {
	if os.Getenv(envAzureDevOpsUserName) == "" {
		t.Skipf("Skipping Azure DevOps test as %s is missing", envAzureDevOpsUserName)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{
		envPath,
		envGitBackupDir,
		envAzureDevOpsUserName,
		envAzureDevOpsPAT,
		envAzureDevOpsOrgs,
		envAzureDevOpsBackups,
		envAzureDevOpsCompare,
	})

	// run
	require.NoError(t, run())

	require.NoError(t, run())
}

func TestGetRequestTimeout(t *testing.T) {
	t.Setenv(envGitRequestTimeout, "600")

	ok, timeout, err := getRequestTimeout()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 600*time.Second, timeout)

	t.Setenv(envGitRequestTimeout, "invalid")

	ok, _, err = getRequestTimeout()
	require.False(t, ok)
	require.Error(t, err)

	t.Setenv(envGitRequestTimeout, "")

	ok, timeout, err = getRequestTimeout()
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, defaultHTTPClientRequestTimeout, timeout)
}

func TestPublicGithubRepositoryBackupWithBackupsToKeepAsOne(t *testing.T) {
	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	// create dummy bundle
	backupDir := os.Getenv(envGitBackupDir)
	dfDir := path.Join(backupDir, "github.com", goSobaOrg, "repo0")
	require.NoError(t, os.MkdirAll(dfDir, 0o755))
	require.NoError(t, os.Setenv(envGitHubBackups, "1"))
	// run
	require.NoError(t, run())
	// check only one bundle exists
	files, err := os.ReadDir(dfDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	firstBackupFileName := files[0].Name()
	// run for a second time
	require.NoError(t, run())
	// check only one bundle exists
	files, err = os.ReadDir(dfDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, firstBackupFileName, files[0].Name())
}

func TestPublicGithubRepositoryBackupWithBackupsToKeepUnset(t *testing.T) {
	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()

	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	// create dummy bundle
	backupDir := os.Getenv(envGitBackupDir)
	dfDir := path.Join(backupDir, "github.com", goSobaOrg, "repo0")
	require.NoError(t, os.MkdirAll(dfDir, 0o755))

	dfName := "repo0.20200401111111.bundle"
	dfPath := path.Join(dfDir, dfName)

	_, err := os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
	require.NoError(t, err)
	// run
	require.NoError(t, run())
	// check both bundles remain
	files, err := os.ReadDir(dfDir)
	require.NoError(t, err)
	require.Len(t, files, 2)
}

func TestGithubRepositoryBackupWithInvalidToken(t *testing.T) {
	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})

	// set invalid token
	_ = os.Setenv(envGitHubToken, "invalid")

	defer os.Unsetenv(envGitHubToken)

	githubHost, err := githosts.NewGitHubHost(githosts.NewGitHubHostInput{
		Caller:           appName,
		APIURL:           os.Getenv(envGitHubAPIURL),
		DiffRemoteMethod: os.Getenv(envGitHubCompare),
		BackupDir:        os.TempDir(),
		Token:            os.Getenv(envGitHubToken),
		Orgs:             getOrgsListFromEnvVar(envGitHubOrgs),
		BackupsToRetain:  getBackupsToRetain(envGitHubBackups),
		SkipUserRepos:    envTrue(envGitHubSkipUserRepos),
		LogLevel:         getLogLevel(),
	})
	require.NoError(t, err)

	result := githubHost.Backup()
	require.NotNil(t, result.Error)
	require.Contains(t, errors.Unwrap(result.Error).Error(), "Bad credentials")
}

func TestPublicGithubRepositoryBackup(t *testing.T) {
	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	require.NoError(t, run())
}

func TestPublicGithubRepositoryBackupWithExistingBackups(t *testing.T) {
	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	require.NoError(t, run())
	// run for second time now we have existing bundles
	require.NoError(t, run())
}

func TestPublicGithubRepositoryBackupWithExistingBackupsUsingRefs(t *testing.T) {
	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	require.NoError(t, run())
	// run for second time now we have existing bundles
	require.NoError(t, run())
}

func TestPublicGitLabRepositoryBackup(t *testing.T) {
	if os.Getenv(envGitLabToken) == "" {
		t.Skipf("Skipping GitLab test as %s is missing", envGitLabToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitLabToken})
	require.NoError(t, run())
}

func TestPublicGitLabRepositoryBackup2(t *testing.T) {
	if os.Getenv(envGitLabToken) == "" {
		t.Skipf("Skipping GitLab test as %s is missing", envGitLabToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitLabToken})
	require.NoError(t, run())
}

func TestGiteaRepositoryBackup(t *testing.T) {
	if os.Getenv(envGiteaToken) == "" {
		t.Skipf("Skipping Gitea test as %s is missing", envGiteaToken)
	}

	if os.Getenv(envGiteaAPIURL) == "" {
		t.Skipf("Skipping Gitea test as %s is missing", envGiteaAPIURL)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGiteaToken, envGiteaAPIURL})
	require.NoError(t, run())
}

func TestFormatIntervalDuration(t *testing.T) {
	require.Equal(t, "", formatIntervalDuration(0))
	require.Equal(t, "1h", formatIntervalDuration(60))
	require.Equal(t, "1h1m0s", formatIntervalDuration(61))
	require.Equal(t, "3m0s", formatIntervalDuration(3))
}

func TestGiteaOrgsRepositoryBackup(t *testing.T) {
	if os.Getenv(envGiteaToken) == "" {
		t.Skipf("Skipping Gitea test as %s is missing", envGiteaToken)
	}

	if os.Getenv(envGiteaAPIURL) == "" {
		t.Skipf("Skipping Gitea test as %s is missing", envGiteaAPIURL)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGiteaToken, envGiteaAPIURL})

	for _, org := range []string{sobaOrgTwo, "*"} {
		require.NoError(t, os.Setenv(envGiteaOrgs, org))

		require.NoError(t, run())

		switch org {
		case sobaOrgTwo:
			require.DirExists(t, path.Join(os.Getenv(envGitBackupDir), "gitea.lessknown.co.uk", sobaOrgTwo))
			require.NoDirExists(t, path.Join(os.Getenv(envGitBackupDir), "gitea.lessknown.co.uk", sobaOrgOne))
			entriesOrgTwo, err := os.ReadDir(path.Join(os.Getenv(envGitBackupDir), "gitea.lessknown.co.uk", sobaOrgTwo))
			require.NoError(t, err)

			require.Len(t, entriesOrgTwo, 2)

			var foundOne, foundTwo bool

			for _, entry := range entriesOrgTwo {
				if strings.HasPrefix(entry.Name(), "soba-org-two-repo-one") {
					foundOne = true
				}

				if strings.HasPrefix(entry.Name(), "soba-org-two-repo-two") {
					foundTwo = true
				}
			}

			require.True(t, foundOne)
			require.True(t, foundTwo)

			resetBackups()
		case "*":
			require.DirExists(t, path.Join(os.Getenv(envGitBackupDir), "gitea.lessknown.co.uk", sobaOrgTwo))
			require.DirExists(t, path.Join(os.Getenv(envGitBackupDir), "gitea.lessknown.co.uk", sobaOrgOne))
			entriesOrgOne, err := os.ReadDir(path.Join(os.Getenv(envGitBackupDir), "gitea.lessknown.co.uk", sobaOrgOne))
			require.NoError(t, err)
			entriesOrgTwo, err := os.ReadDir(path.Join(os.Getenv(envGitBackupDir), "gitea.lessknown.co.uk", sobaOrgTwo))
			require.NoError(t, err)

			require.Len(t, entriesOrgOne, 1)
			require.Len(t, entriesOrgTwo, 2)

			var foundOne, foundTwo, foundThree bool

			for _, entry := range entriesOrgOne {
				if strings.HasPrefix(entry.Name(), "soba-org-one-repo-one") {
					foundThree = true
				}
			}

			for _, entry := range entriesOrgTwo {
				if strings.HasPrefix(entry.Name(), "soba-org-two-repo-one") {
					foundOne = true
				}

				if strings.HasPrefix(entry.Name(), "soba-org-two-repo-two") {
					foundTwo = true
				}
			}

			require.True(t, foundOne)
			require.True(t, foundTwo)
			require.True(t, foundThree)

			resetBackups()
		}

		resetBackups()
	}
}

func TestPublicBitBucketRepositoryBackupWithRefCompare(t *testing.T) {
	if os.Getenv(envBitBucketUser) == "" {
		t.Skipf("Skipping BitBucket test as %s is missing", envBitBucketUser)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	resetGlobals()

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envBitBucketUser, envBitBucketKey, envBitBucketSecret})

	defer func() {
		if err := os.Unsetenv(envBitBucketCompare); err != nil {
			panic(fmt.Sprintf("failed to unset envvar: %s", err.Error()))
		}
	}()

	_ = os.Setenv(envBitBucketCompare, compareTypeRefs)

	defer os.Unsetenv(envBitBucketCompare)

	require.NoError(t, run())

	require.NoError(t, run())
}

func TestPublicBitBucketInvalidCredentials(t *testing.T) {
	if os.Getenv(envBitBucketUser) == "" {
		t.Skipf("Skipping BitBucket test as %s is missing", envBitBucketUser)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	resetGlobals()

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envBitBucketUser, envBitBucketKey, envBitBucketSecret})

	defer func() {
		if err := os.Unsetenv(envBitBucketCompare); err != nil {
			panic(fmt.Sprintf("failed to unset envvar: %s", err.Error()))
		}
	}()

	_ = os.Setenv(envBitBucketCompare, compareTypeRefs)

	// set invalid key
	_ = os.Setenv(envBitBucketKey, "invalid")

	if os.Getenv("BE_CRASHER") == "1" {
		_ = run()

		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestPublicBitBucketInvalidCredentials") // nolint:gosec
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")

	err := cmd.Run()

	var e *exec.ExitError

	if errors.As(err, &e) && !e.Success() {
		return
	}

	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func TestCheckProvidersFailureWhenNoneDefined(t *testing.T) {
	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	unsetEnvVarsExcept([]string{envPath})

	err := checkProvidersDefined()
	require.Error(t, err)

	require.Contains(t, err.Error(), "no providers defined")
}

func TestFailureIfGitBackupDirUndefined(t *testing.T) {
	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	unsetEnvVarsExcept([]string{envPath})

	_ = os.Setenv(envGitHubToken, "ABCD1234")

	defer os.Unsetenv(envGitHubToken)

	require.Errorf(t, run(), "expected: %s undefined error", envGitBackupDir)
}

func TestGithubRepositoryBackupWithSingleOrgNoPersonal(t *testing.T) {
	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	// create dummy bundle
	backupDir := os.Getenv(envGitBackupDir)

	githubHost, err := githosts.NewGitHubHost(githosts.NewGitHubHostInput{
		Caller:           appName,
		LogLevel:         1,
		APIURL:           os.Getenv(envGitHubAPIURL),
		DiffRemoteMethod: os.Getenv(envGitHubCompare),
		BackupDir:        backupDir,
		Token:            os.Getenv(envGitHubToken),
		SkipUserRepos:    true,
		BackupsToRetain:  1,
		Orgs:             []string{"Nudelmesse"},
	})
	if err != nil {
		logger.Fatal(err)
	}

	result := githubHost.Backup()

	out, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err)

	fmt.Println(string(out))

	for _, repoName := range []string{"public1", "public2"} {
		require.DirExists(t, path.Join(backupDir, "github.com", "Nudelmesse", repoName))

		var entries []os.DirEntry

		entries, err = os.ReadDir(path.Join(backupDir, "github.com", "Nudelmesse", repoName))
		require.NoError(t, err)
		require.Len(t, entries, 1)
		require.Regexp(t, regexp.MustCompile(`^public[1,2]\.\d{14}\.bundle$`), entries[0].Name())
	}
}

func TestGithubRepositoryBackupWithWildcardOrgsAndPersonal(t *testing.T) {
	if os.Getenv(envGitHubToken) == "" {
		t.Skipf(skipGitHubTestMissingToken, envGitHubToken)
	}

	_ = os.Unsetenv(envSobaWebHookURL)

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()

	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envGitHubToken, envGitHubCompare})
	// create dummy bundle
	backupDir := os.Getenv(envGitBackupDir)

	githubHost, err := githosts.NewGitHubHost(githosts.NewGitHubHostInput{
		Caller:           appName,
		LogLevel:         1,
		APIURL:           os.Getenv(envGitHubAPIURL),
		DiffRemoteMethod: os.Getenv(envGitHubCompare),
		BackupDir:        backupDir,
		Token:            os.Getenv(envGitHubToken),
		SkipUserRepos:    false,
		Orgs:             []string{"*"},
		BackupsToRetain:  1,
	})
	if err != nil {
		logger.Fatal(err)
	}

	result := githubHost.Backup()
	require.Len(t, result.BackupResults, 7)
	require.Nil(t, result.Error)

	for _, r := range result.BackupResults {
		require.Nil(t, r.Error)
	}

	require.Nil(t, result.Error)

	for _, repoName := range []string{"public1", "public2"} {
		require.DirExists(t, path.Join(backupDir, "github.com", "Nudelmesse", repoName))

		var entries []os.DirEntry

		entries, err = os.ReadDir(path.Join(backupDir, "github.com", "Nudelmesse", repoName))
		require.NoError(t, err)

		require.Len(t, entries, 1)
		require.Regexp(t, regexp.MustCompile(`^public[1,2]\.\d{14}\.bundle$`), entries[0].Name())
	}

	for _, repoName := range []string{"repo0", "repo1"} {
		require.DirExists(t, path.Join(backupDir, "github.com", goSobaOrg, repoName))

		var entries []os.DirEntry

		entries, err = os.ReadDir(path.Join(backupDir, "github.com", goSobaOrg, repoName))
		require.NoError(t, err)

		// one bundle in each folder
		require.Len(t, entries, 1)
		// repo2 has no commits and bundle not created for empty repos
		require.Regexp(t, regexp.MustCompile(`^repo[0,1]\.\d{14}\.bundle$`), entries[0].Name())
	}
}

func TestAzureDevOpsCredentialsFileSupport(t *testing.T) {
	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	unsetEnvVarsExcept([]string{envPath, envGitBackupDir, envAzureDevOpsUserName, envAzureDevOpsPAT, envAzureDevOpsOrgs, envAzureDevOpsBackups, envAzureDevOpsCompare})

	tempDir := t.TempDir()

	// Write username and PAT to files
	usernameFile := filepath.Join(tempDir, "az_username")
	patFile := filepath.Join(tempDir, "az_pat")
	os.WriteFile(usernameFile, []byte("fileuser"), 0o600)
	os.WriteFile(patFile, []byte("filepat"), 0o600)

	os.Setenv(envAzureDevOpsUserName+"_FILE", usernameFile)
	os.Setenv(envAzureDevOpsPAT+"_FILE", patFile)
	os.Setenv(envAzureDevOpsOrgs, "dummyorg")
	os.Setenv(envAzureDevOpsBackups, "1")
	os.Setenv(envAzureDevOpsCompare, "refs")

	// Should pick up credentials from files
	user := getEnvOrFile(envAzureDevOpsUserName)
	pat := getEnvOrFile(envAzureDevOpsPAT)
	require.Equal(t, "fileuser", user)
	require.Equal(t, "filepat", pat)

	// Now set env vars directly, which should take precedence
	os.Setenv(envAzureDevOpsUserName, "envuser")
	os.Setenv(envAzureDevOpsPAT, "envpat")

	user = getEnvOrFile(envAzureDevOpsUserName)
	pat = getEnvOrFile(envAzureDevOpsPAT)
	require.Equal(t, "envuser", user)
	require.Equal(t, "envpat", pat)
}
