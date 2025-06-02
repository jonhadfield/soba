package main

import (
	"os"

	"gitlab.com/tozd/go/errors"

	"github.com/jonhadfield/githosts-utils"
)

func Gitlab(backupDir string) *ProviderBackupResults {
	logger.Println("backing up GitLab repos")

	var gitlabHost *githosts.GitLabHost

	gitlabHost, err := githosts.NewGitLabHost(githosts.NewGitLabHostInput{
		Caller:                appName,
		HTTPClient:            httpClient,
		APIURL:                os.Getenv(envGitLabAPIURL),
		DiffRemoteMethod:      os.Getenv(envGitLabCompare),
		Token:                 getEnvOrFile(envGitLabToken),
		BackupDir:             backupDir,
		BackupsToRetain:       getBackupsToRetain(envGitLabBackups),
		ProjectMinAccessLevel: getProjectMinimumAccessLevel(),
		LogLevel:              getLogLevel(),
	})
	if err != nil {
		return &ProviderBackupResults{
			Provider: providerNameGitLab,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.Wrap(err, "failed to create GitLab host"),
			},
		}
	}

	return &ProviderBackupResults{
		Provider: providerNameGitLab,
		Results:  gitlabHost.Backup(),
	}
}
