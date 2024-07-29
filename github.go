package main

import (
	"os"

	"github.com/jonhadfield/githosts-utils"

	"gitlab.com/tozd/go/errors"
)

func GitHub(backupDir string) *ProviderBackupResults {
	logger.Println("backing up GitHub repos")

	githubHost, err := githosts.NewGitHubHost(githosts.NewGitHubHostInput{
		Caller:           appName,
		HTTPClient:       httpClient,
		APIURL:           os.Getenv(envGitHubAPIURL),
		DiffRemoteMethod: os.Getenv(envGitHubCompare),
		BackupDir:        backupDir,
		Token:            os.Getenv(envGitHubToken),
		Orgs:             getOrgsListFromEnvVar(envGitHubOrgs),
		BackupsToRetain:  getBackupsToRetain(envGitHubBackups),
		SkipUserRepos:    envTrue(envGitHubSkipUserRepos),
		LimitUserOwned:   envTrue(envGitHubLimitUserOwned),
		LogLevel:         getLogLevel(),
	})
	if err != nil {
		return &ProviderBackupResults{
			Provider: providerNameGitHub,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.Wrap(err, "failed to create GitHub host"),
			},
		}
	}

	return &ProviderBackupResults{
		Provider: providerNameGitHub,
		Results:  githubHost.Backup(),
	}
}
