// and returns the same result type. Keeping them parallel and independent is
// deliberate: collapsing them into a parameterised helper would hide which
// provider does what, for no behavioural gain.
//
//nolint:dupl // Each provider reads its own variables, builds its own host
package internal

import (
	"os"

	"github.com/jonhadfield/githosts-utils/v2"
	"gitlab.com/tozd/go/errors"
)

func Codeberg(backupDir string) *ProviderBackupResults {
	logger.Println("backing up Codeberg repos")

	codebergToken, exists := GetEnvOrFile(envCodebergToken)
	if !exists || codebergToken == "" {
		logger.Println("Skipping Codeberg backup as", envCodebergToken, "is missing")

		return &ProviderBackupResults{
			Provider: providerNameCodeberg,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.New("Codeberg token is not set"),
			},
		}
	}

	bundlePassphrase, _ := GetEnvOrFile(envVarBundlePassphrase)

	codebergHost, err := githosts.NewCodebergHost(githosts.NewCodebergHostInput{
		Caller:               AppName,
		BackupDir:            backupDir,
		HTTPClient:           httpClient,
		APIURL:               os.Getenv(envCodebergAPIURL),
		DiffRemoteMethod:     os.Getenv(envCodebergCompare),
		Token:                codebergToken,
		Orgs:                 getOrgsListFromEnvVar(envCodebergOrgs),
		SkipUserRepos:        envTrue(envCodebergSkipUserRepos),
		LimitUserOwned:       envTrue(envCodebergLimitUserOwned),
		BackupsToRetain:      getBackupsToRetain(envCodebergBackups),
		LogLevel:             getLogLevel(),
		BackupLFS:            envTrue(envCodebergBackupLFS),
		EncryptionPassphrase: bundlePassphrase,
	})
	if err != nil {
		return &ProviderBackupResults{
			Provider: providerNameCodeberg,
			Results: githosts.ProviderBackupResult{
				BackupResults: []githosts.RepoBackupResults{},
				Error:         errors.Wrap(err, "failed to create Codeberg host"),
			},
		}
	}

	return &ProviderBackupResults{
		Provider: providerNameCodeberg,
		Results:  codebergHost.Backup(),
	}
}
