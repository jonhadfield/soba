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
	"strconv"
	"strings"
	"time"

	"gitlab.com/tozd/go/errors"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/peterhellberg/link"
)

const (
	giteaUsersPerPageDefault         = 20
	giteaUsersLimit                  = -1
	giteaOrganizationsPerPageDefault = 20
	giteaOrganizationsLimit          = -1
	giteaReposPerPageDefault         = 20
	giteaReposLimit                  = -1
	giteaEnvVarAPIUrl                = "GITEA_APIURL"
	giteaMatchByExact                = "exact"
	giteaMatchByIfDefined            = "anyDefined"
	giteaProviderName                = "Gitea"
	txtNext                          = "next"
)

type NewGiteaHostInput struct {
	Caller           string
	HTTPClient       *retryablehttp.Client
	APIURL           string
	DiffRemoteMethod string
	BackupDir        string
	Token            string
	Orgs             []string
	BackupsToRetain  int
	LogLevel         int
}

type GiteaHost struct {
	Caller           string
	httpClient       *retryablehttp.Client
	APIURL           string
	DiffRemoteMethod string
	BackupDir        string
	BackupsToRetain  int
	Token            string
	Orgs             []string
	LogLevel         int
}

func NewGiteaHost(input NewGiteaHostInput) (*GiteaHost, error) {
	setLoggerPrefix(input.Caller)

	if input.APIURL == "" {
		return nil, fmt.Errorf("%s API URL missing", giteaProviderName)
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

	return &GiteaHost{
		httpClient:       httpClient,
		APIURL:           input.APIURL,
		DiffRemoteMethod: diffRemoteMethod,
		BackupDir:        input.BackupDir,
		BackupsToRetain:  input.BackupsToRetain,
		Token:            input.Token,
		Orgs:             input.Orgs,
		LogLevel:         input.LogLevel,
	}, nil
}

type giteaUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	LoginName string `json:"login_name"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
}

type giteaOrganization struct {
	ID                       int    `json:"id"`
	Name                     string `json:"name"`
	FullName                 string `json:"full_name"`
	AvatarURL                string `json:"avatar_url"`
	Description              string `json:"description"`
	Website                  string `json:"website"`
	Location                 string `json:"location"`
	Visibility               string `json:"visibility"`
	RepoAdminChangeTeamAcces bool   `json:"repo_admin_change_team_access"`
	Username                 string `json:"username"`
}

type (
	giteaGetUsersResponse         []giteaUser
	giteaGetOrganizationsResponse []giteaOrganization
)

func (g *GiteaHost) makeGiteaRequest(reqUrl string) (*http.Response, []byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultHttpRequestTimeout)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to request %s: %w", reqUrl, err)
	}

	req.Header.Set("Authorization", "token "+g.Token)
	req.Header.Set("Content-Type", contentTypeApplicationJSON)
	req.Header.Set("Accept", contentTypeApplicationJSON)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to request %s: %w", reqUrl, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	body = bytes.ReplaceAll(body, []byte("\r"), []byte("\r\n"))

	_ = resp.Body.Close()

	return resp, body, err
}

type repoExistsInput struct {
	matchBy           string // anyDefined, allDefined, exact
	repos             []repository
	name              string
	owner             string
	pathWithNamespace string
	domain            string
	httpsUrl          string
	sshUrl            string
	urlWithToken      string
	urlWithBasicAuth  string
	logLevel          int
}

type userExistsInput struct {
	matchBy   string // anyDefined, allDefined, exact
	users     []giteaUser
	id        int
	login     string
	loginName string
	email     string
	fullName  string
}
type organizationExistsInput struct {
	matchBy       string // anyDefined, allDefined, exact
	organizations []giteaOrganization
	name          string
	fullName      string
}

func repoExists(in repoExistsInput) bool {
	switch in.matchBy {
	case giteaMatchByExact:
		if in.logLevel > 0 {
			logger.Printf("matchBy %s", giteaMatchByExact)
		}
	case giteaMatchByIfDefined:
		if in.logLevel > 0 {
			logger.Printf("matchBy %s", giteaMatchByExact)
		}
	case "":
		if in.logLevel > 0 {
			logger.Printf("matchBy not defined")
		}

		return false
	default:
		logger.Printf("unexpected matchBy value %s", in.matchBy)

		return false
	}

	if in.matchBy == "" {
		if in.logLevel > 0 {
			logger.Printf("matchBy not defined, defaulting to %s", giteaMatchByExact)
		}
	}

	if len(in.repos) == 0 {
		return false
	}

	for _, r := range in.repos {
		nameMatch := in.name == r.Name
		ownerMatch := in.owner == r.Owner
		domainMatch := in.domain == r.Domain
		cloneUrlMatch := in.httpsUrl == r.HTTPSUrl
		sshUrlMatch := in.sshUrl == r.SSHUrl
		urlWithTokenMatch := in.urlWithToken == r.URLWithToken
		urlWithBasicAuthMatch := in.urlWithBasicAuth == r.URLWithBasicAuth
		pathWithNamespaceMatch := in.pathWithNamespace == r.PathWithNameSpace

		switch in.matchBy {
		case giteaMatchByExact:
			if allTrue(nameMatch, domainMatch, ownerMatch, cloneUrlMatch, sshUrlMatch, urlWithTokenMatch,
				urlWithBasicAuthMatch, pathWithNamespaceMatch) {
				return true
			}

			continue
		case giteaMatchByIfDefined:
			anyDefined := in.name != "" || in.domain != "" || in.owner != "" || in.httpsUrl != "" || in.sshUrl != ""

			switch {
			case in.name != "" && !nameMatch:
				continue
			case in.domain != "" && !domainMatch:
				continue
			case in.owner != "" && !ownerMatch:
				continue
			case in.httpsUrl != "" && !cloneUrlMatch:
				continue
			case in.sshUrl != "" && !sshUrlMatch:
				continue
			case in.urlWithToken != "" && !urlWithTokenMatch:
				continue
			case in.urlWithBasicAuth != "" && !urlWithBasicAuthMatch:
				continue
			case in.pathWithNamespace != "" && !pathWithNamespaceMatch:
				continue
			default:
				if anyDefined {
					return true
				}

				continue
			}
		}
	}

	return false
}

func userExists(in userExistsInput) bool {
	for _, u := range in.users {
		loginMatch := in.login == u.Login
		idMatch := in.id == u.ID
		loginNameMatch := in.loginName == u.LoginName
		emailMatch := in.email == u.Email
		fullNameMatch := in.fullName == u.FullName

		switch in.matchBy {
		case giteaMatchByExact:
			if allTrue(loginMatch, loginNameMatch, idMatch, emailMatch, fullNameMatch) {
				return true
			}

			continue
		case giteaMatchByIfDefined:
			anyDefined := in.login != "" || in.id != 0 || in.loginName != "" || in.email != "" || in.fullName != ""

			switch {
			case in.login != "" && !loginMatch:
				continue
			case in.id != 0 && !idMatch:
				continue
			case in.loginName != "" && !loginNameMatch:
				continue
			case in.email != "" && !emailMatch:
				continue
			case in.fullName != "" && !fullNameMatch:
				continue
			default:
				if anyDefined {
					return true
				}

				continue
			}
		}
	}

	return false
}

func organisationExists(in organizationExistsInput) bool {
	for _, o := range in.organizations {
		nameMatch := in.name == o.Name
		fullNameMatch := in.fullName == o.FullName

		switch in.matchBy {
		case giteaMatchByExact:
			if allTrue(nameMatch, fullNameMatch) {
				return true
			}

			continue
		case giteaMatchByIfDefined:
			switch {
			case in.name != "" && !nameMatch:
				continue
			case in.fullName != "" && !fullNameMatch:
				continue
			}

			return true
		}
	}

	return false
}

func (g *GiteaHost) describeRepos() (describeReposOutput, errors.E) {
	logger.Println("listing repositories")

	userRepos, err := g.getAllUserRepositories()
	if err != nil {
		return describeReposOutput{}, errors.Errorf("failed to get user repositories: %s", err)
	}

	orgs, err := g.getOrganizations()
	if err != nil {
		return describeReposOutput{}, errors.Errorf("failed to get organizations: %s", err)
	}

	var orgsRepos []repository
	if len(orgs) > 0 {
		orgsRepos, err = g.getOrganizationsRepos(orgs)
		if err != nil {
			return describeReposOutput{}, errors.Errorf("failed to get organizations repos: %s", err)
		}
	}

	return describeReposOutput{
		Repos: append(userRepos, orgsRepos...),
	}, nil
}

func extractDomainFromAPIUrl(apiUrl string) string {
	u, err := url.Parse(apiUrl)
	if err != nil {
		logger.Printf("failed to parse apiUrl %s: %v", apiUrl, err)
	}

	return u.Hostname()
}

func (g *GiteaHost) getOrganizationsRepos(organizations []giteaOrganization) ([]repository, errors.E) {
	domain := extractDomainFromAPIUrl(g.APIURL)

	var repos []repository

	for _, org := range organizations {
		if g.LogLevel > 0 {
			logger.Printf("getting repositories from gitea organization %s", org.Name)
		}

		orgRepos, err := g.getOrganizationRepos(org.Name)
		if err != nil {
			return nil, errors.Errorf("failed to get organization %s repos: %s", org.Name, err)
		}

		for _, orgRepo := range orgRepos {
			repos = append(repos, repository{
				Name:              orgRepo.Name,
				Owner:             orgRepo.Owner.Login,
				HTTPSUrl:          orgRepo.CloneUrl,
				SSHUrl:            orgRepo.SshUrl,
				PathWithNameSpace: orgRepo.FullName,
				Domain:            domain,
			})
		}
	}

	return repos, nil
}

func (g *GiteaHost) getAllUsers() ([]giteaUser, errors.E) {
	if strings.TrimSpace(g.APIURL) == "" {
		g.APIURL = gitlabAPIURL
	}

	getUsersURL := g.APIURL + "/admin/users"
	if g.LogLevel > 0 {
		logger.Printf("get users url: %s", getUsersURL)
	}

	// Initial request
	u, err := url.Parse(getUsersURL)
	if err != nil {
		logger.Printf("failed to parse get users URL %s: %v", getUsersURL, err)

		return nil, errors.Wrap(err, "failed to parse get users URL")
	}

	q := u.Query()
	// set initial max per page
	q.Set("per_page", strconv.Itoa(giteaUsersPerPageDefault))
	q.Set("limit", strconv.Itoa(giteaUsersLimit))
	u.RawQuery = q.Encode()

	var body []byte

	reqUrl := u.String()

	var users []giteaUser

	for {
		var resp *http.Response

		resp, body, err = g.makeGiteaRequest(reqUrl)
		if err != nil {
			logger.Printf("failed to get users: %v", err)

			return nil, errors.Wrap(err, "failed to make Gitea request")
		}

		if g.LogLevel > 0 {
			logger.Printf(string(body))
		}

		switch resp.StatusCode {
		case http.StatusOK:
			if g.LogLevel > 0 {
				logger.Println("users retrieved successfully")
			}
		case http.StatusForbidden:
			logger.Println("failed to get users due to invalid or missing credentials (HTTP 403)")

			return nil, errors.Wrap(err, "forbidden response to Gitea request")
		default:
			logger.Printf("failed to get users with unexpected response: %d (%s)", resp.StatusCode, resp.Status)

			return nil, errors.Wrap(err, "unexpected errors making Gitea request")
		}

		var respObj giteaGetUsersResponse

		if err = json.Unmarshal(body, &respObj); err != nil {
			logger.Println(err)

			return nil, errors.Wrap(err, "failed to unmarshal Gitea response")
		}

		users = append(users, respObj...)
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

	return users, nil
}

func (g *GiteaHost) getOrganizations() ([]giteaOrganization, errors.E) {
	if len(g.Orgs) == 0 {
		if g.LogLevel > 0 {
			logger.Print("no organizations specified")
		}

		return nil, nil
	}

	if strings.TrimSpace(g.APIURL) == "" {
		g.APIURL = gitlabAPIURL
	}

	var organizations []giteaOrganization

	if slices.Contains(g.Orgs, "*") {
		var err errors.E

		organizations, err = g.getAllOrganizations()
		if err != nil {
			return nil, errors.Errorf("failed to get all organizations: %s", err.Error())
		}
	} else {
		for _, orgName := range g.Orgs {
			org, err := g.getOrganization(orgName)
			if err != nil {
				return nil, errors.Errorf("failed to get organization %s: %s", orgName, err.Error())
			}

			organizations = append(organizations, org)
		}
	}

	return organizations, nil
}

func (g *GiteaHost) getOrganization(orgName string) (giteaOrganization, errors.E) {
	if g.LogLevel > 0 {
		logger.Printf("retrieving organization %s", orgName)
	}

	if strings.TrimSpace(g.APIURL) == "" {
		g.APIURL = gitlabAPIURL
	}

	getOrganizationsURL := fmt.Sprintf("%s%s", g.APIURL+"/orgs/", orgName)

	if g.LogLevel > 0 {
		logger.Printf("get organization url: %s", getOrganizationsURL)
	}

	// Initial request
	u, err := url.Parse(getOrganizationsURL)
	if err != nil {
		logger.Printf("failed to parse get organization URL %s: %v", getOrganizationsURL, err)

		return giteaOrganization{}, errors.Errorf("failed to parse get organization URL: %s", err.Error())
	}

	// u.RawQuery = q.Encode()
	var body []byte

	reqUrl := u.String()

	var resp *http.Response

	resp, body, err = g.makeGiteaRequest(reqUrl)
	if err != nil {
		return giteaOrganization{}, errors.Wrap(err, fmt.Sprintf("failed to get organization: %s", orgName))
	}

	if g.LogLevel > 0 {
		logger.Print(string(body))
	}

	var organization giteaOrganization

	switch resp.StatusCode {
	case http.StatusOK:
		if g.LogLevel > 0 {
			logger.Println("organizations retrieved successfully")
		}
	case http.StatusForbidden:
		logger.Println("failed to get organizations due to invalid or missing credentials (HTTP 403)")

		return giteaOrganization{}, errors.Errorf("failed to get organizations due to invalid or missing credentials (HTTP 403)")
	default:
		logger.Printf("failed to get organizations with unexpected response: %d (%s)", resp.StatusCode, resp.Status)

		return giteaOrganization{}, errors.Errorf("failed to get organizations with unexpected response: %d (%s)", resp.StatusCode, resp.Status)
	}

	if err = json.Unmarshal(body, &organization); err != nil {
		logger.Printf("failed to unmarshal organization json response: %v", err.Error())

		return giteaOrganization{}, errors.Errorf("failed to unmarshal organization json response: %s", err.Error())
	}

	// if we got a link response then
	// reset request url
	// link: <https://gitea.lessknown.co.uk/api/v1/admin/organisations?limit=2&page=2>; rel="next",<https://gitea.lessknown.co.uk/api/v1/admin/organisations?limit=2&page=2>; rel="last"

	return organization, nil
}

func (g *GiteaHost) getAllOrganizations() ([]giteaOrganization, errors.E) {
	logger.Printf("retrieving organizations")

	if strings.TrimSpace(g.APIURL) == "" {
		g.APIURL = gitlabAPIURL
	}

	getOrganizationsURL := g.APIURL + "/orgs"
	if g.LogLevel > 0 {
		logger.Printf("get organizations url: %s", getOrganizationsURL)
	}

	// Initial request
	u, err := url.Parse(getOrganizationsURL)
	if err != nil {
		logger.Printf("failed to parse get organizations URL %s: %v", getOrganizationsURL, err)

		return nil, nil
	}

	q := u.Query()
	// set initial max per page
	q.Set("per_page", strconv.Itoa(giteaOrganizationsPerPageDefault))
	q.Set("limit", strconv.Itoa(giteaOrganizationsLimit))
	u.RawQuery = q.Encode()

	var body []byte

	reqUrl := u.String()

	var organizations []giteaOrganization

	for {
		var resp *http.Response

		resp, body, err = g.makeGiteaRequest(reqUrl)
		if err != nil {
			logger.Printf("failed to get organizations: %v", err.Error())

			return nil, nil
		}

		if g.LogLevel > 0 {
			logger.Print(string(body))
		}

		switch resp.StatusCode {
		case http.StatusOK:
			if g.LogLevel > 0 {
				logger.Println("organizations retrieved successfully")
			}
		case http.StatusForbidden:
			logger.Println("failed to get organizations due to invalid or missing credentials (HTTP 403)")

			return organizations, nil
		default:
			logger.Printf("failed to get organizations with unexpected response: %d (%s)",
				resp.StatusCode, resp.Status)

			return organizations, nil
		}

		var respObj giteaGetOrganizationsResponse

		if err = json.Unmarshal(body, &respObj); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal Gitea response")
		}

		organizations = append(organizations, respObj...)

		// if we got a link response then
		// reset request url
		// link: <https://gitea.lessknown.co.uk/api/v1/admin/organisations?limit=2&page=2>; rel="next",<https://gitea.lessknown.co.uk/api/v1/admin/organisations?limit=2&page=2>; rel="last"
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

	return organizations, nil
}

type giteaRepository struct {
	Id    int `json:"id"`
	Owner struct {
		Id                int       `json:"id"`
		Login             string    `json:"login"`
		LoginName         string    `json:"login_name"`
		FullName          string    `json:"full_name"`
		Email             string    `json:"email"`
		AvatarUrl         string    `json:"avatar_url"`
		Language          string    `json:"language"`
		IsAdmin           bool      `json:"is_admin"`
		LastLogin         time.Time `json:"last_login"`
		Created           time.Time `json:"created"`
		Restricted        bool      `json:"restricted"`
		Active            bool      `json:"active"`
		ProhibitLogin     bool      `json:"prohibit_login"`
		Location          string    `json:"location"`
		Website           string    `json:"website"`
		Description       string    `json:"description"`
		Visibility        string    `json:"visibility"`
		FollowersCount    int       `json:"followers_count"`
		FollowingCount    int       `json:"following_count"`
		StarredReposCount int       `json:"starred_repos_count"`
		Username          string    `json:"username"`
	} `json:"owner"`
	Name            string      `json:"name"`
	FullName        string      `json:"full_name"`
	Description     string      `json:"description"`
	Empty           bool        `json:"empty"`
	Private         bool        `json:"private"`
	Fork            bool        `json:"fork"`
	Template        bool        `json:"template"`
	Parent          interface{} `json:"parent"`
	Mirror          bool        `json:"mirror"`
	Size            int         `json:"size"`
	Language        string      `json:"language"`
	LanguagesUrl    string      `json:"languages_url"`
	HtmlUrl         string      `json:"html_url"`
	Link            string      `json:"link"`
	SshUrl          string      `json:"ssh_url"`
	CloneUrl        string      `json:"clone_url"`
	OriginalUrl     string      `json:"original_url"`
	Website         string      `json:"website"`
	StarsCount      int         `json:"stars_count"`
	ForksCount      int         `json:"forks_count"`
	WatchersCount   int         `json:"watchers_count"`
	OpenIssuesCount int         `json:"open_issues_count"`
	OpenPrCounter   int         `json:"open_pr_counter"`
	ReleaseCounter  int         `json:"release_counter"`
	DefaultBranch   string      `json:"default_branch"`
	Archived        bool        `json:"archived"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	ArchivedAt      time.Time   `json:"archived_at"`
	Permissions     struct {
		Admin bool `json:"admin"`
		Push  bool `json:"push"`
		Pull  bool `json:"pull"`
	} `json:"permissions"`
	HasIssues       bool `json:"has_issues"`
	InternalTracker struct {
		EnableTimeTracker                bool `json:"enable_time_tracker"`
		AllowOnlyContributorsToTrackTime bool `json:"allow_only_contributors_to_track_time"`
		EnableIssueDependencies          bool `json:"enable_issue_dependencies"`
	} `json:"internal_tracker"`
	HasWiki                       bool        `json:"has_wiki"`
	HasPullRequests               bool        `json:"has_pull_requests"`
	HasProjects                   bool        `json:"has_projects"`
	HasReleases                   bool        `json:"has_releases"`
	HasPackages                   bool        `json:"has_packages"`
	HasActions                    bool        `json:"has_actions"`
	IgnoreWhitespaceConflicts     bool        `json:"ignore_whitespace_conflicts"`
	AllowMergeCommits             bool        `json:"allow_merge_commits"`
	AllowRebase                   bool        `json:"allow_rebase"`
	AllowRebaseExplicit           bool        `json:"allow_rebase_explicit"`
	AllowSquashMerge              bool        `json:"allow_squash_merge"`
	AllowRebaseUpdate             bool        `json:"allow_rebase_update"`
	DefaultDeleteBranchAfterMerge bool        `json:"default_delete_branch_after_merge"`
	DefaultMergeStyle             string      `json:"default_merge_style"`
	DefaultAllowMaintainerEdit    bool        `json:"default_allow_maintainer_edit"`
	AvatarUrl                     string      `json:"avatar_url"`
	Internal                      bool        `json:"internal"`
	MirrorInterval                string      `json:"mirror_interval"`
	MirrorUpdated                 time.Time   `json:"mirror_updated"`
	RepoTransfer                  interface{} `json:"repo_transfer"`
}

