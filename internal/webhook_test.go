package internal

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/jonhadfield/githosts-utils/v2"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

const exampleWebHookURL = "https://webhook.example.com"

var testProviderBackupResults = []ProviderBackupResults{
	{
		Provider: "GitHub",
		Results: githosts.ProviderBackupResult{
			BackupResults: []githosts.RepoBackupResults{
				{
					Repo:   "https://github.com/jonhadfield/githosts-utils",
					Status: "ok",
					Error:  nil,
				},
				{
					Repo:   "https://github.com/jonhadfield/soba",
					Status: "ok",
					Error:  nil,
				},
			},
			Error: nil,
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

	json := `{"app":"soba","type":"backups.complete","stats":{"succeeded":2,"failed":0},"timestamp":"2024-01-15T14:30:45Z","data":{"started_at":"2024-01-15T14:10:45Z","finished_at":"2024-01-15T14:30:35Z","results":[{"provider":"GitHub","results":{"BackupResults":[{"repo":"https://github.com/jonhadfield/githosts-utils","status":"ok"},{"repo":"https://github.com/jonhadfield/soba","status":"ok"}],"Error":null}}]}}`
	gock.New(exampleWebHookURL).
		Post(u.Path).
		MatchHeader("Content-Type", "application/json").
		MatchType("json").
		JSON(json).
		Reply(http.StatusOK)

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
		Reply(http.StatusOK)

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

func TestBackupEventType(t *testing.T) {
	tests := []struct {
		name      string
		succeeded int
		failed    int
		want      string
	}{
		{name: "all succeeded", succeeded: 171, failed: 0, want: eventBackupsComplete},
		{name: "some failed", succeeded: 171, failed: 1, want: eventBackupsWithErrors},
		{name: "all failed", succeeded: 0, failed: 12, want: eventBackupsFailed},
		{name: "nothing ran", succeeded: 0, failed: 0, want: eventBackupsComplete},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := backupEventType(tc.succeeded, tc.failed); got != tc.want {
				t.Errorf("backupEventType(%d, %d) = %q, want %q", tc.succeeded, tc.failed, got, tc.want)
			}
		})
	}
}

func TestWithPushStatus(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		succeeded  int
		failed     int
		wantStatus string
		wantMsg    string
	}{
		{
			name:       "healthy run reports up",
			url:        "http://kuma.example/api/push/abc123",
			succeeded:  171,
			failed:     0,
			wantStatus: "up",
			wantMsg:    "succeeded: 171, failed: 0",
		},
		{
			name:       "a single failure reports down",
			url:        "http://kuma.example/api/push/abc123",
			succeeded:  171,
			failed:     1,
			wantStatus: "down",
			wantMsg:    "succeeded: 171, failed: 1",
		},
		{
			name:       "existing query parameters are preserved",
			url:        "http://kuma.example/api/push/abc123?ping=1",
			succeeded:  0,
			failed:     3,
			wantStatus: "down",
			wantMsg:    "succeeded: 0, failed: 3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := withPushStatus(tc.url, tc.succeeded, tc.failed)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("result is not a valid url: %v", err)
			}

			if u.Query().Get("status") != tc.wantStatus {
				t.Errorf("status = %q, want %q", u.Query().Get("status"), tc.wantStatus)
			}

			if u.Query().Get("msg") != tc.wantMsg {
				t.Errorf("msg = %q, want %q", u.Query().Get("msg"), tc.wantMsg)
			}

			if u.Path != "/api/push/abc123" {
				t.Errorf("path was altered: %q", u.Path)
			}
		})
	}

	t.Run("existing parameters survive", func(t *testing.T) {
		got, err := withPushStatus("http://kuma.example/api/push/abc123?ping=1", 1, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		u, _ := url.Parse(got)
		if u.Query().Get("ping") != "1" {
			t.Errorf("pre-existing query parameter was dropped: %q", got)
		}
	})

	t.Run("invalid url is reported", func(t *testing.T) {
		if _, err := withPushStatus("://not-a-url", 1, 0); err == nil {
			t.Error("expected an error for an unparsable url")
		}
	})
}
