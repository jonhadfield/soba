package main

import (
	"os"

	"github.com/jonhadfield/githosts-utils"
	"gitlab.com/tozd/go/errors"
)

func Gitea(backupDir string) *ProviderBackupResults {
	logger.Println("backing up Gitea repos")

	giteaHost, err := githosts.NewGiteaHost(githosts.NewGiteaHostInput{
		Caller:           appName,
		BackupDir:        backupDir,
		HTTPClient:       httpClient,
		APIURL:           os.Getenv(envGiteaAPIURL),
		DiffRemoteMethod: os.Getenv(envGiteaCompare),
		Token:            getEnvOrFile(envGiteaToken),
		Orgs:             getOrgsListFromEnvVar(envGiteaOrgs),
		BackupsToRetain:  getBackupsToRetain(envGiteaBackups),
		LogLevel:         getLogLevel(),
	})
	if err != nil {
		return &ProviderBackupResults{
			Provider: providerNameGitea,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.Wrap(err, "failed to create Gitea host"),
			},
		}
	}

	return &ProviderBackupResults{
		Provider: providerNameGitea,
		Results:  giteaHost.Backup(),
	}
}
