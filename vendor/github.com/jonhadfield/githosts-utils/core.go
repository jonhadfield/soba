package githosts

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"gitlab.com/tozd/go/errors"
)

const (
	envVarGitBackupDir  = "GIT_BACKUP_DIR"
	envVarGitHostsLog   = "GITHOSTS_LOG"
	refsMethod          = "refs"
	cloneMethod         = "clone"
	defaultRemoteMethod = cloneMethod
	logEntryPrefix      = "githosts-utils: "
	statusOk            = "ok"
	statusFailed        = "failed"
)

type repository struct {
	Name              string
	Owner             string
	PathWithNameSpace string
	Domain            string
	HTTPSUrl          string
	SSHUrl            string
	URLWithToken      string
	URLWithBasicAuth  string
}

type describeReposOutput struct {
	Repos []repository
}

type RepoBackupResults struct {
	Repo   string   `json:"repo,omitempty"`
	Status string   `json:"status,omitempty"` // ok, failed
	Error  errors.E `json:"error,omitempty"`
}

// type ProviderBackupResult []RepoBackupResults
type ProviderBackupResult struct {
	BackupResults []RepoBackupResults
	Error         errors.E
}

type gitProvider interface {
	getAPIURL() string
	describeRepos() (describeReposOutput, errors.E)
	Backup() ProviderBackupResult
	diffRemoteMethod() string
}

// gitRefs is a mapping of references to SHAs.
type gitRefs map[string]string

func remoteRefsMatchLocalRefs(cloneURL, backupPath string) bool {
	// if there's no backup path then return false
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return false
	}

	// if there are no backups
	if !dirHasBundles(backupPath) {
		return false
	}

	var rHeads, lHeads gitRefs

	var err error

	lHeads, err = getLatestBundleRefs(backupPath)
	if err != nil {
		logger.Printf("failed to get latest bundle refs for %s", backupPath)

		return false
	}

	rHeads, err = getRemoteRefs(cloneURL)
	if err != nil {
		logger.Printf("failed to get remote refs")

		return false
	}

	if reflect.DeepEqual(lHeads, rHeads) {
		return true
	}

	return false
}

func cutBySpaceAndTrimOutput(in string) (before, after string, found bool) {
	// remove leading and trailing space
	in = strings.TrimSpace(in)
	// try cutting by tab
	b, a, f := strings.Cut(in, "\t")
	if f {
		b = strings.TrimSpace(b)
		a = strings.TrimSpace(a)

		if len(a) > 0 && len(b) > 0 {
			return b, a, true
		}
	}

	// try cutting by tab
	b, a, f = strings.Cut(in, " ")
	if f {
		b = strings.TrimSpace(b)
		a = strings.TrimSpace(a)
		if len(a) > 0 && len(b) > 0 {
			return b, a, true
		}
	}

	return
}

func generateMapFromRefsCmdOutput(in []byte) (refs gitRefs, err error) {
	refs = make(map[string]string)
	lines := strings.Split(string(in), "\n")

	for x := range lines {
		// if empty (final line perhaps) then skip
		if len(strings.TrimSpace(lines[x])) == 0 {
			continue
		}

		// try cutting ref by both space and tab as its possible for both to be used
		sha, ref, found := cutBySpaceAndTrimOutput(lines[x])

		// expect only a sha and a ref
		if !found {
			logger.Printf("skipping invalid ref: %s", strings.TrimSpace(lines[x]))

			continue
		}

		// git bundle list-heads returns pseudo-refs but not peeled tags
		// this is required for comparison with remote references
		if slices.Contains([]string{"HEAD", "FETCH_HEAD", "ORIG_HEAD", "MERGE_HEAD", "CHERRY_PICK_HEAD"}, ref) {
			continue
		}

		refs[ref] = sha
	}

	return
}

func getRemoteRefs(cloneURL string) (refs gitRefs, err error) {
	// --refs ignores pseudo-refs like HEAD and FETCH_HEAD, and also peeled tags that reference other objects
	// this enables comparison with refs from existing bundles
	remoteHeadsCmd := exec.Command("git", "ls-remote", "--refs", cloneURL)

	out, err := remoteHeadsCmd.CombinedOutput()
	if err != nil {
		return refs, errors.Wrap(err, "failed to retrieve remote heads")
	}

	refs, err = generateMapFromRefsCmdOutput(out)

	return
}

