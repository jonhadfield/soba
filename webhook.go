package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

func sendWebhook(c *retryablehttp.Client, sendTime sobaTime, results BackupResults, url, format string) error {
	ok, failed := getBackupsStats(results)

	if sendTime.IsZero() {
		sendTime = sobaTime{
			Time: time.Now(),
			f:    time.RFC3339,
		}
	}

	webhookData := WebhookData{
		App:       appName,
		Type:      "backups.complete",
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

	// o, err := json.MarshalIndent(webhookData, "", "  ")

	o, err := json.Marshal(webhookData)
	if err != nil {
		return fmt.Errorf("error marshalling webhook data: %w", err)
	}

	// send to webhook
	c.RetryMax = webhookRetryMax
	c.RetryWaitMin = webhookRetryWaitMin
	c.RetryWaitMax = webhookRetryWaitMax

	var req *retryablehttp.Request

	req, err = retryablehttp.NewRequest(http.MethodPost, url, strings.NewReader(string(o)))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		fmt.Printf("error: %s\n", err)
	}

	defer resp.Body.Close()

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

		for _, r := range pr.Results.BackupResults {
			// catch error from repository backup
			if r.Error != nil {
				failed++

				continue
			}

			ok++
		}
	}

	return ok, failed
}
