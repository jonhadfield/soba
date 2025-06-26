package internal

import (
	"os"

	"github.com/jonhadfield/githosts-utils"
	"gitlab.com/tozd/go/errors"
)

func Gitea(backupDir string) *ProviderBackupResults {
	logger.Println("backing up Gitea repos")

	giteaToken, exists := GetEnvOrFile(envGiteaToken)
	if !exists || giteaToken == "" {
		logger.Println("Skipping Gitea backup as", envGiteaToken, "is missing")

		return &ProviderBackupResults{
			Provider: providerNameGitea,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("Gitea token is not set"),
			},
		}
	}

	giteaHost, err := githosts.NewGiteaHost(githosts.NewGiteaHostInput{
		Caller:           AppName,
		BackupDir:        backupDir,
		HTTPClient:       httpClient,
		APIURL:           os.Getenv(envGiteaAPIURL),
		DiffRemoteMethod: os.Getenv(envGiteaCompare),
		Token:            giteaToken,
		Orgs:             getOrgsListFromEnvVar(envGiteaOrgs),
		BackupsToRetain:  getBackupsToRetain(envGiteaBackups),
		LogLevel:         getLogLevel(),
		BackupLFS:        envTrue(envGiteaBackupLFS),
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