func processBackup(logLevel int, repo repository, backupDIR string, backupsToKeep int, diffRemoteMethod string) errors.E {
	// create backup path
	workingPath := filepath.Join(backupDIR, workingDIRName, repo.Domain, repo.PathWithNameSpace)
	backupPath := filepath.Join(backupDIR, repo.Domain, repo.PathWithNameSpace)
	// clean existing working directory
	delErr := os.RemoveAll(workingPath)
	if delErr != nil {
		return errors.Errorf("failed to remove working directory: %s: %s", workingPath, delErr)
	}

	var cloneURL string

	if repo.URLWithToken != "" {
		cloneURL = repo.URLWithToken
	} else if repo.URLWithBasicAuth != "" {
		cloneURL = repo.URLWithBasicAuth
	}

	// Check if existing, latest bundle refs, already match the remote
	if diffRemoteMethod == refsMethod {
		// check backup path exists before attempting to compare remote and local heads
		if remoteRefsMatchLocalRefs(cloneURL, backupPath) {
			logger.Printf("skipping clone of %s repo '%s' as refs match existing bundle", repo.Domain, repo.PathWithNameSpace)

			return nil
		}
	}

	// clone repo
	logger.Printf("cloning: %s to: %s", repo.HTTPSUrl, workingPath)

	cloneCmd := exec.Command("git", "clone", "-v", "--mirror", cloneURL, workingPath)
	cloneCmd.Dir = backupDIR

	cloneOut, cloneErr := cloneCmd.CombinedOutput()
	if cloneErr != nil {
		fmt.Printf("cloning failed for repository: %s - %s\n", repo.Name, cloneErr)
	}

	cloneOutLines := strings.Split(string(cloneOut), "\n")

	if cloneErr != nil {
		if os.Getenv(envVarGitHostsLog) == "debug" {
			fmt.Printf("debug: cloning failed for repository: %s - %s\n", repo.Name, strings.Join(cloneOutLines, ", "))

			return errors.Errorf("cloning failed: %s: %s", strings.Join(cloneOutLines, ", "), cloneErr)
		}

		return errors.Errorf("cloning failed for repository: %s - %s", repo.Name, cloneErr)
	}

	// create bundle
	if err := createBundle(logLevel, workingPath, backupPath, repo); err != nil {
		if strings.HasSuffix(err.Error(), "is empty") {
			logger.Printf("skipping empty %s repository %s", repo.Domain, repo.PathWithNameSpace)

			return nil
		}

		return err
	}

	removeBundleIfDuplicate(backupPath)

	if backupsToKeep > 0 {
		if err := pruneBackups(backupPath, backupsToKeep); err != nil {
			return err
		}
	}

	return nil
}

func getHTTPClient() *retryablehttp.Client {
	tr := &http.Transport{
		DisableKeepAlives:  false,
		DisableCompression: true,
		MaxIdleConns:       maxIdleConns,
		IdleConnTimeout:    idleConnTimeout,
		ForceAttemptHTTP2:  false,
	}

	rc := retryablehttp.NewClient()
	rc.HTTPClient = &http.Client{
		Transport: tr,
		Timeout:   120 * time.Second,
	}

	rc.Logger = nil
	rc.RetryWaitMax = 120 * time.Second
	rc.RetryWaitMin = 60 * time.Second
	rc.RetryMax = 2

	return rc
}

func validDiffRemoteMethod(method string) error {
	if !slices.Contains([]string{cloneMethod, refsMethod}, method) {
		return fmt.Errorf("invalid diff remote method: %s", method)
	}

	return nil
}

func setLoggerPrefix(prefix string) {
	if prefix != "" {
		logger.SetPrefix(fmt.Sprintf("%s: ", prefix))
	}
}

func allTrue(in ...bool) bool {
	for _, v := range in {
		if !v {
			return false
		}
	}

	return true
}

func ToPtr[T any](v T) *T {
	return &v
}
