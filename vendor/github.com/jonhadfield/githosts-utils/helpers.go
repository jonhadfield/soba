package githosts

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"gitlab.com/tozd/go/errors"
)

const (
	backupDirMode = 0o755
)

func createDirIfAbsent(path string) error {
	return os.MkdirAll(path, backupDirMode)
}

func getTimestamp() string {
	t := time.Now()

	return t.Format(timeStampFormat)
}

func timeStampToTime(s string) (time.Time, errors.E) {
	if len(s) != bundleTimestampChars {
		return time.Time{}, errors.New("invalid timestamp")
	}

	ptime, err := time.Parse(timeStampFormat, s)
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to parse timestamp")
	}

	return ptime, nil
}

func stripTrailing(input string, toStrip string) string {
	if strings.HasSuffix(input, toStrip) {
		return input[:len(input)-len(toStrip)]
	}

	return input
}

func urlWithToken(httpsURL, token string) string {
	pos := strings.Index(httpsURL, "//")
	if pos == -1 {
		return httpsURL
	}

	return fmt.Sprintf("%s%s@%s", httpsURL[:pos+2], stripTrailing(token, "\n"), httpsURL[pos+2:])
}

func urlWithBasicAuth(httpsURL, user, password string) string {
	parts := strings.SplitN(httpsURL, "//", 2)
	if len(parts) != 2 {
		return httpsURL
	}

	return fmt.Sprintf("%s//%s:%s@%s", parts[0], user, password, parts[1])
}

func isEmpty(clonedRepoPath string) (bool, errors.E) {
	remoteHeadsCmd := exec.Command("git", "count-objects", "-v")
	remoteHeadsCmd.Dir = clonedRepoPath

	out, err := remoteHeadsCmd.CombinedOutput()
	if err != nil {
		return true, errors.Wrapf(err, "failed to count objects in %s", clonedRepoPath)
	}

	cmdOutput := strings.Split(string(out), "\n")

	var looseObjects bool

	var inPackObjects bool

	var matchingLinesFound int

	for _, line := range cmdOutput {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			switch fields[0] {
			case "count:":
				matchingLinesFound++

				looseObjects = fields[1] != "0"
			case "in-pack:":
				matchingLinesFound++

				inPackObjects = fields[1] != "0"
			}
		}
	}

	if matchingLinesFound != 2 {
		return false, errors.Errorf("failed to get object counts from %s", clonedRepoPath)
	}

	if !looseObjects && !inPackObjects {
		return true, nil
	}

	return false, nil
}

func getResponseBody(resp *http.Response) ([]byte, error) {
	var output io.ReadCloser

	var err error

	if resp.Header.Get("Content-Encoding") == "gzip" {
		output, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to get response body: %w", err)
		}
	} else {
		output = resp.Body
	}

	buf := new(bytes.Buffer)
	if _, err = buf.ReadFrom(output); err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return buf.Bytes(), nil
}

func maskSecrets(content string, secret []string) string {
	for _, s := range secret {
		content = strings.ReplaceAll(content, s, strings.Repeat("*", len(s)))
	}

	return content
}

type httpRequestInput struct {
	client            *retryablehttp.Client
	url               string
	method            string
	headers           http.Header
	reqBody           []byte
	secrets           []string
	basicAuthUser     string
	basicAuthPassword string
	timeout           time.Duration
}

func httpRequest(in httpRequestInput) ([]byte, http.Header, int, error) {
	if in.method == "" {
		return nil, nil, 0, errors.New("HTTP method not specified")
	}

	req, err := retryablehttp.NewRequest(in.method, in.url, in.reqBody)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to request %s: %w", maskSecrets(in.url, in.secrets), err)
	}

	req.Header = in.headers

	var resp *http.Response

	resp, err = in.client.Do(req)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("request failed: %w", err)
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %s\n", err.Error())
		}
	}(resp.Body)

	body, err := getResponseBody(resp)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("%w", err)
	}

	return body, resp.Header, resp.StatusCode, err
}

func getDiffRemoteMethod(input string) (string, error) {
	if input == "" {
		return input, nil
	}

	if err := validDiffRemoteMethod(input); err != nil {
		return input, err
	}

	return input, nil
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}

	return s
}
