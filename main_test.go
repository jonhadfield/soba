package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/exp/slices"

	"github.com/stretchr/testify/require"
)

var sobaEnvVarKeys = []string{
	"GIT_BACKUP_DIR", "GITHUB_TOKEN", "GITHUB_BACKUPS", "GITLAB_TOKEN", "GITLAB_BACKUPS", "GITLAB_APIURL", "SOBA_DEV",
	"BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET", "BITBUCKET_BACKUPS",
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		if err = d.Close(); err != nil {
			panic(err)
		}
	}()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

func preflight() {
	var err error

	// prepare clean backup directory
	bud := os.Getenv("GIT_BACKUP_DIR")

	// if path not provided, create one
	if bud == "" {
		bud, err = os.MkdirTemp(os.TempDir(), "sobabackup-*")
		if err != nil {
			panic(err)
		}

		_ = os.Setenv("GIT_BACKUP_DIR", bud)

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
	backupDir := os.Getenv("GIT_BACKUP_DIR")
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
	require.NoError(t, os.Unsetenv("GITHUB_TOKEN"))
	require.NoError(t, os.Unsetenv("GITLAB_TOKEN"))
	require.NoError(t, os.Setenv("GITHUB_ORGS", "example,example2"))
	err := run()
	require.NoError(t, os.Unsetenv("GITHUB_ORGS"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "GITHUB_TOKEN must be set if GITHUB_ORGS is set")
}

func TestInvalidBundleIsMoved(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN", "SOBA_DEV"})
	// create invalid bundle
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
	require.Equal(t, renamedFound, 1)
}

func TestPublicGithubRepositoryBackupWithBackupsToKeepAsOne(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN", "SOBA_DEV"})
	// create dummy bundle
	backupDir := os.Getenv("GIT_BACKUP_DIR")
	dfDir := path.Join(backupDir, "github.com", "go-soba", "repo0")
	require.NoError(t, os.MkdirAll(dfDir, 0o755))
	require.NoError(t, os.Setenv("GITHUB_BACKUPS", "1"))
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
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN", "SOBA_DEV"})
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
}

func TestPublicGithubRepositoryBackup(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN", "SOBA_DEV"})
	require.NoError(t, run())
}

func TestPublicGithubRepositoryBackupWithExistingBackups(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub test as GITHUB_TOKEN is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	// Unset Env Vars but exclude those defined
	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "GITHUB_TOKEN", "SOBA_DEV"})
	require.NoError(t, run())
	// run for second time now we have existing bundles
	require.NoError(t, run())
}

func TestPublicGitLabRepositoryBackup(t *testing.T) {
	if os.Getenv("GITLAB_TOKEN") == "" {
		t.Skip("Skipping GitLab test as GITLAB_TOKEN is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "GITLAB_TOKEN"})
	require.NoError(t, run())
}

func TestPublicGitLabRepositoryBackup2(t *testing.T) {
	if os.Getenv("GITLAB_TOKEN") == "" {
		t.Skip("Skipping GitLab test as GITLAB_TOKEN is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "GITLAB_TOKEN"})
	require.NoError(t, run())
}

func TestPublicBitBucketRepositoryBackupWithRefCompare(t *testing.T) {
	if os.Getenv("BITBUCKET_USER") == "" {
		t.Skip("Skipping BitBucket test as BITBUCKET_USER is missing")
	}
	resetGlobals()
	envBackup := backupEnvironmentVariables()
	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET"})
	defer os.Unsetenv("SOBA_DEV")
	os.Setenv("SOBA_DEV", "true")
	require.NoError(t, run())
	require.NoError(t, run())
	restoreEnvironmentVariables(envBackup)
}

func TestPublicBitBucketRepositoryBackup(t *testing.T) {
	if os.Getenv("BITBUCKET_USER") == "" {
		t.Skip("Skipping BitBucket test as BITBUCKET_USER is missing")
	}

	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	unsetEnvVarsExcept([]string{"GIT_BACKUP_DIR", "BITBUCKET_USER", "BITBUCKET_KEY", "BITBUCKET_SECRET"})
	require.NoError(t, run())
}

func TestCheckProvidersFailureWhenNoneDefined(t *testing.T) {
	envBackup := backupEnvironmentVariables()
	defer restoreEnvironmentVariables(envBackup)

	preflight()
	resetGlobals()
	defer resetBackups()

	unsetEnvVarsExcept([]string{})
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

	unsetEnvVarsExcept([]string{})
	_ = os.Setenv("GITHUB_TOKEN", "ABCD1234")
	require.Error(t, run(), "expected: GIT_BACKUP_DIR undefined error")
}
