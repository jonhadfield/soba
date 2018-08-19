package main

import (
	"testing"
	"os"
	"strings"
	)

var sobaEnvVarKeys = []string{"GIT_BACKUP_DIR", "GITHUB_TOKEN", "GITLAB_TOKEN"}

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

func restoreEnvironmentVariables(input map[string]string)  {
	for key, val := range input {
		os.Setenv(key, val)
	}
}

func unsetEnvVars(exceptionList []string) {
	for _, sobaVar := range sobaEnvVarKeys {
		if ! stringInStrings(sobaVar, exceptionList) {
			os.Unsetenv(sobaVar)
		}
	}
}



func TestPublicGithubRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	err := run()
	restoreEnvironmentVariables(envBackup)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestPublicGitLabRepositoryBackup(t *testing.T) {
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	err := run()
	restoreEnvironmentVariables(envBackup)
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestCheckProvidersFailureWhenNonDefined(t *testing.T) {
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
	os.Setenv("GITHUB_TOKEN", "ABCD1234")
	err := run()
	restoreEnvironmentVariables(envBackup)
	if err == nil {
		t.Errorf("expected: GIT_BACKUP_DIR undefined error")
	}
}
