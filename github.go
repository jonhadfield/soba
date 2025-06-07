package main

import (
	"os"

	"github.com/jonhadfield/githosts-utils"

	"gitlab.com/tozd/go/errors"
)

func GitHub(backupDir string) *ProviderBackupResults {
	logger.Println("backing up GitHub repos")

	ghToken, exists := GetEnvOrFile(envGitHubToken)
	if !exists || ghToken == "" {
		logger.Println("Skipping GitHub backup as", envGitHubToken, "is missing")

		return &ProviderBackupResults{
			Provider: providerNameGitHub,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("GitHub token is not set"),
			},
		}
	}

	githubHost, err := githosts.NewGitHubHost(githosts.NewGitHubHostInput{
		Caller:           appName,
		BackupDir:        backupDir,
		HTTPClient:       httpClient,
		APIURL:           os.Getenv(envGitHubAPIURL),
		DiffRemoteMethod: os.Getenv(envGitHubCompare),
		Token:            ghToken,
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
