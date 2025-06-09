package githosts

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/hashicorp/go-retryablehttp"
	"gitlab.com/tozd/go/errors"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	gitHubCallSize           = 100
	githubEnvVarCallSize     = "GITHUB_CALL_SIZE"
	githubEnvVarWorkerDelay  = "GITHUB_WORKER_DELAY"
	gitHubDomain             = "github.com"
	gitHubProviderName       = "GitHub"
	githubDefaultWorkerDelay = 500
)

type NewGitHubHostInput struct {
	HTTPClient       *retryablehttp.Client
	Caller           string
	APIURL           string
	DiffRemoteMethod string
	BackupDir        string
	Token            string
	LimitUserOwned   bool
	SkipUserRepos    bool
	Orgs             []string
	BackupsToRetain  int
	LogLevel         int
}

func (gh *GitHubHost) getAPIURL() string {
	return gh.APIURL
}

func NewGitHubHost(input NewGitHubHostInput) (*GitHubHost, error) {
	setLoggerPrefix(input.Caller)

	apiURL := githubAPIURL
	if input.APIURL != "" {
		apiURL = input.APIURL
	}

	diffRemoteMethod, err := getDiffRemoteMethod(input.DiffRemoteMethod)
	if err != nil {
		return nil, err
	}

	if diffRemoteMethod == "" {
		logger.Print("using default diff remote method: " + defaultRemoteMethod)
		diffRemoteMethod = defaultRemoteMethod
	} else {
		logger.Print("using diff remote method: " + diffRemoteMethod)
	}

	httpClient := input.HTTPClient
	if httpClient == nil {
		httpClient = getHTTPClient()
	}

	return &GitHubHost{
		Caller:           input.Caller,
		HttpClient:       httpClient,
		Provider:         gitHubProviderName,
		APIURL:           apiURL,
		DiffRemoteMethod: diffRemoteMethod,
		BackupDir:        input.BackupDir,
		SkipUserRepos:    input.SkipUserRepos,
		LimitUserOwned:   input.LimitUserOwned,
		BackupsToRetain:  input.BackupsToRetain,
		Token:            input.Token,
		Orgs:             input.Orgs,
		LogLevel:         input.LogLevel,
	}, nil
}

type GitHubHost struct {
	Caller           string
	HttpClient       *retryablehttp.Client
	Provider         string
	APIURL           string
	DiffRemoteMethod string
	BackupDir        string
	SkipUserRepos    bool
	LimitUserOwned   bool
	BackupsToRetain  int
	Token            string
	Orgs             []string
	LogLevel         int
}

type edge struct {
	Node struct {
		Name          string
		NameWithOwner string
		URL           string `json:"Url"`
		SSHURL        string `json:"sshUrl"`
	}
	Cursor string
}

type githubQueryNamesResponse struct {
	Data struct {
		Viewer struct {
			Repositories struct {
				Edges    []edge
				PageInfo struct {
					EndCursor   string
					HasNextPage bool
				}
			}
		}
	} `json:"data"`
}

type githubQueryOrgsResponse struct {
	Data struct {
		Viewer struct {
			Organizations struct {
				Edges    []orgsEdge
				PageInfo struct {
					EndCursor   string
					HasNextPage bool
				}
			}
		}
	}
	Errors []struct {
		Type    string
		Path    []string
		Message string
	}
}
type orgsEdge struct {
	Node struct {
		Name string
	}
	Cursor string
}

type githubQueryOrgResponse struct {
	Data struct {
		Organization struct {
			Repositories struct {
				Edges    []edge
				PageInfo struct {
					EndCursor   string
					HasNextPage bool
				}
			}
		}
	}
	Errors []struct {
		Type    string
		Path    []string
		Message string
	}
}

type graphQLRequest struct {
	Query     string `json:"query"`
	Variables string `json:"variables"`
}

