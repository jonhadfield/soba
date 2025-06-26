package internal

import (
	"os"

	"gitlab.com/tozd/go/errors"

	"github.com/jonhadfield/githosts-utils"
)

func Gitlab(backupDir string) *ProviderBackupResults {
	logger.Println("backing up GitLab repos")

	var gitlabHost *githosts.GitLabHost

	glToken, exists := GetEnvOrFile(envGitLabToken)
	if !exists || glToken == "" {
		logger.Println("Skipping GitLab backup as", envGitLabToken, "is missing")

		return &ProviderBackupResults{
			Provider: providerNameGitLab,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("GitLab token is not set"),
			},
		}
	}

	gitlabHost, err := githosts.NewGitLabHost(githosts.NewGitLabHostInput{
		Caller:                AppName,
		HTTPClient:            httpClient,
		APIURL:                os.Getenv(envGitLabAPIURL),
		DiffRemoteMethod:      os.Getenv(envGitLabCompare),
		Token:                 glToken,
		BackupDir:             backupDir,
		BackupsToRetain:       getBackupsToRetain(envGitLabBackups),
		ProjectMinAccessLevel: getProjectMinimumAccessLevel(),
		LogLevel:              getLogLevel(),
		BackupLFS:             envTrue(envGitLabBackupLFS),
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
