package main

import (
	"os"

	"github.com/jonhadfield/githosts-utils"
	"gitlab.com/tozd/go/errors"
)

func AzureDevOps(backupDir string) *ProviderBackupResults {
	logger.Println("backing up Azure DevOps repos")

	azureDevOpsHost, err := githosts.NewAzureDevOpsHost(githosts.NewAzureDevOpsHostInput{
		Caller:           appName,
		HTTPClient:       httpClient,
		BackupDir:        backupDir,
		DiffRemoteMethod: os.Getenv(envAzureDevOpsCompare),
		UserName:         os.Getenv(envAzureDevOpsUserName),
		PAT:              os.Getenv(envAzureDevOpsPAT),
		Orgs:             getOrgsListFromEnvVar(envAzureDevOpsOrgs),
		BackupsToRetain:  getBackupsToRetain(envAzureDevOpsBackups),
		LogLevel:         getLogLevel(),
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
