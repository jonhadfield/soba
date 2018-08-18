package main

import (
	"testing"
	"os"
)

func resetAllEnvVars() {
	// TODO: Use predefined providers to determine which to first unset
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("GITLAB_TOKEN")
}

func TestCheckProvidersWhenNonDefined(t *testing.T) {
	resetAllEnvVars()
	err := checkProvidersDefined()
	if err == nil {
		t.Errorf("expected: no providers defined error")
	}
}
