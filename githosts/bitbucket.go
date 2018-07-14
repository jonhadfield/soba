package githosts

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type bitbucketHost struct {
	Provider string
	APIURL   string
}

type bitbucketProject struct {
	Scm   string `json:"scm"`
	Owner string `json:"owner"`
	Name  string `json:"name"`
}
type bitbucketGetProjectsResponse []bitbucketProject

func injectCreds(url string) string {
	parts := strings.Split(url, "://")
	return parts[0] + "://" + os.Getenv("BITBUCKET_USER") + ":" + os.Getenv("BITBUCKET_APP_PASSWORD") + "@" + parts[1]
}

func (provider bitbucketHost) describeRepos() describeReposOutput {
	logger.Println("listing BitBucket repositories")
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}
	var repos []repository
	rawRequestURL := provider.APIURL + string(os.PathSeparator) + "user" + string(os.PathSeparator) + "repositories"
	getReposURL := injectCreds(rawRequestURL)
	req, _ := http.NewRequest(http.MethodGet, getReposURL, nil)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	resp, _ := client.Do(req)
	bodyB, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(bytes.Replace(bodyB, []byte("\r"), []byte("\r\n"), -1))
	var respObj bitbucketGetProjectsResponse
	if err := json.Unmarshal([]byte(bodyStr), &respObj); err != nil {
		logger.Fatal(err)
		os.Exit(1)
	}
	for _, project := range respObj {
		if project.Scm == "git" {
			var repo = repository{
				Name:          project.Name,
				Domain:        "bitbucket.org",
				HTTPSUrl:      "https://bitbucket.org/" + project.Owner + "/" + project.Name + ".git",
				NameWithOwner: project.Owner + "/" + project.Name,
			}
			repos = append(repos, repo)
		}
	}
	return describeReposOutput{
		Repos: repos,
	}
}

func (provider bitbucketHost) getAPIURL() string {
	return provider.APIURL
}

func (provider bitbucketHost) Backup(backupDIR string) {
	describe := provider.describeRepos()
	for _, repo := range describe.Repos {
		parts := strings.Split(repo.HTTPSUrl, "//")
		repo.URLWithBasicAuth = parts[0] + "//" + os.Getenv("BITBUCKET_USER") + ":" + os.Getenv("BITBUCKET_APP_PASSWORD") + "@" + parts[1]
		processBackup(repo, backupDIR)
	}
}
