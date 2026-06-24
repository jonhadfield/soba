package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/jonhadfield/githosts-utils"
	"github.com/stretchr/testify/require"
)

// newGitFixtureServer creates a bare git repository per name (each with a
// single commit) and serves them over dumb HTTP. The backup code clones with
// `git clone --mirror http://<token>@host/<name>.git`; dumb HTTP ignores the
// unused basic-auth userinfo, so the token injected by githosts-utils is
// harmless. Returns the server base URL; callers build clone URLs as
// baseURL + "/" + name + ".git".
func newGitFixtureServer(t *testing.T, names []string) string {
	t.Helper()

	root := t.TempDir()

	for _, name := range names {
		work := filepath.Join(root, "work-"+name)
		bare := filepath.Join(root, name+".git")

		runGit(t, "", "init", "-q", work)
		writeFixtureFile(t, filepath.Join(work, "README.md"), "# "+name+"\n")
		runGit(t, work, "-c", "user.email=test@example.com", "-c", "user.name=test", "add", ".")
		runGit(t, work, "-c", "user.email=test@example.com", "-c", "user.name=test", "commit", "-q", "-m", "init")
		// Bare clone, then publish dumb-HTTP metadata so `git clone` works
		// without a smart-HTTP server.
		runGit(t, "", "clone", "-q", "--bare", work, bare)
		runGit(t, bare, "update-server-info")
	}

	srv := httptest.NewServer(http.FileServer(http.Dir(root)))
	t.Cleanup(srv.Close)

	return srv.URL
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeFixtureFile(t *testing.T, p, content string) {
	t.Helper()

	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

// newGitHubMockAPI returns an httptest server that answers the GraphQL
// `viewer { repositories }` query with the supplied repositories. Each repo's
// url points at the local git fixture server so the subsequent clone is fully
// local. owner/name form the github.com/<owner>/<name> backup path.
func newGitHubMockAPI(t *testing.T, owner, fixtureBaseURL string, names []string) *httptest.Server {
	t.Helper()

	edges := make([]map[string]any, 0, len(names))
	for _, name := range names {
		edges = append(edges, map[string]any{
			"node": map[string]any{
				"name":          name,
				"nameWithOwner": owner + "/" + name,
				"url":           fixtureBaseURL + "/" + name + ".git",
				"sshUrl":        "",
			},
			"cursor": name,
		})
	}

	body := map[string]any{
		"data": map[string]any{
			"viewer": map[string]any{
				"repositories": map[string]any{
					"edges": edges,
					"pageInfo": map[string]any{
						"endCursor":   "",
						"hasNextPage": false,
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)

	return srv
}

// TestGithubRepositoryBackupWithMockAPI exercises the full GitHub backup path
// — repo enumeration, clone and bundle creation — against a mock GraphQL API
// and a local git fixture, so it is deterministic and needs no token, network
// access or live GitHub (which secondary-rate-limits the CI token; see #176).
func TestGithubRepositoryBackupWithMockAPI(t *testing.T) {
	const owner = "mockuser"

	names := []string{"repo-one", "repo-two"}

	fixtureURL := newGitFixtureServer(t, names)
	api := newGitHubMockAPI(t, owner, fixtureURL, names)

	backupDir := t.TempDir()

	githubHost, err := githosts.NewGitHubHost(githosts.NewGitHubHostInput{
		Caller:          AppName,
		APIURL:          api.URL,
		BackupDir:       backupDir,
		Token:           "mock-token",
		BackupsToRetain: 1,
	})
	require.NoError(t, err)

	result := githubHost.Backup()
	require.Nil(t, result.Error)
	require.Len(t, result.BackupResults, len(names))

	for _, r := range result.BackupResults {
		require.Nil(t, r.Error)
	}

	for _, name := range names {
		repoDir := path.Join(backupDir, "github.com", owner, name)
		require.DirExists(t, repoDir)

		entries, rErr := os.ReadDir(repoDir)
		require.NoError(t, rErr)
		require.Len(t, entries, 1)
		require.Regexp(t, regexp.MustCompile(fmt.Sprintf(`^%s\.\d{14}\.bundle$`, regexp.QuoteMeta(name))), entries[0].Name())
	}
}
