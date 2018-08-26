package githosts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	gitHubCallSize = 100
)

type githubHost struct {
	Provider string
	APIURL   string
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
	}
}

func (provider githubHost) describeRepos() describeReposOutput {
	logger.Println("listing GitHub repositories")
	tr := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr}

	var repos []repository
	reqBody := "{\"query\": \"query { viewer { repositories(first:" + strconv.Itoa(gitHubCallSize) + ") { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }} } }\""
	for {
		mJSON := reqBody
		contentReader := bytes.NewReader([]byte(mJSON))
		req, newReqErr := http.NewRequest(http.MethodPost, "https://api.github.com/graphql", contentReader)
		if newReqErr != nil {
			logger.Fatal(newReqErr)
		}
		req.Header.Set("Authorization", fmt.Sprintf("bearer %s",
			stripTrailing(os.Getenv("GITHUB_TOKEN"), "\n")))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "application/json; charset=utf-8")

		resp, reqErr := client.Do(req)
		if reqErr != nil {
			logger.Fatal(reqErr)
		}
		bodyB, _ := ioutil.ReadAll(resp.Body)
		bodyStr := string(bytes.Replace(bodyB, []byte("\r"), []byte("\r\n"), -1))
		var respObj githubQueryNamesResponse
		if err := json.Unmarshal([]byte(bodyStr), &respObj); err != nil {
			logger.Fatal(err)
		}
		for _, repo := range respObj.Data.Viewer.Repositories.Edges {
			repos = append(repos, repository{
				Name:          repo.Node.Name,
				SSHUrl:        repo.Node.SSHURL,
				HTTPSUrl:      repo.Node.URL,
				NameWithOwner: repo.Node.NameWithOwner,
				Domain:        "github.com",
			})
		}
		if !respObj.Data.Viewer.Repositories.PageInfo.HasNextPage {
			break
		} else {
			reqBody = "{\"query\": \"query($first:Int $after:String){ viewer { repositories(first:$first after:$after) { edges { node { name nameWithOwner url sshUrl } cursor } pageInfo { endCursor hasNextPage }} } }\", \"variables\":{\"first\":" + strconv.Itoa(gitHubCallSize) + ",\"after\":\"" + respObj.Data.Viewer.Repositories.PageInfo.EndCursor + "\"} }"
		}
	}

	return describeReposOutput{
		Repos: repos,
	}
}

func (provider githubHost) getAPIURL() string {
	return provider.APIURL
}

func gitHubWorker(backupDIR string, jobs <-chan repository, results chan<- error) {
	for repo := range jobs {
		firstPos := strings.Index(repo.HTTPSUrl, "//")
		repo.URLWithToken = fmt.Sprintf("%s%s@%s", repo.HTTPSUrl[:firstPos+2], stripTrailing(os.Getenv("GITHUB_TOKEN"), "\n"), repo.HTTPSUrl[firstPos+2:])
		results <- processBackup(repo, backupDIR)
	}
}

func (provider githubHost) Backup(backupDIR string) {
	maxConcurrent := 5
	repoDesc := provider.describeRepos()

	jobs := make(chan repository, len(repoDesc.Repos))
	results := make(chan error, maxConcurrent)

	for w := 1; w <= maxConcurrent; w++ {
		go gitHubWorker(backupDIR, jobs, results)
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