func (gh *GitHubHost) makeGithubRequest(payload string) (string, errors.E) {
	contentReader := bytes.NewReader([]byte(payload))

	ctx, cancel := context.WithTimeout(context.Background(), defaultHttpRequestTimeout)
	defer cancel()

	req, newReqErr := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", contentReader)

	if newReqErr != nil {
		logger.Println(newReqErr)

		return "", errors.Wrap(newReqErr, "failed to create request")
	}

	req.Header.Set("Authorization", "bearer "+gh.Token)
	req.Header.Set("Content-Type", contentTypeApplicationJSON)
	req.Header.Set("Accept", contentTypeApplicationJSON)

	resp, reqErr := gh.HttpClient.Do(req)
	if reqErr != nil {
		logger.Print(reqErr)

		return "", errors.Wrap(reqErr, "failed to make request")
	}

	bodyB, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Print(err)

		return "", errors.Wrap(err, "failed to read response body")
	}

	defer resp.Body.Close()

	bodyStr := string(bytes.ReplaceAll(bodyB, []byte("\r"), []byte("\r\n")))

	// check response for errors
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		if strings.Contains(bodyStr, "Personal access tokens with fine grained access do not support the GraphQL API") {
			logger.Println("GitHub authorisation with fine grained PAT (Personal Access Token) failed as their GraphQL endpoint currently only supports classic PATs: https://github.blog/2022-10-18-introducing-fine-grained-personal-access-tokens-for-github/#coming-next")

			return "", errors.New("GitHub authorisation with fine grained PAT (Personal Access Token) failed as their GraphQL endpoint currently only supports classic PATs: https://github.blog/2022-10-18-introducing-fine-grained-personal-access-tokens-for-github/#coming-next")
		}

		logger.Printf("GitHub authorisation failed: %s", bodyStr)

		return "", errors.Errorf("GitHub authorisation failed: %s", bodyStr)
	case http.StatusOK:
		// authorisation successful
	default:
		return "", errors.New("GitHub authorisation failed")
	}

	return bodyStr, nil
}

// describeGithubUserRepos returns a list of repositories owned by authenticated user.
func (gh *GitHubHost) describeGithubUserRepos() ([]repository, errors.E) {
	logger.Println("listing GitHub user's owned repositories")

	gcs := gitHubCallSize

	envCallSize := os.Getenv(githubEnvVarCallSize)
	if envCallSize != "" {
		if callSize, err := strconv.Atoi(envCallSize); err != nil {
			gcs = callSize
		}
	}

	var repos []repository

	var reqBody string

	if gh.LimitUserOwned {
		reqBody = "{\"query\": \"query { viewer { repositories(first:" + strconv.Itoa(gcs) + ", affiliations: OWNER, ownerAffiliations: OWNER) { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }} } }\""
	} else {
		reqBody = "{\"query\": \"query { viewer { repositories(first:" + strconv.Itoa(gcs) + ") { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }} } }\""
	}

	for {
		bodyStr, err := gh.makeGithubRequest(reqBody)
		if err != nil {
			return nil, errors.Wrap(err, "GitHub request failed")
		}

		var respObj githubQueryNamesResponse
		if uErr := json.Unmarshal([]byte(bodyStr), &respObj); uErr != nil {
			logger.Print(uErr)

			return nil, errors.Wrap(uErr, "failed to unmarshal response")
		}

		for _, repo := range respObj.Data.Viewer.Repositories.Edges {
			repos = append(repos, repository{
				Name:              repo.Node.Name,
				SSHUrl:            repo.Node.SSHURL,
				HTTPSUrl:          repo.Node.URL,
				PathWithNameSpace: repo.Node.NameWithOwner,
				Domain:            gitHubDomain,
			})
		}

		if !respObj.Data.Viewer.Repositories.PageInfo.HasNextPage {
			break
		} else {
			if gh.LimitUserOwned {
				reqBody = "{\"query\": \"query($first:Int $after:String){ viewer { repositories(first:$first after:$after, affiliations: OWNER, ownerAffiliations: OWNER) { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }} } }\", \"variables\":{\"first\":" + strconv.Itoa(gcs) + ",\"after\":\"" + respObj.Data.Viewer.Repositories.PageInfo.EndCursor + "\"} }"
			} else {
				reqBody = "{\"query\": \"query($first:Int $after:String){ viewer { repositories(first:$first after:$after) { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }} } }\", \"variables\":{\"first\":" + strconv.Itoa(gcs) + ",\"after\":\"" + respObj.Data.Viewer.Repositories.PageInfo.EndCursor + "\"} }"
			}
		}
	}

	return repos, nil
}

