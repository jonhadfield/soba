package internal

import (
	"os"

	"github.com/jonhadfield/githosts-utils"

	"gitlab.com/tozd/go/errors"
)

func Sourcehut(backupDir string) *ProviderBackupResults {
	logger.Println("backing up Sourcehut repos")

	ghToken, exists := GetEnvOrFile(envSourcehutToken)
	if !exists || ghToken == "" {
		logger.Println("Skipping Sourcehut backup as", envSourcehutToken, "is missing")

		return &ProviderBackupResults{
			Provider: providerNameSourcehut,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("Sourcehut token is not set"),
			},
		}
	}

	sourcehutHost, err := githosts.NewSourcehutHost(githosts.NewSourcehutHostInput{
		Caller:              AppName,
		BackupDir:           backupDir,
		HTTPClient:          httpClient,
		APIURL:              os.Getenv(envSourcehutAPIURL),
		DiffRemoteMethod:    os.Getenv(envSourcehutCompare),
		PersonalAccessToken: ghToken,
		BackupsToRetain:     getBackupsToRetain(envSourcehutBackups),
		LogLevel:            getLogLevel(),
		BackupLFS:           envTrue(envSourcehutBackupLFS),
	})
	if err != nil {
		return &ProviderBackupResults{
			Provider: providerNameSourcehut,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.Wrap(err, "failed to create Sourcehut host"),
			},
		}
	}

	return &ProviderBackupResults{
		Provider: providerNameSourcehut,
		Results:  sourcehutHost.Backup(),
	}
}
