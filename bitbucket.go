package main

import (
	"os"

	"gitlab.com/tozd/go/errors"

	"github.com/jonhadfield/githosts-utils"
)

func Bitbucket(backupDir string) *ProviderBackupResults {
	logger.Println("backing up BitBucket repos")

	bbUser, exists := GetEnvOrFile(envBitBucketUser)
	if !exists || bbUser == "" {
		logger.Println("Skipping BitBucket backup as", envBitBucketUser, "is missing")
		return &ProviderBackupResults{
			Provider: providerNameBitBucket,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("BitBucket user is not set"),
			},
		}
	}

	bbKey, exists := GetEnvOrFile(envBitBucketKey)
	if !exists || bbKey == "" {
		logger.Println("Skipping BitBucket backup as", envBitBucketKey, "is missing")
		return &ProviderBackupResults{
			Provider: providerNameBitBucket,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("BitBucket key is not set"),
			},
		}
	}

	bbSecret, exists := GetEnvOrFile(envBitBucketSecret)
	if !exists || bbSecret == "" {
		logger.Println("Skipping BitBucket backup as", envBitBucketSecret, "is missing")
		return &ProviderBackupResults{
			Provider: providerNameBitBucket,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("BitBucket secret is not set"),
			},
		}
	}

	bitbucketHost, err := githosts.NewBitBucketHost(githosts.NewBitBucketHostInput{
		Caller:           appName,
		BackupDir:        backupDir,
		HTTPClient:       httpClient,
		APIURL:           os.Getenv(envBitBucketAPIURL),
		DiffRemoteMethod: os.Getenv(envBitBucketCompare),
		User:             bbUser,
		Key:              bbKey,
		Secret:           bbSecret,
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
