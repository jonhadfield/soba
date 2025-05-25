package githosts

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/microsoft/azure-devops-go-api/azuredevops/v7"
	"gitlab.com/tozd/go/errors"

	azdevopscore "github.com/microsoft/azure-devops-go-api/azuredevops/v7/core"
)

const (
	sUsingDiffRemoteMethod            = "using diff remote method"
	sUsingDefaultDiffRemoteMethod     = "using default diff remote method"
	AzureDevOpsProviderName           = "AzureDevOps"
	azureDevOpsDomain                 = "dev.azure.com"
	envAzureDevOpsUserName            = "AZURE_DEVOPS_USERNAME"
	msgSkipAzureDevOpsUserNameMissing = "Skipping Azure DevOps test as " + envAzureDevOpsUserName + " is missing"
)

func (ad *AzureDevOpsHost) Backup() ProviderBackupResult {
	if ad.BackupDir == "" {
		logger.Printf("backup skipped as backup directory not specified")

		return ProviderBackupResult{
			BackupResults: nil,
			Error:         errors.New("backup directory not specified"),
		}
	}

	maxConcurrent := 10

	repoDesc, err := ad.describeRepos()
	if err != nil {
		return ProviderBackupResult{
			BackupResults: nil,
			Error:         err,
		}
	}

	jobs := make(chan repository, len(repoDesc.Repos))
	results := make(chan RepoBackupResults, maxConcurrent)

	for w := 1; w <= maxConcurrent; w++ {
		go azureDevOpsWorker(ad.LogLevel, ad.BackupDir, ad.DiffRemoteMethod, ad.BackupsToRetain, jobs, results)
	}

	for x := range repoDesc.Repos {
		repo := repoDesc.Repos[x]
		jobs <- repo
	}

	close(jobs)

	var providerBackupResults ProviderBackupResult

	for a := 1; a <= len(repoDesc.Repos); a++ {
		res := <-results
		if res.Error != nil {
			logger.Printf("backup failed: %+v\n", res.Error)
		}

		providerBackupResults.BackupResults = append(providerBackupResults.BackupResults, res)
	}

	return providerBackupResults
}

func azureDevOpsWorker(logLevel int, backupDIR, diffRemoteMethod string, backupsToKeep int,
	jobs <-chan repository, results chan<- RepoBackupResults,
) {
	for repo := range jobs {
		err := processBackup(logLevel, repo, backupDIR, backupsToKeep, diffRemoteMethod)

		backupResult := RepoBackupResults{
			Repo: repo.PathWithNameSpace,
		}

		status := statusOk
		if err != nil {
			status = statusFailed
			backupResult.Error = err
		}

		backupResult.Status = status

		results <- backupResult
	}
}

func NewAzureDevOpsHost(input NewAzureDevOpsHostInput) (*AzureDevOpsHost, error) {
	setLoggerPrefix(input.Caller)

	switch {
	case input.BackupDir == "":
		return nil, errors.New("backup directory not specified")
	case input.UserName == "":
		return nil, errors.New("username not specified")
	case input.PAT == "":
		return nil, errors.New("personal access token not specified")
	case len(input.Orgs) == 0:
		return nil, errors.New("no organizations specified")
	}

	diffRemoteMethod, err := getDiffRemoteMethod(input.DiffRemoteMethod)
	if err != nil {
		return nil, err
	}

	if diffRemoteMethod == "" {
		logger.Printf("%s: %s", sUsingDefaultDiffRemoteMethod, defaultRemoteMethod)
		diffRemoteMethod = defaultRemoteMethod
	} else {
		logger.Printf("%s: %s", sUsingDiffRemoteMethod, diffRemoteMethod)
	}

	httpClient := input.HTTPClient
	if httpClient == nil {
		httpClient = getHTTPClient()
	}

	return &AzureDevOpsHost{
		Caller:           input.Caller,
		HttpClient:       httpClient,
		Provider:         AzureDevOpsProviderName,
		PAT:              input.PAT,
		Orgs:             input.Orgs,
		UserName:         input.UserName,
		DiffRemoteMethod: diffRemoteMethod,
		BackupDir:        input.BackupDir,
		BackupsToRetain:  input.BackupsToRetain,
		LogLevel:         input.LogLevel,
	}, nil
}

func (ad *AzureDevOpsHost) describeRepos() (describeReposOutput, errors.E) {
	var repos []repository

	var org string

	switch len(ad.Orgs) {
	case 0:
		return describeReposOutput{}, errors.New("no organizations specified")
	case 1:
		org = ad.Orgs[0]
	default:
		log.Printf("multiple organizations not currently supported. using first: %s", ad.Orgs[0])

		org = ad.Orgs[0]
	}

	// append repos belonging to any orgs specified
	logger.Printf("listing Azure DevOps organization %s's repositories", org)

	orgRepos, err := ad.describeAzureDevOpsOrgsRepos(org)
	if err != nil {
		logger.Printf("failed to get Azure DevOps organization %s repos", org)

		return describeReposOutput{}, errors.Wrapf(err, "failed to get Azure DevOps organization %s repos", org)
	}

	if len(orgRepos) == 0 {
		logger.Printf("no repos found for organization: %s", org)

		return describeReposOutput{}, nil
	}

	repos = append(repos, orgRepos...)

	return describeReposOutput{
		Repos: repos,
	}, nil
}

