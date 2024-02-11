package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jonhadfield/githosts-utils"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

const exampleWebHookURL = "https://webhook.example.com"

var testProviderBackupResults = []ProviderBackupResults{
	{
		Provider: "GitHub",
		Results: githosts.ProviderBackupResult{
			githosts.RepoBackupResults{
				Repo:   "https://github.com/jonhadfield/githosts-utils",
				Status: "ok",
				Error:  nil,
			},
			githosts.RepoBackupResults{
				Repo:   "https://github.com/jonhadfield/soba",
				Status: "ok",
				Error:  nil,
			},
		},
	},
}

func (j sobaTime) Add(d time.Duration) sobaTime {
	return sobaTime{
		Time: j.Time.Add(d),
		f:    time.RFC3339,
	}
}

func TestWebhookLongFormat(t *testing.T) {
	defer gock.Off()

	u, err := url.Parse(exampleWebHookURL)
	require.NoError(t, err)

	theTime := sobaTime{
		Time: time.Date(2024, 1, 15, 14, 30, 45, 100, time.UTC),
		f:    time.RFC3339,
	}

	start := theTime.Add(-time.Minute * 20)
	end := theTime.Add(-time.Second * 10)

	json := `{"app":"soba","type":"backups.complete","stats":{"succeeded":2,"failed":0},"timestamp":"2024-01-15T14:30:45Z","data":{"started_at":"2024-01-15T14:10:45Z","finished_at":"2024-01-15T14:30:35Z","results":[{"provider":"GitHub","results":[{"repo":"https://github.com/jonhadfield/githosts-utils","status":"ok"},{"repo":"https://github.com/jonhadfield/soba","status":"ok"}]}]}}`
	gock.New(exampleWebHookURL).
		Post(u.Path).
		MatchHeader("Content-Type", "application/json").
		MatchType("json").
		JSON(json).
		Reply(200)

	gock.Observe(gock.DumpRequest)

	c := retryablehttp.NewClient()

	gock.InterceptClient(c.HTTPClient)

	backupResults := BackupResults{
		StartedAt:  start,
		FinishedAt: end,
		Results:    &testProviderBackupResults,
	}

	require.NoError(t, sendWebhook(c, theTime, backupResults, exampleWebHookURL, ""))
	require.True(t, gock.IsDone())
}

func TestWebhookShortFormat(t *testing.T) {
	t.Log("Testing webhook")

	defer gock.Off()

	u, err := url.Parse(exampleWebHookURL)
	require.NoError(t, err)

	theTime := sobaTime{
		Time: time.Date(2024, 1, 15, 14, 30, 45, 100, time.UTC),
		f:    time.RFC3339,
	}

	start := theTime.Add(-time.Minute * 20)
	end := theTime.Add(-time.Second * 10)

	json := `{"app":"soba","type":"backups.complete","stats":{"succeeded":2,"failed":0},"timestamp":"2024-01-15T14:30:45Z","data":{"started_at":"2024-01-15T14:10:45Z","finished_at":"2024-01-15T14:30:35Z"}}`
	gock.New(exampleWebHookURL).
		Post(u.Path).
		MatchHeader("Content-Type", "application/json").
		MatchType("json").
		JSON(json).
		Reply(200)

	gock.Observe(gock.DumpRequest)

	c := retryablehttp.NewClient()

	gock.InterceptClient(c.HTTPClient)

	backupResults := BackupResults{
		StartedAt:  start,
		FinishedAt: end,
		Results:    &testProviderBackupResults,
	}

	require.NoError(t, sendWebhook(c, theTime, backupResults, exampleWebHookURL, "short"))
	require.True(t, gock.IsDone())
}
