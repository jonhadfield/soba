package internal

import (
	"os"

	"gitlab.com/tozd/go/errors"

	"github.com/jonhadfield/githosts-utils"
)

func Bitbucket(backupDir string) *ProviderBackupResults {
	logger.Println("backing up BitBucket repos")

	// Check for API OAuthToken authentication (preferred method)
	bbEmail, emailExists := GetEnvOrFile(envBitBucketEmail)
	bbAPIToken, tokenExists := GetEnvOrFile(envBitBucketAPIToken)

	// Check for OAuth2 authentication (legacy method)
	bbUser, userExists := GetEnvOrFile(envBitBucketUser)
	bbKey, keyExists := GetEnvOrFile(envBitBucketKey)
	bbSecret, secretExists := GetEnvOrFile(envBitBucketSecret)

	// Validate that at least one complete authentication method is available
	apiTokenComplete := emailExists && bbEmail != "" && tokenExists && bbAPIToken != ""
	oauth2Complete := userExists && bbUser != "" && keyExists && bbKey != "" && secretExists && bbSecret != ""

	if !apiTokenComplete && !oauth2Complete {
		logger.Println("Skipping BitBucket backup: neither API OAuthToken nor OAuth2 authentication is properly configured")
		logger.Println("API OAuthToken method requires:", envBitBucketEmail, "and", envBitBucketAPIToken)
		logger.Println("OAuth2 method requires:", envBitBucketUser, ",", envBitBucketKey, "and", envBitBucketSecret)

		return &ProviderBackupResults{
			Provider: providerNameBitBucket,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("BitBucket authentication not properly configured"),
			},
		}
	}

	var authType string

	if apiTokenComplete {
		logger.Println("Using BitBucket API OAuthToken authentication")

		authType = githosts.AuthTypeBitbucketAPIToken
	} else {
		logger.Println("Using BitBucket OAuth2 authentication")

		authType = githosts.AuthTypeBitbucketOAuth2
	}

	bundlePassphrase, _ := GetEnvOrFile(envVarBundlePassphrase)

	bitbucketHost, err := githosts.NewBitBucketHost(githosts.NewBitBucketHostInput{
		Caller:               AppName,
		HTTPClient:           httpClient,
		APIURL:               os.Getenv(envBitBucketAPIURL),
		DiffRemoteMethod:     os.Getenv(envBitBucketCompare),
		BackupDir:            backupDir,
		Email:                bbEmail,
		BasicAuth:            githosts.BasicAuth{},
		AuthType:             authType,
		APIToken:             bbAPIToken,
		User:                 bbUser,
		Key:                  bbKey,
		Secret:               bbSecret,
		BackupsToRetain:      getBackupsToRetain(envBitBucketBackups),
		LogLevel:             getLogLevel(),
		BackupLFS:            envTrue(envBitBucketBackupLFS),
		EncryptionPassphrase: bundlePassphrase,
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