type NewAzureDevOpsHostInput struct {
	HTTPClient       *retryablehttp.Client
	Caller           string
	BackupDir        string
	DiffRemoteMethod string
	UserName         string
	PAT              string
	Orgs             []string
	BackupsToRetain  int
	LogLevel         int
}

type AzureDevOpsHost struct {
	Caller           string
	HttpClient       *retryablehttp.Client
	Provider         string
	PAT              string
	Orgs             []string
	UserName         string
	DiffRemoteMethod string
	BackupDir        string
	BackupsToRetain  int
	LogLevel         int
}

func AddBasicAuthToURL(originalURL, username, password string) (string, error) {
	parsedURL, err := url.Parse(originalURL)
	if err != nil {
		return "", fmt.Errorf("error parsing URL: %w", err)
	}

	parsedURL.User = url.UserPassword(username, password)

	return parsedURL.String(), nil
}

func (ad *AzureDevOpsHost) describeAzureDevOpsOrgsRepos(org string) ([]repository, errors.E) {
	if org == "" {
		return nil, errors.New("organization not specified")
	}

	organizationUrl := fmt.Sprintf("https://%s/%s", azureDevOpsDomain, org)

	basicAuth := generateBasicAuth(ad.UserName, ad.PAT)

	connection := azuredevops.NewPatConnection(organizationUrl, ad.PAT)

	ctx := context.Background()

	coreClient, err := azdevopscore.NewClient(ctx, connection)
	if err != nil {
		return nil, errors.Errorf("failed to create Azure DevOps core client: %s", err)
	}

	projects, err := listProjects(ctx, coreClient)
	if err != nil {
		return nil, errors.Errorf("failed to list projects: %s", err)
	}

	var allRepos []AzureDevOpsRepo

	for _, project := range projects {
		logger.Printf("listing Azure DevOps organization %s's project %s repositories", org, *project.Name)

		var projectRepos []AzureDevOpsRepo

		projectRepos, err = ListAllRepositories(ad.HttpClient, basicAuth, *project.Name, org)
		if err != nil {
			return nil, errors.Errorf("failed to list repositories for organization: %s project: %s - %s", org, *project.Name, err)
		}

		if len(projectRepos) == 0 || projectRepos[0].Name == "" {
			log.Printf("No repositories found for project: %v", *project.Name)

			continue
		}

		allRepos = append(allRepos, projectRepos...)
	}

	var gRepos []repository

	for _, repo := range allRepos {
		var cloneURL string

		cloneURL, err = AddBasicAuthToURL(repo.WebUrl, ad.UserName, ad.PAT)
		if err != nil {
			return nil, errors.Errorf("failed to add basic auth to URL: %s - %s", repo.WebUrl, err)
		}

		gRepos = append(gRepos, repository{
			Name:              repo.Name,
			Owner:             org,
			PathWithNameSpace: org + "/" + repo.Project.Name + "/" + repo.Name,
			Domain:            azureDevOpsDomain,
			HTTPSUrl:          repo.RemoteUrl,
			URLWithToken:      cloneURL,
		})
	}

	return gRepos, nil
}

func listProjects(ctx context.Context, cClient azdevopscore.Client) ([]azdevopscore.TeamProjectReference, error) {
	var projects []azdevopscore.TeamProjectReference

	var continuationToken *int

	for {
		responseValue, err := cClient.GetProjects(ctx,
			azdevopscore.GetProjectsArgs{ContinuationToken: continuationToken})
		if err != nil {
			return nil, fmt.Errorf("failed to get projects: %w", err)
		}

		projects = append(projects, (responseValue).Value...)

		if responseValue.ContinuationToken == "" {
			break
		}

		continuationTokenValue, err := strconv.Atoi(responseValue.ContinuationToken)
		if err != nil {
			return nil, fmt.Errorf("failed to convert continuation token to int: %w", err)
		}

		continuationToken = &continuationTokenValue
	}

	return projects, nil
}

type AzureDevOpsRepo struct {
	Id            string  `json:"id"`
	Url           string  `json:"url"`
	Name          string  `json:"name"`
	Size          int64   `json:"size"`
	SshUrl        string  `json:"sshUrl"`
	WebUrl        string  `json:"webUrl"`
	Project       Project `json:"project"`
	RemoteUrl     string  `json:"remoteUrl"`
	DefaultBranch string  `json:"defaultBranch"`
}

type Project struct {
	Id             string    `json:"id"`
	Url            string    `json:"url"`
	Name           string    `json:"name"`
	State          string    `json:"state"`
	Revision       int       `json:"revision"`
	Visibility     string    `json:"visibility"`
	Description    string    `json:"description"`
	LastUpdateTime time.Time `json:"lastUpdateTime"`
}

type repoListBody struct {
	Value []AzureDevOpsRepo `json:"value"`
}

func generateBasicAuth(userName string, pat string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", userName, pat)))
}

func ListAllRepositories(httpClient *retryablehttp.Client, basicAuth, projectName, orgName string) ([]AzureDevOpsRepo, error) {
	req, err := retryablehttp.NewRequest(http.MethodGet,
		fmt.Sprintf("https://%s/%s/%s/_apis/git/repositories", azureDevOpsDomain, orgName, projectName), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Basic "+basicAuth)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			return
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	r := &repoListBody{}

	if err = json.Unmarshal(body, r); err != nil {
		return nil, fmt.Errorf("failed to unmarshall json: %w", err)
	}

	return r.Value, nil
}
