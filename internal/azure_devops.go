package internal

import (
	"os"

	"github.com/jonhadfield/githosts-utils"
	"gitlab.com/tozd/go/errors"
)

func AzureDevOps(backupDir string) *ProviderBackupResults {
	logger.Println("backing up Azure DevOps repos")

	adou, exists := GetEnvOrFile(envAzureDevOpsUserName)
	if !exists || adou == "" {
		logger.Println("Skipping Azure DevOps backup as", envAzureDevOpsUserName, "is missing")

		return &ProviderBackupResults{
			Provider: providerNameAzureDevOps,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("Azure DevOps username is not set"),
			},
		}
	}

	pat, exists := GetEnvOrFile(envAzureDevOpsPAT)
	if !exists || pat == "" {
		logger.Println("Skipping Azure DevOps backup as", envAzureDevOpsPAT, "is missing")

		return &ProviderBackupResults{
			Provider: providerNameAzureDevOps,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("Azure DevOps PAT is not set"),
			},
		}
	}

	bundlePassphrase, _ := GetEnvOrFile(envVarBundlePassphrase)

	azureDevOpsHost, err := githosts.NewAzureDevOpsHost(githosts.NewAzureDevOpsHostInput{
		Caller:               AppName,
		HTTPClient:           httpClient,
		BackupDir:            backupDir,
		DiffRemoteMethod:     os.Getenv(envAzureDevOpsCompare),
		UserName:             adou,
		PAT:                  pat,
		Orgs:                 getOrgsListFromEnvVar(envAzureDevOpsOrgs),
		BackupsToRetain:      getBackupsToRetain(envAzureDevOpsBackups),
		LogLevel:             getLogLevel(),
		BackupLFS:            envTrue(envAzureDevOpsBackupLFS),
		EncryptionPassphrase: bundlePassphrase,
	})
	if err != nil {
		return &ProviderBackupResults{
			Provider: providerNameAzureDevOps,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.Wrap(err, "failed to create AzureDevOps host"),
			},
		}
	}

	return &ProviderBackupResults{
		Provider: providerNameAzureDevOps,
		Results:  azureDevOpsHost.Backup(),
	}
}
