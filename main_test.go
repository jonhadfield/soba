package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/stretchr/testify/require"
)

var sobaEnvVarKeys = []string{
	"GIT_BACKUP_DIR", "GITHUB_TOKEN", "GITHUB_BACKUPS", "GITLAB_TOKEN", "GITLAB_BACKUPS", "GITLAB_APIURL", "SOBA_DEV",
	"BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET", "BITBUCKET_BACKUPS",
}

func preflight() {
	// create backup dir if defined but missing
	bud := os.Getenv("GIT_BACKUP_DIR")
	if bud == "" {
		bud = os.TempDir()
	}

	_, err := os.Stat(bud)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(bud, 0o755)
		if errDir != nil {
			log.Fatal(err)
		}
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

func unsetEnvVars(exceptionList []string) {
	for _, sobaVar := range sobaEnvVarKeys {
		if !slices.Contains(exceptionList, sobaVar) {
			_ = os.Unsetenv(sobaVar)
		}
	}
}

func resetBackups() {
	_ = os.RemoveAll(os.Getenv("GIT_BACKUP_DIR"))
	if err := os.MkdirAll(os.Getenv("GIT_BACKUP_DIR"), 0o755); err != nil {
		log.Fatal(err)
	}
}

func TestPublicGithubRepositoryBackupWithBackupsToKeepAsOne(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}
	preflight()
	resetGlobals()
	defer resetBackups()
	envBackup := backupEnvironmentVariables()
	// Unset Env Vars but exclude those defined
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN"})
	// create dummy bundle
	backupDir := os.Getenv("GIT_BACKUP_DIR")
	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
	require.NoError(t, os.MkdirAll(dfDir, 0o755))
	dfName := "repo0.20200401111111.bundle"
	dfPath := path.Join(dfDir, dfName)
	_, err := os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
	require.NoError(t, err)
	require.NoError(t, os.Setenv("GITHUB_BACKUPS", "1"))
	// run
	require.NoError(t, run())
	// check only one bundle remains
	files, err := os.ReadDir(dfDir)
	require.NoError(t, err)
	var found int
	for _, f := range files {
		require.NotEqual(t, f.Name(), dfName, fmt.Sprintf("unexpected bundle: %s", f.Name()))
		found++
	}
	require.Equal(t, found, 1)
	// reset
	restoreEnvironmentVariables(envBackup)
}

func TestPublicGithubRepositoryBackupWithBackupsToKeepUnset(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}
	preflight()
	resetGlobals()
	defer resetBackups()

	envBackup := backupEnvironmentVariables()
	// Unset Env Vars but exclude those defined
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN"})
	// create dummy bundle
	backupDir := os.Getenv("GIT_BACKUP_DIR")
	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
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
	// reset
	restoreEnvironmentVariables(envBackup)
}

func TestPublicGithubRepositoryBackup(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	// Unset Env Vars but exclude those defined
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN"})
	require.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestPublicGitLabRepositoryBackup(t *testing.T) {
	if os.Getenv("GITLAB_TOKEN") == "" {
		t.Skip("Skipping GitLab test as GITLAB_TOKEN is missing")
	}
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITLAB_TOKEN"})
	require.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestPublicGitLabRepositoryBackup2(t *testing.T) {
	if os.Getenv("GITLAB_TOKEN") == "" {
		t.Skip("Skipping GitLab test as GITLAB_TOKEN is missing")
	}
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITLAB_TOKEN"})
	require.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestPublicBitBucketRepositoryBackup(t *testing.T) {
	if os.Getenv("BITBUCKET_USER") == "" {
		t.Skip("Skipping BitBucket test as BITBUCKET_USER is missing")
	}
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET"})
	require.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestCheckProvidersFailureWhenNoneDefined(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{})
	require.Error(t, checkProvidersDefined(), "expected: no providers defined error")
	restoreEnvironmentVariables(envBackup)
}

func TestFailureIfGitBackupDirUndefined(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{})
	_ = os.Setenv("GITHUB_TOKEN", "ABCD1234")
	require.Error(t, run(), "expected: GIT_BACKUP_DIR undefined error")
	restoreEnvironmentVariables(envBackup)
}
