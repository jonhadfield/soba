package githosts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/tozd/go/errors"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/peterhellberg/link"
)

const (
	// GitLabDefaultMinimumProjectAccessLevel https://docs.gitlab.com/ee/user/permissions.html#roles
	GitLabDefaultMinimumProjectAccessLevel = 20
	gitLabDomain                           = "gitlab.com"
)

type gitlabUser struct {
	ID       int    `json:"id"`
	UserName string `json:"username"`
}

type GitLabHost struct {
	Caller                string
	httpClient            *retryablehttp.Client
	APIURL                string
	DiffRemoteMethod      string
	BackupDir             string
	BackupsToRetain       int
	ProjectMinAccessLevel int
	Token                 string
	User                  gitlabUser
	LogLevel              int
}

func (gl *GitLabHost) getAuthenticatedGitLabUser() (gitlabUser, errors.E) {
	gitlabToken := strings.TrimSpace(gl.Token)
	if gitlabToken == "" {
		return gitlabUser{}, errors.New("GitLab token not provided")
	}

	var err error

	// use default if not passed
	if gl.APIURL == "" {
		gl.APIURL = gitlabAPIURL
	}

	getUserIDURL := gl.APIURL + "/user"

	ctx, cancel := context.WithTimeout(context.Background(), defaultHttpRequestTimeout)
	defer cancel()

	var req *retryablehttp.Request

	req, err = retryablehttp.NewRequestWithContext(ctx, http.MethodGet, getUserIDURL, nil)
	if err != nil {
		return gitlabUser{}, errors.Errorf("failed to create request: %s", err)
	}

	req.Header.Set("Private-Token", gl.Token)
	req.Header.Set("Content-Type", contentTypeApplicationJSON)
	req.Header.Set("Accept", contentTypeApplicationJSON)

	var resp *http.Response

	resp, err = gl.httpClient.Do(req)
	if err != nil {
		return gitlabUser{}, errors.Errorf("request failed: %s", err)
	}

	bodyB, err := io.ReadAll(resp.Body)
	if err != nil {
		return gitlabUser{
			UserName: "",
		}, nil
	}

	bodyStr := string(bytes.ReplaceAll(bodyB, []byte("\r"), []byte("\r\n")))

	_ = resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if gl.LogLevel > 0 {
			logger.Println("authentication successful")
		}
	case http.StatusForbidden:
		logger.Println("failed to authenticate (HTTP 403)")
	case http.StatusUnauthorized:
		logger.Println("failed to authenticate due to invalid credentials (HTTP 401)")
	default:
		logger.Printf("failed to authenticate due to unexpected response: %d (%s)", resp.StatusCode, resp.Status)

		return gitlabUser{}, nil
	}

	var user gitlabUser

	if err = json.Unmarshal([]byte(bodyStr), &user); err != nil {
		return gitlabUser{}, errors.Errorf("failed to unmarshall gitlab json response: %s", err.Error())
	}

	return user, nil
}

type gitLabOwner struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type gitLabProject struct {
	Path              string      `json:"path"`
	PathWithNameSpace string      `json:"path_with_namespace"`
	HTTPSURL          string      `json:"http_url_to_repo"`
	SSHURL            string      `json:"ssh_url_to_repo"`
	Owner             gitLabOwner `json:"owner"`
}
type gitLabGetProjectsResponse []gitLabProject

var validAccessLevels = map[int]string{
	20: "Reporter",
	30: "Developer",
	40: "Maintainer",
	50: "Owner",
}

