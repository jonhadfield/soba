package main

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
	"strings"
	"time"
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

	// send to webhook
	client := c
	client.RetryMax = 3
	client.RetryWaitMin = 1 * time.Second
	client.RetryWaitMax = 3 * time.Second

	var req *retryablehttp.Request
	req, err = retryablehttp.NewRequest(http.MethodPost, url, strings.NewReader(string(o)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// var resp *http.Response
	_, err = client.Do(req)
	if err != nil {
		fmt.Printf("error: %s\n", err)
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
	return j.Time.Format(j.f)
}

func (j sobaTime) MarshalText() ([]byte, error) {
	return []byte(j.format()), nil
}

func (j sobaTime) MarshalJSON() ([]byte, error) {
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
		for _, r := range pr.Results {
			if r.Error != nil {
				failed++

				continue
			}

			ok++
		}
	}

	return ok, failed
}