func (gh *GitHubHost) describeGithubUserOrganizations() ([]githubOrganization, errors.E) {
	logger.Println("listing GitHub user's related Organizations")

	var orgs []githubOrganization

	reqBody := "{\"query\": \"{ viewer { organizations(first:100) { edges { node { name } } } } }\""

	bodyStr, err := gh.makeGithubRequest(reqBody)
	if err != nil {
		logger.Print(err)

		return nil, errors.Wrap(err, "GitHub request failed")
	}

	var respObj githubQueryOrgsResponse
	if uErr := json.Unmarshal([]byte(bodyStr), &respObj); uErr != nil {
		logger.Print(uErr)

		return nil, errors.Wrap(uErr, "failed to unmarshal response")
	}

	if len(respObj.Errors) > 0 {
		for _, queryError := range respObj.Errors {
			logger.Printf("failed to retrieve organizations user's a member of: %s", queryError.Message)
		}

		return nil, errors.New("failed to retrieve organizations user's a member of")
	}

	for _, org := range respObj.Data.Viewer.Organizations.Edges {
		orgs = append(orgs, githubOrganization{
			Name: org.Node.Name,
		})
	}

	return orgs, nil
}

type githubOrganization struct {
	Name string `json:"name"`
}

func createGithubRequestPayload(body string) (string, errors.E) {
	gqlMarshalled, err := json.Marshal(graphQLRequest{Query: body})
	if err != nil {
		logger.Print(err)

		return "", errors.Wrap(err, "failed to marshal request")
	}

	return string(gqlMarshalled), nil
}

func (gh *GitHubHost) describeGithubOrgRepos(orgName string) ([]repository, errors.E) {
	logger.Printf("listing GitHub organization %s's repositories", orgName)

	gcs := gitHubCallSize

	envCallSize := os.Getenv(githubEnvVarCallSize)
	if envCallSize != "" {
		if callSize, err := strconv.Atoi(envCallSize); err != nil {
			gcs = callSize
		}
	}

	var repos []repository

	reqBody := "query { organization(login: \"" + orgName + "\") { repositories(first:" + strconv.Itoa(gcs) + ") { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }}}}"

	for {
		payload, err := createGithubRequestPayload(reqBody)
		if err != nil {
			logger.Print(err)

			return nil, errors.Wrap(err, "failed to create request payload")
		}

		bodyStr, err := gh.makeGithubRequest(payload)
		if err != nil {
			logger.Print(err)

			return nil, nil
		}

		var respObj githubQueryOrgResponse

		if uErr := json.Unmarshal([]byte(bodyStr), &respObj); err != nil {
			logger.Print(err)

			return nil, errors.Wrap(uErr, "failed to unmarshal response")
		}

		if respObj.Errors != nil {
			for _, gqlErr := range respObj.Errors {
				if gqlErr.Type == "NOT_FOUND" {
					logger.Printf("organization %s not found", orgName)

					return nil, errors.Errorf("organization %s not found", orgName)
				} else {
					logger.Printf("unexpected error: type: %s message: %s", gqlErr.Type, gqlErr.Message)

					return nil, errors.Errorf("unexpected error: type: %s message: %s", gqlErr.Type, gqlErr.Message)
				}
			}
		}

		for _, repo := range respObj.Data.Organization.Repositories.Edges {
			repos = append(repos, repository{
				Name:              repo.Node.Name,
				SSHUrl:            repo.Node.SSHURL,
				HTTPSUrl:          repo.Node.URL,
				PathWithNameSpace: repo.Node.NameWithOwner,
				Domain:            gitHubDomain,
			})
		}

		if !respObj.Data.Organization.Repositories.PageInfo.HasNextPage {
			break
		} else {
			reqBody = "query { organization(login: \"" + orgName + "\") { repositories(first:" + strconv.Itoa(gcs) + " after: \"" + respObj.Data.Organization.Repositories.PageInfo.EndCursor + "\") { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }}}}"
		}
	}

	return repos, nil
}

