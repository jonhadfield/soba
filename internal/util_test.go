package internal

import (
	"errors"
	"os/exec"
	"testing"

	githosts "github.com/jonhadfield/githosts-utils"
	"github.com/stretchr/testify/require"
	tozdErrors "gitlab.com/tozd/go/errors"
)

func TestFormatIntervalDurationAdditional(t *testing.T) {
	require.Equal(t, "", formatIntervalDuration(0))
	require.Equal(t, "1h", formatIntervalDuration(60))
	require.Equal(t, "1h1m0s", formatIntervalDuration(61))
	require.Equal(t, "3m0s", formatIntervalDuration(3))
}

func TestGetOrgsListFromEnvVarAdditional(t *testing.T) {
	t.Setenv("TEST_ORGS", "alpha,beta")
	list := getOrgsListFromEnvVar("TEST_ORGS")
	require.Equal(t, []string{"alpha", "beta"}, list)

	t.Setenv("TEST_ORGS", "")
	list = getOrgsListFromEnvVar("TEST_ORGS")
	require.Empty(t, list)
}

func TestGetRequestTimeoutAdditional(t *testing.T) {
	t.Setenv(envGitRequestTimeout, "600")
	ok, timeout, err := getRequestTimeout()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 600, int(timeout.Seconds()))

	t.Setenv(envGitRequestTimeout, "invalid")
	ok, _, err = getRequestTimeout()
	require.Error(t, err)
	require.False(t, ok)

	t.Setenv(envGitRequestTimeout, "")
	ok, timeout, err = getRequestTimeout()
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, defaultHTTPClientRequestTimeout, timeout)
}

func TestGitInstallPathAdditional(t *testing.T) {
	// ensure we detect git when exec.LookPath succeeds
	lookPath = func(file string) (string, error) { return "/usr/bin/git", nil }
	require.Equal(t, "/usr/bin/git", gitInstallPath())

	// when exec.LookPath fails, empty string returned
	lookPath = func(file string) (string, error) { return "", errors.New("missing") }
	require.Empty(t, gitInstallPath())

	lookPath = exec.LookPath
}

func TestGetBackupsStatsAdditional(t *testing.T) {
	// two success
	br := BackupResults{Results: &[]ProviderBackupResults{{
		Provider: "test",
		Results: githosts.ProviderBackupResult{
			BackupResults: []githosts.RepoBackupResults{{Repo: "a"}, {Repo: "b"}},
		},
	}}}

	ok, failed := getBackupsStats(br)
	require.Equal(t, 2, ok)
	require.Zero(t, failed)

	// one failed provider error
	br = BackupResults{Results: &[]ProviderBackupResults{{
		Provider: "test",
		Results:  githosts.ProviderBackupResult{Error: tozdErrors.New("err")},
	}}}

	ok, failed = getBackupsStats(br)
	require.Zero(t, ok)
	require.Equal(t, 1, failed)

	// repo error counts as failed
	br = BackupResults{Results: &[]ProviderBackupResults{{
		Provider: "test",
		Results: githosts.ProviderBackupResult{
			BackupResults: []githosts.RepoBackupResults{{Repo: "a", Error: tozdErrors.New("boom")}},
		},
	}}}
	ok, failed = getBackupsStats(br)
	require.Zero(t, ok)
	require.Equal(t, 1, failed)
}