func (gl *GitLabHost) getAllProjectRepositories(client http.Client) ([]repository, errors.E) {
	var sortedLevels []int
	for k := range validAccessLevels {
		sortedLevels = append(sortedLevels, k)
	}

	sort.Ints(sortedLevels)

	var validMinimumProjectAccessLevels []string

	for _, level := range sortedLevels {
		validMinimumProjectAccessLevels = append(validMinimumProjectAccessLevels, fmt.Sprintf("%s (%d)", validAccessLevels[level], level))
	}

	logger.Printf("retrieving all projects for user %s (%d):", gl.User.UserName, gl.User.ID)

	if strings.TrimSpace(gl.APIURL) == "" {
		gl.APIURL = gitlabAPIURL
	}

	getProjectsURL := gl.APIURL + "/projects"

	if gl.ProjectMinAccessLevel == 0 {
		gl.ProjectMinAccessLevel = GitLabDefaultMinimumProjectAccessLevel
	}

	if !slices.Contains(sortedLevels, gl.ProjectMinAccessLevel) {
		logger.Printf("project minimum access level must be one of %s so using default %d",
			strings.Join(validMinimumProjectAccessLevels, ", "), GitLabDefaultMinimumProjectAccessLevel)

		gl.ProjectMinAccessLevel = GitLabDefaultMinimumProjectAccessLevel
	}

	logger.Printf("project minimum access level set to %s (%d)",
		validAccessLevels[gl.ProjectMinAccessLevel],
		gl.ProjectMinAccessLevel)

	// Initial request
	u, err := url.Parse(getProjectsURL)
	if err != nil {
		logger.Print(err)

		return []repository{}, errors.Wrap(err, "failed to parse url")
	}

	q := u.Query()
	// set initial max per page
	q.Set("per_page", strconv.Itoa(gitlabProjectsPerPageDefault))
	q.Set("min_access_level", strconv.Itoa(gl.ProjectMinAccessLevel))
	u.RawQuery = q.Encode()

	var body []byte

	reqUrl := u.String()

	var repos []repository

	for {
		var resp *http.Response

		var rErr errors.E

		resp, body, rErr = makeGitLabRequest(&client, reqUrl, gl.Token)
		if rErr != nil {
			logger.Print(rErr)

			return []repository{}, rErr
		}

		if gl.LogLevel > 0 {
			logger.Println(string(body))
		}

		switch resp.StatusCode {
		case http.StatusOK:
			if gl.LogLevel > 0 {
				logger.Println("projects retrieved successfully")
			}
		case http.StatusForbidden:
			logger.Println("failed to get projects due to invalid missing permissions (HTTP 403)")

			return []repository{}, errors.New("failed to get projects due to invalid missing permissions (HTTP 403)")
		default:
			logger.Printf("failed to get projects due to unexpected response: %d (%s)", resp.StatusCode, resp.Status)

			return []repository{}, errors.Errorf("failed to get projects due to unexpected response: %d (%s)", resp.StatusCode, resp.Status)
		}

		var respObj gitLabGetProjectsResponse

		if err = json.Unmarshal(body, &respObj); err != nil {
			logger.Println(err)

			return []repository{}, errors.Errorf("failed to unmarshall gitlab json response: %s", err.Error())
		}

		for _, project := range respObj {
			// gitlab replaces hyphens with spaces in owner names, so fix
			owner := strings.ReplaceAll(project.Owner.Name, " ", "-")
			repo := repository{
				Name:              project.Path,
				Owner:             owner,
				PathWithNameSpace: project.PathWithNameSpace,
				HTTPSUrl:          project.HTTPSURL,
				SSHUrl:            project.SSHURL,
				Domain:            gitLabDomain,
			}

			repos = append(repos, repo)
		}

		// if we got a link response then
		// reset request url
		reqUrl = ""

		for _, l := range link.ParseResponse(resp) {
			if l.Rel == txtNext {
				reqUrl = l.URI
			}
		}

		if reqUrl == "" {
			break
		}
	}

	return repos, nil
}

func makeGitLabRequest(c *http.Client, reqUrl, token string) (*http.Response, []byte, errors.E) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHttpRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, nil, errors.Errorf("failed to request %s: %s", reqUrl, err.Error())
	}

	req.Header.Set("Private-Token", token)
	req.Header.Set("Content-Type", contentTypeApplicationJSON)
	req.Header.Set("Accept", contentTypeApplicationJSON)

	resp, err := c.Do(req)
	if err != nil {
		return nil, nil, errors.Errorf("request failed: %s", err.Error())
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Errorf("failed to read response body: %s", err.Error())
	}

	body = bytes.ReplaceAll(body, []byte("\r"), []byte("\r\n"))

	_ = resp.Body.Close()

	return resp, body, nil
}

