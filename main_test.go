package main

import (
	"testing"
	"os"
)

//func backupEnvVars() {
//
//}

//func restoreEnvVars() {
//
//}

func resetEnvironmentAndGlobals() {
	// TODO: Use predefined providers to determine which to first unset
	// reset global var
	numUserDefinedProviders = 0
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GITLAB_TOKEN")
	os.Unsetenv("GIT_BACKUP_DIR")
}
func TestPublicGithubRepositoryBackup(t *testing.T) {
	err := run()
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestCheckProvidersFailureWhenNonDefined(t *testing.T) {
	resetEnvironmentAndGlobals()
	os.Unsetenv("GITHUB_TOKEN")
	err := checkProvidersDefined()
	if err == nil {
		t.Errorf("expected: no providers defined error")
	}
}

func TestFailureIfGitBackupDirUndefined(t *testing.T) {
	resetEnvironmentAndGlobals()
	os.Setenv("GITHUB_TOKEN", "ABCD1234")
	err := run()
	if err == nil {
		t.Errorf("expected: GIT_BACKUP_DIR undefined error")
	}
}
