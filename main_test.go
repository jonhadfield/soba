package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sobaEnvVarKeys = []string{
	"GIT_BACKUP_DIR", "GITHUB_TOKEN", "GITHUB_BACKUPS", "GITLAB_TOKEN", "GITLAB_BACKUPS",
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
		if !stringInStrings(sobaVar, exceptionList) {
			_ = os.Unsetenv(sobaVar)
		}
	}
}

func resetBackups() {
	os.RemoveAll(os.Getenv("GIT_BACKUP_DIR"))
	if err := os.MkdirAll(os.Getenv("GIT_BACKUP_DIR"), 0o755); err != nil {
		log.Fatal(err)
	}
}

func TestPublicGithubRepositoryBackupWithBackupsToKeepAsOne(t *testing.T) {
	preflight()
	resetGlobals()
	defer resetBackups()
	envBackup := backupEnvironmentVariables()
	// Unset Env Vars but exclude those defined
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN"})
	// create dummy bundle
	backupDir := os.Getenv("GIT_BACKUP_DIR")
	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
	assert.NoError(t, os.MkdirAll(dfDir, 0o755))
	dfName := "repo0.20200401111111.bundle"
	dfPath := path.Join(dfDir, dfName)
	_, err := os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
	assert.NoError(t, err)
	assert.NoError(t, os.Setenv("GITHUB_BACKUPS", "1"))
	// run
	assert.NoError(t, run())
	// check only one bundle remains
	files, err := ioutil.ReadDir(dfDir)
	assert.NoError(t, err)
	var found int
	for _, f := range files {
		assert.NotEqual(t, f.Name(), dfName, fmt.Sprintf("unexpected bundle: %s", f.Name()))
		found++
	}
	assert.Equal(t, found, 1)
	// reset
	restoreEnvironmentVariables(envBackup)
}

func TestPublicGithubRepositoryBackupWithBackupsToKeepUnset(t *testing.T) {
	preflight()
	resetGlobals()
	defer resetBackups()

	envBackup := backupEnvironmentVariables()
	// Unset Env Vars but exclude those defined
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN"})
	// create dummy bundle
	backupDir := os.Getenv("GIT_BACKUP_DIR")
	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
	assert.NoError(t, os.MkdirAll(dfDir, 0o755))
	dfName := "repo0.20200401111111.bundle"
	dfPath := path.Join(dfDir, dfName)
	_, err := os.OpenFile(dfPath, os.O_RDONLY|os.O_CREATE, 0o666)
	assert.NoError(t, err)
	// run
	assert.NoError(t, run())
	// check both bundles remain
	files, err := ioutil.ReadDir(dfDir)
	assert.NoError(t, err)
	assert.Len(t, files, 2)
	// reset
	restoreEnvironmentVariables(envBackup)
}

func TestPublicGithubRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	// Unset Env Vars but exclude those defined
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN"})
	assert.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestPublicGitLabRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITLAB_TOKEN"})
	assert.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestPublicBitBucketRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET"})
	assert.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestCheckProvidersFailureWhenNoneDefined(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{})
	assert.Error(t, checkProvidersDefined(), "expected: no providers defined error")
	restoreEnvironmentVariables(envBackup)
}

func TestFailureIfGitBackupDirUndefined(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{})
	_ = os.Setenv("GITHUB_TOKEN", "ABCD1234")
	assert.Error(t, run(), "expected: GIT_BACKUP_DIR undefined error")
	restoreEnvironmentVariables(envBackup)
}
