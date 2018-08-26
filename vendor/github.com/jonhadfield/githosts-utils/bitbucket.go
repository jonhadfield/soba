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
	return parts[0] + "://" + os.Getenv("BITBUCKET_USER") + ":" + stripTrailing(os.Getenv("BITBUCKET_APP_PASSWORD"), "\n") + "@" + parts[1]
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
	rawRequestURL := provider.APIURL + "/user/repositories"
	getReposURL := injectCreds(rawRequestURL)
	req, errNewReq := http.NewRequest(http.MethodGet, getReposURL, nil)
	if errNewReq != nil {
		logger.Fatal(errNewReq)
	}
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

func bitBucketWorker(backupDIR string, jobs <-chan repository, results chan<- error) {
	for repo := range jobs {
		parts := strings.Split(repo.HTTPSUrl, "//")
		repo.URLWithBasicAuth = parts[0] + "//" + os.Getenv("BITBUCKET_USER") + ":" + stripTrailing(os.Getenv("BITBUCKET_APP_PASSWORD"), "\n") + "@" + parts[1]
		results <- processBackup(repo, backupDIR)
	}
}

func (provider bitbucketHost) Backup(backupDIR string) {
	maxConcurrent := 5
	repoDesc := provider.describeRepos()

	jobs := make(chan repository, len(repoDesc.Repos))
	results := make(chan error, maxConcurrent)

	for w := 1; w <= maxConcurrent; w++ {
		go bitBucketWorker(backupDIR, jobs, results)
	}

	for x := range repoDesc.Repos {
		repo := repoDesc.Repos[x]
		jobs <- repo
	}
	close(jobs)

	for a := 1; a <= len(repoDesc.Repos); a++ {
		res := <-results
		if res != nil {
			logger.Fatal(res)
		}
	}
}
