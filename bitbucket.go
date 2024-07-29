package main

import (
	"os"

	"gitlab.com/tozd/go/errors"

	"github.com/jonhadfield/githosts-utils"
)

func Bitbucket(backupDir string) *ProviderBackupResults {
	logger.Println("backing up BitBucket repos")

	bitbucketHost, err := githosts.NewBitBucketHost(githosts.NewBitBucketHostInput{
		Caller:           appName,
		HTTPClient:       httpClient,
		APIURL:           os.Getenv(envBitBucketAPIURL),
		DiffRemoteMethod: os.Getenv(envBitBucketCompare),
		BackupDir:        backupDir,
		User:             os.Getenv(envBitBucketUser),
		Key:              os.Getenv(envBitBucketKey),
		Secret:           os.Getenv(envBitBucketSecret),
		BackupsToRetain:  getBackupsToRetain(envBitBucketBackups),
		LogLevel:         getLogLevel(),
	})
	if err != nil {
		return &ProviderBackupResults{
			Provider: providerNameBitBucket,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.Wrap(err, "failed to create BitBucket host"),
			},
		}
	}

	return &ProviderBackupResults{
		Provider: providerNameBitBucket,
		Results:  bitbucketHost.Backup(),
	}
}