func (g *GiteaHost) getOrganizationRepos(organizationName string) ([]giteaRepository, errors.E) {
	logger.Printf("retrieving repositories for organization %s", organizationName)

	if strings.TrimSpace(g.APIURL) == "" {
		g.APIURL = gitlabAPIURL
	}

	getOrganizationReposURL := g.APIURL + fmt.Sprintf("/orgs/%s/repos", organizationName)
	if g.LogLevel > 0 {
		logger.Printf("get %s organization repos url: %s", organizationName, getOrganizationReposURL)
	}

	// Initial request
	u, err := url.Parse(getOrganizationReposURL)
	if err != nil {
		return nil, errors.Errorf("failed to parse get %s organization repos URL %s: %s", organizationName, getOrganizationReposURL, err)
	}

	q := u.Query()
	// set initial max per page
	q.Set("per_page", strconv.Itoa(giteaReposPerPageDefault))
	q.Set("limit", strconv.Itoa(giteaReposLimit))
	u.RawQuery = q.Encode()

	var body []byte

	var repos []giteaRepository

	reqUrl := u.String()

	for {
		var resp *http.Response

		resp, body, err = g.makeGiteaRequest(reqUrl)
		if err != nil {
			return nil, errors.Errorf("failed to make Gitea request: %s", err)
		}

		if g.LogLevel > 0 {
			logger.Print(string(body))
		}

		switch resp.StatusCode {
		case http.StatusOK:
			if g.LogLevel > 0 {
				logger.Println("repos retrieved successfully")
			}
		case http.StatusForbidden:
			return nil, errors.Errorf("failed to get repos due to invalid or missing credentials (HTTP 403)")
		default:
			logger.Printf("failed to get repos with unexpected response: %d (%s)", resp.StatusCode, resp.Status)

			return nil, nil
		}

		var respObj []giteaRepository

		if err = json.Unmarshal(body, &respObj); err != nil {
			return nil, errors.Errorf("failed to unmarshal organization repos json response: %s", err)
		}

		repos = append(repos, respObj...)

		// if we got a link response then
		// reset request url
		// link: <https://gitea.lessknown.co.uk/api/v1/admin/repos?limit=2&page=2>; rel="next",<https://gitea.lessknown.co.uk/api/v1/admin/repos?limit=2&page=2>; rel="last"
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

func (g *GiteaHost) getAllUserRepos(userName string) ([]repository, errors.E) {
	logger.Printf("retrieving all repositories for user %s", userName)

	if strings.TrimSpace(g.APIURL) == "" {
		g.APIURL = gitlabAPIURL
	}

	getOrganizationReposURL := g.APIURL + fmt.Sprintf("/users/%s/repos", userName)
	if g.LogLevel > 0 {
		logger.Printf("get %s user repos url: %s", userName, getOrganizationReposURL)
	}

	// Initial request
	u, err := url.Parse(getOrganizationReposURL)
	if err != nil {
		logger.Printf("failed to parse get %s user repos URL %s: %v", userName, getOrganizationReposURL, err)

		return nil, errors.Wrap(err, "failed to parse get user repos URL")
	}

	q := u.Query()
	// set initial max per page
	q.Set("per_page", strconv.Itoa(giteaReposPerPageDefault))
	q.Set("limit", strconv.Itoa(giteaReposLimit))
	u.RawQuery = q.Encode()

	var body []byte

	var repos []repository

	reqUrl := u.String()

	for {
		var resp *http.Response

		resp, body, err = g.makeGiteaRequest(reqUrl)
		if err != nil {
			logger.Printf("failed to get repos: %v", err)

			return nil, errors.Wrap(err, "failed to parse get user repos URL")
		}

		if g.LogLevel > 0 {
			logger.Print(string(body))
		}

		switch resp.StatusCode {
		case http.StatusOK:
			if g.LogLevel > 0 {
				logger.Println("repos retrieved successfully")
			}
		case http.StatusForbidden:
			logger.Println("failed to get repos due to invalid or missing credentials (HTTP 403)")

			return nil, errors.Wrap(err, "failed to get repos due to invalid or missing credentials (HTTP 403)")
		default:
			logger.Printf("failed to get repos with unexpected response: %d (%s)", resp.StatusCode, resp.Status)

			return nil, errors.Wrap(err, "failed to parse get user repos URL")
		}

		var respObj []giteaRepository

		if err = json.Unmarshal(body, &respObj); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal user repos json response")
		}

		for _, r := range respObj {
			var ru *url.URL

			ru, err = url.Parse(r.CloneUrl)
			if err != nil {
				logger.Printf("failed to parse clone url for %s\n", r.Name)

				return nil, errors.Wrap(err, fmt.Sprintf("failed to parse clone url for: %s", r.CloneUrl))
			}

			repos = append(repos, repository{
				Name:              r.Name,
				Owner:             r.Owner.Login,
				HTTPSUrl:          r.CloneUrl,
				SSHUrl:            r.SshUrl,
				Domain:            ru.Host,
				PathWithNameSpace: r.FullName,
			})
		}

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

func (g *GiteaHost) getAPIURL() string {
	return g.APIURL
}

// return normalised method.
func (g *GiteaHost) diffRemoteMethod() string {
	switch strings.ToLower(g.DiffRemoteMethod) {
	case refsMethod:
		return refsMethod
	case cloneMethod:
		return cloneMethod
	default:
		logger.Printf("unexpected diff remote method: %s", g.DiffRemoteMethod)

		return "invalid remote comparison method"
	}
}

func giteaWorker(token string, logLevel int, backupDIR, diffRemoteMethod string, backupsToKeep int, jobs <-chan repository, results chan<- RepoBackupResults) {
	for repo := range jobs {
		firstPos := strings.Index(repo.HTTPSUrl, "//")
		repo.URLWithToken = fmt.Sprintf("%s%s@%s", repo.HTTPSUrl[:firstPos+2], token, repo.HTTPSUrl[firstPos+2:])
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

func (g *GiteaHost) Backup() ProviderBackupResult {
	if g.BackupDir == "" {
		logger.Printf("backup skipped as backup directory not specified")

		return ProviderBackupResult{}
	}

	maxConcurrent := 5

	repoDesc, err := g.describeRepos()
	if err != nil {
		return ProviderBackupResult{
			BackupResults: nil,
			Error:         err,
		}
	}

	jobs := make(chan repository, len(repoDesc.Repos))
	results := make(chan RepoBackupResults, maxConcurrent)

	for w := 1; w <= maxConcurrent; w++ {
		go giteaWorker(g.Token, g.LogLevel, g.BackupDir, g.diffRemoteMethod(), g.BackupsToRetain, jobs, results)
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

func (g *GiteaHost) getAllUserRepositories() ([]repository, errors.E) {
	users, err := g.getAllUsers()
	if err != nil {
		logger.Print("failed to get all users")

		return nil, errors.Wrap(err, "failed to get all users")
	}

	var repos []repository

	var userCount int

	for _, user := range users {
		userCount++

		var userRepos []repository

		userRepos, err = g.getAllUserRepos(user.Login)
		if err != nil {
			logger.Print("failed to get all user repositories")

			return nil, errors.Wrap(err, "failed to get all user repositories")
		}

		repos = append(repos, userRepos...)
	}

	var repositories []repository
	for _, repo := range repos {
		repositories = append(repositories, repository{
			Name:              repo.Name,
			Owner:             repo.Owner,
			PathWithNameSpace: repo.PathWithNameSpace,
			Domain:            repo.Domain,
			HTTPSUrl:          repo.HTTPSUrl,
			SSHUrl:            repo.SSHUrl,
		})
	}

	return repositories, nil
}
