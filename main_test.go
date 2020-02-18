package main

import (
	"os"
	"strings"
	"testing"
)

var sobaEnvVarKeys = []string{"GIT_BACKUP_DIR", "GITHUB_TOKEN", "GITLAB_TOKEN",
	"BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET"}

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

func TestPublicGithubRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	// Unset Env Vars but exclude those defined
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN"})
	err := run()
	restoreEnvironmentVariables(envBackup)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestPublicGitLabRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "GITLAB_TOKEN"})
	err := run()
	restoreEnvironmentVariables(envBackup)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestPublicBitBucketRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{"GIT_BACKUP_DIR", "BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET"})
	err := run()
	restoreEnvironmentVariables(envBackup)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestCheckProvidersFailureWhenNoneDefined(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{})
	err := checkProvidersDefined()
	restoreEnvironmentVariables(envBackup)
	if err == nil {
		t.Errorf("expected: no providers defined error")
	}
}

func TestFailureIfGitBackupDirUndefined(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVars([]string{})
	_ = os.Setenv("GITHUB_TOKEN", "ABCD1234")
	err := run()
	restoreEnvironmentVariables(envBackup)
	if err == nil {
		t.Errorf("expected: GIT_BACKUP_DIR undefined error")
	}
}
