package internal

import (
	"log"
	"os"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	AppName                                = "soba"
	workingDIRName                         = ".working"
	workingDIRMode                         = 0o755
	defaultBackupsToRetain                 = 2
	defaultGitLabMinimumProjectAccessLevel = 20

	defaultHTTPClientRequestTimeout = 300 * time.Second

	// general constants
	pathSep        = string(os.PathSeparator)
	minutesPerHour = 60

	// retry settings
	httpRetryWaitMax    = 120 * time.Second
	httpRetryWaitMin    = 60 * time.Second
	httpRetryMax        = 2
	webhookRetryWaitMin = 1 * time.Second
	webhookRetryWaitMax = 3 * time.Second
	webhookRetryMax     = 3

	// http
	maxIdleConns    = 10
	idleConnTimeout = 30 * time.Second

	// env vars
	envPath                 = "PATH"
	envSobaLogLevel         = "SOBA_LOG"
	envSobaWebHookURL       = "SOBA_WEBHOOK_URL"
	envSobaWebHookFormat    = "SOBA_WEBHOOK_FORMAT"
	envGitBackupInterval    = "GIT_BACKUP_INTERVAL"
	envGitBackupCron        = "GIT_BACKUP_CRON"
	envGitBackupDir         = "GIT_BACKUP_DIR"
	envGitRequestTimeout    = "GIT_REQUEST_TIMEOUT"
	envGitHubAPIURL         = "GITHUB_APIURL"
	envGitHubBackups        = "GITHUB_BACKUPS"
	envGitHubBackupLFS      = "GITHUB_BACKUP_LFS"
	envAzureDevOpsOrgs      = "AZURE_DEVOPS_ORGS"
	envAzureDevOpsUserName  = "AZURE_DEVOPS_USERNAME"
	envAzureDevOpsPAT       = "AZURE_DEVOPS_PAT"
	envAzureDevOpsCompare   = "AZURE_DEVOPS_COMPARE"
	envAzureDevOpsBackups   = "AZURE_DEVOPS_BACKUPS"
	envAzureDevOpsBackupLFS = "AZURE_DEVOPS_BACKUP_LFS"
	// nolint:gosec
	envGitHubToken          = "GITHUB_TOKEN"
	envGitHubOrgs           = "GITHUB_ORGS"
	envGitHubSkipUserRepos  = "GITHUB_SKIP_USER_REPOS"
	envGitHubLimitUserOwned = "GITHUB_LIMIT_USER_OWNED"
	envGitHubCompare        = "GITHUB_COMPARE"
	envGitLabBackups        = "GITLAB_BACKUPS"
	envGitLabBackupLFS      = "GITLAB_BACKUP_LFS"
	envGitLabMinAccessLevel = "GITLAB_PROJECT_MIN_ACCESS_LEVEL"
	envGitLabToken          = "GITLAB_TOKEN"
	envGitLabAPIURL         = "GITLAB_APIURL"
	envGitLabCompare        = "GITLAB_COMPARE"
	envBitBucketUser        = "BITBUCKET_USER"
	envBitBucketKey         = "BITBUCKET_KEY"
	envBitBucketSecret      = "BITBUCKET_SECRET"
	envBitBucketEmail       = "BITBUCKET_EMAIL"
	envBitBucketAPIToken    = "BITBUCKET_API_TOKEN"
	envBitBucketAPIURL      = "BITBUCKET_APIURL"
	envBitBucketCompare     = "BITBUCKET_COMPARE"
	envBitBucketBackups     = "BITBUCKET_BACKUPS"
	envBitBucketBackupLFS   = "BITBUCKET_BACKUP_LFS"
	envGiteaToken           = "GITEA_TOKEN"
	envGiteaAPIURL          = "GITEA_APIURL"
	envGiteaBackups         = "GITEA_BACKUPS"
	envGiteaBackupLFS       = "GITEA_BACKUP_LFS"
	envGiteaCompare         = "GITEA_COMPARE"
	envGiteaOrgs            = "GITEA_ORGS"

	// provider names
	providerNameAzureDevOps       = "AzureDevOps"
	providerNameBitBucket         = "BitBucket"
	providerNameBitBucketOAuth    = "BitBucketOAuth"
	providerNameBitBucketAPIToken = "BitBucketAPIToken"
	providerNameGitHub            = "GitHub"
	providerNameGitLab            = "GitLab"
	providerNameGitea             = "Gitea"

	// compare types
	compareTypeRefs  = "refs"
	compareTypeClone = "clone"
)

var (
	logger *log.Logger

	httpClient *retryablehttp.Client

	enabledProviderAuth = map[string][]string{
		providerNameAzureDevOps: {
			envAzureDevOpsUserName,
			envAzureDevOpsPAT,
		},
		providerNameGitHub: {
			envGitHubToken,
		},
		providerNameGitLab: {
			envGitLabToken,
		},
		providerNameBitBucketAPIToken: {
			envBitBucketEmail,
			envBitBucketAPIToken,
		},
		providerNameBitBucketOAuth: {
			envBitBucketUser,
			envBitBucketKey,
			envBitBucketSecret,
		},
		providerNameGitea: {
			envGiteaAPIURL,
			envGiteaToken,
		},
	}
	justTokenProviders = []string{
		providerNameGitHub,
		providerNameGitLab,
		providerNameGitea,
	}
	userAndPasswordProviders = []string{
		providerNameBitBucketAPIToken,
		providerNameBitBucketOAuth,
		providerNameAzureDevOps,
	}
	numUserDefinedProviders int64
)