type NewGitLabHostInput struct {
	Caller                string
	HTTPClient            *retryablehttp.Client
	APIURL                string
	DiffRemoteMethod      string
	BackupDir             string
	Token                 string
	ProjectMinAccessLevel int
	BackupsToRetain       int
	LogLevel              int
}

func NewGitLabHost(input NewGitLabHostInput) (*GitLabHost, error) {
	setLoggerPrefix(input.Caller)

	apiURL := gitlabAPIURL
	if input.APIURL != "" {
		apiURL = input.APIURL
	}

	diffRemoteMethod, err := getDiffRemoteMethod(input.DiffRemoteMethod)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff remote method: %w", err)
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

	return &GitLabHost{
		Caller:                input.Caller,
		httpClient:            httpClient,
		APIURL:                apiURL,
		DiffRemoteMethod:      diffRemoteMethod,
		BackupDir:             input.BackupDir,
		BackupsToRetain:       input.BackupsToRetain,
		Token:                 input.Token,
		ProjectMinAccessLevel: input.ProjectMinAccessLevel,
		LogLevel:              input.LogLevel,
	}, nil
}

func (gl *GitLabHost) describeRepos() (describeReposOutput, errors.E) {
	logger.Println("listing repositories")

	tr := &http.Transport{
		MaxIdleConns:       maxIdleConns,
		IdleConnTimeout:    idleConnTimeout,
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	userRepos, err := gl.getAllProjectRepositories(*client)
	if err != nil {
		return describeReposOutput{}, err
	}

	return describeReposOutput{
		Repos: userRepos,
	}, nil
}

func (gl *GitLabHost) getAPIURL() string {
	return gl.APIURL
}

func gitlabWorker(logLevel int, userName, token, backupDIR, diffRemoteMethod string, backupsToKeep int, jobs <-chan repository, results chan<- RepoBackupResults) {
	for repo := range jobs {
		firstPos := strings.Index(repo.HTTPSUrl, "//")
		repo.URLWithToken = repo.HTTPSUrl[:firstPos+2] + userName + ":" + stripTrailing(token, "\n") + "@" + repo.HTTPSUrl[firstPos+2:]
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

func (gl *GitLabHost) Backup() ProviderBackupResult {
	if gl.BackupDir == "" {
		logger.Printf("backup skipped as backup directory not specified")

		return ProviderBackupResult{}
	}

	maxConcurrent := 5

	var err errors.E

	gl.User, err = gl.getAuthenticatedGitLabUser()
	if err != nil {
		return ProviderBackupResult{
			BackupResults: nil,
			Error:         err,
		}
	}

	if gl.User.ID == 0 {
		// skip backup if user is not authenticated
		return ProviderBackupResult{}
	}

	repoDesc, err := gl.describeRepos()
	if err != nil {
		return ProviderBackupResult{
			Error: errors.Wrap(err, "failed to describe repos"),
		}
	}

	jobs := make(chan repository, len(repoDesc.Repos))
	results := make(chan RepoBackupResults, maxConcurrent)

	for w := 1; w <= maxConcurrent; w++ {
		go gitlabWorker(gl.LogLevel, gl.User.UserName, gl.Token, gl.BackupDir, gl.diffRemoteMethod(), gl.BackupsToRetain, jobs, results)
	}

	var providerBackupResults ProviderBackupResult

	for x := range repoDesc.Repos {
		repo := repoDesc.Repos[x]
		jobs <- repo
	}

	close(jobs)

	for a := 1; a <= len(repoDesc.Repos); a++ {
		res := <-results
		if res.Error != nil {
			logger.Printf("backup failed: %+v\n", res.Error)
		}

		providerBackupResults.BackupResults = append(providerBackupResults.BackupResults, res)
	}

	return providerBackupResults
}

// return normalised method.
func (gl *GitLabHost) diffRemoteMethod() string {
	switch strings.ToLower(gl.DiffRemoteMethod) {
	case refsMethod:
		return refsMethod
	case cloneMethod:
		return cloneMethod
	default:
		logger.Printf("unexpected diff remote method: %s", gl.DiffRemoteMethod)

		// default to bundle as safest
		return cloneMethod
	}
}
