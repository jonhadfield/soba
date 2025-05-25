package githosts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gitlab.com/tozd/go/errors"

	"github.com/hashicorp/go-retryablehttp"
)

const (
	BitbucketProviderName = "BitBucket"
	bitbucketEnvVarKey    = "BITBUCKET_KEY"
	bitbucketEnvVarSecret = "BITBUCKET_SECRET"
	bitbucketEnvVarUser   = "BITBUCKET_USER"
	bitbucketDomain       = "bitbucket.com"
)

type NewBitBucketHostInput struct {
	Caller           string
	HTTPClient       *retryablehttp.Client
	APIURL           string
	DiffRemoteMethod string
	BackupDir        string
	User             string
	Key              string
	Secret           string
	BackupsToRetain  int
	LogLevel         int
}

func NewBitBucketHost(input NewBitBucketHostInput) (*BitbucketHost, error) {
	setLoggerPrefix(input.Caller)

	apiURL := bitbucketAPIURL
	if input.APIURL != "" {
		apiURL = input.APIURL
	}

	diffRemoteMethod, err := getDiffRemoteMethod(input.DiffRemoteMethod)
	if err != nil {
		return nil, errors.Errorf("failed to get diff remote method: %s", err)
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

	return &BitbucketHost{
		HttpClient:       httpClient,
		Provider:         BitbucketProviderName,
		APIURL:           apiURL,
		DiffRemoteMethod: diffRemoteMethod,
		BackupDir:        input.BackupDir,
		BackupsToRetain:  input.BackupsToRetain,
		User:             input.User,
		Key:              input.Key,
		Secret:           input.Secret,
	}, nil
}

func (bb BitbucketHost) auth(key, secret string) (string, error) {
	b, _, _, err := httpRequest(httpRequestInput{
		client: bb.HttpClient,
		url:    fmt.Sprintf("https://%s:%s@bitbucket.org/site/oauth2/access_token", key, secret),
		method: http.MethodPost,
		headers: http.Header{
			"Host":         []string{"bitbucket.org"},
			"Content-Type": []string{"application/x-www-form-urlencoded"},
			"Accept":       []string{"*/*"},
		},
		reqBody:           []byte("grant_type=client_credentials"),
		basicAuthUser:     key,
		basicAuthPassword: secret,
		secrets:           []string{key, secret},
		timeout:           defaultHttpRequestTimeout,
	})
	if err != nil {
		return "", errors.Errorf("failed to get bitbucket auth token: %s", err)
	}

	bodyStr := string(bytes.ReplaceAll(b, []byte("\r"), []byte("\r\n")))

	var authResp bitbucketAuthResponse

	if err = json.Unmarshal([]byte(bodyStr), &authResp); err != nil {
		return "", errors.Errorf("failed to unmarshall bitbucket json response: %s", err)
	}

	// check for any errors
	if authResp.AccessToken == "" {
		var authErrResp bitbucketAuthErrorResponse

		if err = json.Unmarshal([]byte(bodyStr), &authErrResp); err != nil {
			return "", errors.Errorf("failed to unmarshall bitbucket json error response: %s", err)
		}

		return "", errors.Errorf("failed to get bitbucket auth token: %s - %s", authErrResp.Error, authErrResp.ErrorDescription)
	}

	return authResp.AccessToken, nil
}

type bitbucketAuthResponse struct {
	AccessToken  string `json:"access_token"`
	Scopes       string `json:"scopes"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type bitbucketAuthErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (bb BitbucketHost) describeRepos() (describeReposOutput, errors.E) {
	logger.Println("listing BitBucket repositories")

	var err error

	key := os.Getenv(bitbucketEnvVarKey)
	secret := os.Getenv(bitbucketEnvVarSecret)

	var token string

	token, err = bb.auth(key, secret)
	if err != nil {
		return describeReposOutput{}, errors.Wrap(err, "failed to get bitbucket auth token")
	}

	var repos []repository

	rawRequestURL := bb.APIURL + "/repositories?role=member"

	ctx, cancel := context.WithTimeout(context.Background(), defaultHttpRequestTimeout)
	defer cancel()

	for {
		req, errNewReq := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, rawRequestURL, nil)
		if errNewReq != nil {
			logger.Println(errNewReq)

			return describeReposOutput{}, errors.Wrap(errNewReq, "failed to create new request")
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", contentTypeApplicationJSON)
		req.Header.Set("Accept", contentTypeApplicationJSON)

		var resp *http.Response

		resp, err = bb.HttpClient.Do(req)
		if err != nil {
			logger.Println(err)

			return describeReposOutput{}, errors.Wrap(err, "failed to make request")
		}

		var bodyB []byte

		bodyB, err = io.ReadAll(resp.Body)
		if err != nil {
			return describeReposOutput{}, errors.Errorf("failed to read response body: %s", err)
		}

		bodyStr := string(bytes.ReplaceAll(bodyB, []byte("\r"), []byte("\r\n")))

		_ = resp.Body.Close()

		var respObj bitbucketGetProjectsResponse
		if err = json.Unmarshal([]byte(bodyStr), &respObj); err != nil {
			logger.Println(err)

			return describeReposOutput{}, errors.Wrap(err, "failed to unmarshall bitbucket json response")
		}

		for _, r := range respObj.Values {
			if r.Scm == "git" {
				repo := repository{
					Name:              r.Name,
					HTTPSUrl:          "https://bitbucket.org/" + r.FullName + ".git",
					PathWithNameSpace: r.FullName,
					Domain:            bitbucketDomain,
				}

				repos = append(repos, repo)
			}
		}

		if respObj.Next != "" {
			rawRequestURL = respObj.Next

			continue
		}

		break
	}

	return describeReposOutput{
		Repos: repos,
	}, nil
}

func (bb BitbucketHost) getAPIURL() string {
	return bb.APIURL
}

func bitBucketWorker(logLevel int, user, token, backupDIR, diffRemoteMethod string, backupsToKeep int, jobs <-chan repository, results chan<- RepoBackupResults) {
	for repo := range jobs {
		repo.URLWithBasicAuth = urlWithBasicAuth(repo.HTTPSUrl, user, token)
		err := processBackup(logLevel, repo, backupDIR, backupsToKeep, diffRemoteMethod)
		results <- repoBackupResult(repo, err)
	}
}

func (bb BitbucketHost) Backup() ProviderBackupResult {
	if bb.BackupDir == "" {
		logger.Printf("backup skipped as backup directory not specified")

		return ProviderBackupResult{}
	}

	maxConcurrent := 5

	var err error

	var token string

	token, err = bb.auth(bb.Key, bb.Secret)
	if err != nil {
		return ProviderBackupResult{
			Error: errors.Errorf("failed to get bitbucket auth token: %s", err),
		}
	}

	drO, err := bb.describeRepos()
	if err != nil {
		return ProviderBackupResult{}
	}

	jobs := make(chan repository, len(drO.Repos))

	results := make(chan RepoBackupResults, maxConcurrent)

	for w := 1; w <= maxConcurrent; w++ {
		go bitBucketWorker(bb.LogLevel, bb.User, token, bb.BackupDir, bb.diffRemoteMethod(), bb.BackupsToRetain, jobs, results)
	}

	for x := range drO.Repos {
		repo := drO.Repos[x]
		jobs <- repo
	}

	close(jobs)

	var providerBackupResults ProviderBackupResult

	for a := 1; a <= len(drO.Repos); a++ {
		res := <-results
		if res.Error != nil {
			logger.Printf("backup failed: %+v\n", res.Error)

			providerBackupResults.Error = res.Error

			return providerBackupResults
		}

		providerBackupResults.BackupResults = append(providerBackupResults.BackupResults, res)
	}

	return providerBackupResults
}

type BitbucketHost struct {
	Caller           string
	HttpClient       *retryablehttp.Client
	Provider         string
	APIURL           string
	DiffRemoteMethod string
	BackupDir        string
	BackupsToRetain  int
	User             string
	Key              string
	Secret           string
	LogLevel         int
}

type bitbucketOwner struct {
	DisplayName string `json:"display_name"`
}

type bitbucketProject struct {
	Scm       string `json:"scm"`
	Owner     bitbucketOwner
	Name      string            `json:"name"`
	FullName  string            `json:"full_name"`
	IsPrivate bool              `json:"is_private"`
	Links     bitbucketRepoLink `json:"links"`
}

type bitbucketCloneDetail struct {
	Href string `json:"href"`
	Name string `json:"name"`
}

type bitbucketRepoLink struct {
	Clone []bitbucketCloneDetail `json:"clone"`
}

type bitbucketGetProjectsResponse struct {
	Pagelen int                `json:"pagelen"`
	Values  []bitbucketProject `json:"values"`
	Next    string             `json:"next"`
}

// return normalised method.
func (bb BitbucketHost) diffRemoteMethod() string {
	switch strings.ToLower(bb.DiffRemoteMethod) {
	case refsMethod:
		return refsMethod
	case cloneMethod:
		return cloneMethod
	case "":
		return cloneMethod
	default:
		logger.Printf("unexpected diff remote method: %s", bb.DiffRemoteMethod)

		// default to bundle as safest
		return cloneMethod
	}
}