func (gh *GitHubHost) describeRepos() (describeReposOutput, errors.E) {
	var repos []repository

	if !gh.SkipUserRepos {
		// get authenticated user's owned repos
		var err errors.E

		repos, err = gh.describeGithubUserRepos()
		if err != nil {
			logger.Print("failed to get GitHub user repos")

			return describeReposOutput{}, err
		}
	}

	// set orgs repos to retrieve to those specified when client constructed
	orgs := gh.Orgs

	// if we get a wildcard, get all orgs user belongs to
	if slices.Contains(gh.Orgs, "*") {
		// delete the wildcard, leaving any existing specified orgs that may have been passed in
		orgs = remove(orgs, "*")
		// get a list of orgs the authenticated user belongs to
		githubOrgs, err := gh.describeGithubUserOrganizations()
		if err != nil {
			logger.Print("failed to get user's GitHub organizations")

			return describeReposOutput{}, err
		}

		for _, gho := range githubOrgs {
			orgs = append(orgs, gho.Name)
		}
	}

	// append repos belonging to any orgs specified
	for _, org := range orgs {
		dRepos, err := gh.describeGithubOrgRepos(org)
		if err != nil {
			logger.Printf("failed to get GitHub organization %s repos", org)

			return describeReposOutput{}, errors.Wrapf(err, "failed to get GitHub organization %s repos", org)
		}

		repos = append(repos, dRepos...)
	}

	// remove any duplicate repos
	// this can happen if the authenticated user is a member of an org and also has their own repos
	repos = removeDuplicates(repos)

	return describeReposOutput{
		Repos: repos,
	}, nil
}

func removeDuplicates(repos []repository) []repository {
	var uniqueRepos []repository

	keys := make(map[string]bool)

	for _, repo := range repos {
		if _, value := keys[repo.PathWithNameSpace]; !value {
			keys[repo.PathWithNameSpace] = true

			uniqueRepos = append(uniqueRepos, repo)
		}
	}

	return uniqueRepos
}

func gitHubWorker(logLevel int, token, backupDIR, diffRemoteMethod string, backupsToKeep int, jobs <-chan repository, results chan<- RepoBackupResults) {
	for repo := range jobs {
		repo.URLWithToken = urlWithToken(repo.HTTPSUrl, stripTrailing(token, "\n"))
		err := processBackup(logLevel, repo, backupDIR, backupsToKeep, diffRemoteMethod)
		results <- repoBackupResult(repo, err)
	}
}

func (gh *GitHubHost) Backup() ProviderBackupResult {
	if gh.BackupDir == "" {
		logger.Printf("backup skipped as backup directory not specified")

		return ProviderBackupResult{
			BackupResults: nil,
			Error:         errors.New("backup directory not specified"),
		}
	}

	maxConcurrent := 10

	repoDesc, err := gh.describeRepos()
	if err != nil {
		return ProviderBackupResult{
			BackupResults: nil,
			Error:         err,
		}
	}

	jobs := make(chan repository, len(repoDesc.Repos))
	results := make(chan RepoBackupResults, maxConcurrent)

	for w := 1; w <= maxConcurrent; w++ {
		go gitHubWorker(gh.LogLevel, gh.Token, gh.BackupDir, gh.DiffRemoteMethod, gh.BackupsToRetain, jobs, results)

		delay := githubDefaultWorkerDelay
		if envDelay, sErr := strconv.Atoi(os.Getenv(githubEnvVarWorkerDelay)); sErr == nil {
			delay = envDelay
		}

		time.Sleep(time.Duration(delay) * time.Millisecond)
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
			logger.Printf("backup failed: %+v\n", errors.Unwrap(res.Error))
		}

		providerBackupResults.BackupResults = append(providerBackupResults.BackupResults, res)
	}

	return providerBackupResults
}

// return normalised method.
func (gh *GitHubHost) diffRemoteMethod() string {
	if gh.DiffRemoteMethod == "" {
		logger.Printf("diff remote method not specified. defaulting to:%s", cloneMethod)
	}

	return canonicalDiffRemoteMethod(gh.DiffRemoteMethod)
}
