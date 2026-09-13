package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// withPushStatus adds the status and msg query parameters that push-style
// monitors (Uptime Kuma, and others following the same convention) read to
// decide whether a heartbeat is healthy. Without it every run reads as healthy,
// because such monitors treat the arrival of the request as the signal and
// ignore the body - so a run in which every repository failed would still show
// green. Opt-in, since appending parameters to an arbitrary webhook URL would
// otherwise be surprising.
func withPushStatus(rawURL string, succeeded, failed int) (string, error) {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("error parsing webhook url: %w", err)
	}

	status := "up"
	if failed > 0 {
		status = "down"
	}

	q := u.Query()
	q.Set("status", status)
	q.Set("msg", fmt.Sprintf("succeeded: %d, failed: %d", succeeded, failed))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// Event types reported in the webhook payload. They mirror the three outcomes
// backupStatusTitle distinguishes for the other notifiers, so a consumer can
// branch on the outcome without having to compare the counts itself.
const (
	eventBackupsComplete   = "backups.complete"
	eventBackupsWithErrors = "backups.complete_with_errors"
	eventBackupsFailed     = "backups.failed"
)

// backupEventType classifies a run. Every run previously reported
// backups.complete, including one in which every repository failed, which left
// the field useless for deciding whether to alert.
func backupEventType(succeeded, failed int) string {
	switch {
	case failed == 0:
		return eventBackupsComplete
	case succeeded > 0:
		return eventBackupsWithErrors
	default:
		return eventBackupsFailed
	}
}

func sendWebhook(c *retryablehttp.Client, sendTime sobaTime, results BackupResults, url, format string) error {
	ok, failed := getBackupsStats(results)

	if envTrue(envSobaWebHookPushStatus) {
		var err error
		if url, err = withPushStatus(url, ok, failed); err != nil {
			return err
		}
	}

	if sendTime.IsZero() {
		sendTime = sobaTime{
			Time: time.Now(),
			f:    time.RFC3339,
		}
	}

	webhookData := WebhookData{
		App:       AppName,
		Type:      backupEventType(ok, failed),
		Timestamp: sendTime,
		Stats: BackupStats{
			Succeeded: ok,
			Failed:    failed,
		},
		Data: results,
	}

	// exclude result data if format is short
	if format == "short" {
		webhookData.Data.Results = nil
	}

	o, err := json.Marshal(webhookData)
	if err != nil {
		return fmt.Errorf("error marshalling webhook data: %w", err)
	}

	// Build a dedicated webhook client so webhook-specific retry tuning does
	// not mutate the shared client used by provider HTTP calls. retryablehttp
	// clients are non-copyable (sync.Once), so construct a fresh one.
	wc := retryablehttp.NewClient()
	if c != nil && c.HTTPClient != nil {
		wc.HTTPClient = c.HTTPClient
	}

	wc.RetryMax = webhookRetryMax
	wc.RetryWaitMin = webhookRetryWaitMin
	wc.RetryWaitMax = webhookRetryWaitMax
	wc.Logger = nil

	var req *retryablehttp.Request

	req, err = retryablehttp.NewRequest(http.MethodPost, url, strings.NewReader(string(o)))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := wc.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

type BackupStats struct {
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}

type sobaTime struct {
	time.Time
	f string
}

func (j sobaTime) format() string {
	return j.Format(j.f)
}

func (j sobaTime) MarshalText() ([]byte, error) { // nolint: unparam
	return []byte(j.format()), nil
}

func (j sobaTime) MarshalJSON() ([]byte, error) { // nolint: unparam
	return []byte(`"` + j.format() + `"`), nil
}

type WebhookData struct {
	App       string        `json:"app"`
	Type      string        `json:"type"`
	Stats     BackupStats   `json:"stats"`
	Timestamp sobaTime      `json:"timestamp"`
	Data      BackupResults `json:"data,omitempty"`
}

func getBackupsStats(br BackupResults) (ok, failed int) {
	if br.Results == nil {
		return 0, 0
	}

	for _, pr := range *br.Results {
		// catch error from provider
		if pr.Results.Error != nil {
			failed++

			continue
		}

		providerOk := 0

		for _, r := range pr.Results.BackupResults {
			// catch error from repository backup
			if r.Error != nil {
				failed++

				continue
			}

			ok++
			providerOk++
		}

		// If provider has credentials configured but no successful backups,
		// count it as a failure (likely authentication error)
		if providerOk == 0 && len(pr.Results.BackupResults) == 0 {
			failed++
		}
	}

	return ok, failed
}
