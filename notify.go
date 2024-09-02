package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/hashicorp/go-retryablehttp"
	"gitlab.com/tozd/go/errors"
)

const (
	envSobaNtfyURL    = "SOBA_NTFY_URL"
	envSlackChannelID = "SLACK_CHANNEL_ID"
	envSlackAPIToken  = "SLACK_API_TOKEN"
)

func getResultsErrors(results BackupResults) []errors.E {
	var errs []errors.E

	if results.Results == nil {
		return nil
	}

	for _, providerResults := range *results.Results {
		if providerResults.Results.Error != nil {
			errs = append(errs, providerResults.Results.Error)
		}
	}

	return errs
}

func notify(backupResults BackupResults, succeeded int, failed int) {
	// optimistic create retryable http client
	errs := getResultsErrors(backupResults)

	webHookURL := os.Getenv(envSobaWebHookURL)
	if webHookURL != "" {
		err := sendWebhook(httpClient, sobaTime{
			Time: time.Now(),
			f:    time.RFC3339,
		}, backupResults, os.Getenv(envSobaWebHookURL), os.Getenv(envSobaWebHookFormat))
		if err != nil {
			logger.Printf("error sending webhook: %s", err)
		} else {
			logger.Println("webhook sent")
		}
	}

	ntfyURL := os.Getenv(envSobaNtfyURL)
	if ntfyURL != "" {
		sendNtfy(httpClient, ntfyURL, succeeded, failed, errs)
	}

	slackChannelID := os.Getenv(envSlackChannelID)
	if slackChannelID != "" {
		sendSlackMessage(slackChannelID, succeeded, failed, errs)
	}
}

func sendNtfy(hc *retryablehttp.Client, nURL string, succeeded, failed int, errs []errors.E) {
	nu, err := url.Parse(nURL)
	if err != nil {
		logger.Printf("ntfy failed to parse url: %v", err)

		return
	}

	var req *retryablehttp.Request

	msg := fmt.Sprintf("completed: %d, failed: %d",
		succeeded, failed)

	if len(errs) > 0 && errs[0] != nil {
		msg = fmt.Sprintf("%s\nerror: %s", msg, errs[0].Error())
	}

	req, err = retryablehttp.NewRequest(http.MethodPost, nu.String(),
		strings.NewReader(msg))
	if err != nil {
		logger.Printf("ntfy failed to create request: %v", err)

		return
	}

	switch {
	case succeeded > 0 && failed == 0:
		req.Header.Set("Title", "🚀 soba backups succeeded")
	case failed > 0 && succeeded > 0:
		req.Header.Set("Title", "️⚠️ soba backups completed with errors")
	default:
		req.Header.Set("Title", "️🚨 soba backups failed")
	}

	req.Header.Set("Tags", "soba,backup,git")

	_, err = hc.Do(req)
	if err != nil {
		logger.Printf("error: %s", err)
	}

	logger.Println("ntfy publish sent")
}

func sendSlackMessage(slackChannelID string, succeeded, failed int, errs []errors.E) {
	errorMsgs := make([]string, 0)
	for _, err := range errs {
		if err != nil {
			errorMsgs = append(errorMsgs, err.Error())
		}
	}

	var title string
	switch {
	case succeeded > 0 && failed == 0:
		title = "🚀 soba backups succeeded"
	case failed > 0 && succeeded > 0:

		title = "️⚠️ soba backups completed with errors"
	default:
		title = "️🚨 soba backups failed"
	}

	attachment := slack.Attachment{
		Pretext: fmt.Sprintf("Succeeded: %d, Failed: %d", succeeded, failed),
		Text:    strings.Join(errorMsgs, "\n"),
	}

	api := slack.New(os.Getenv(envSlackAPIToken))

	channelID, timestamp, err := api.PostMessage(
		slackChannelID,
		slack.MsgOptionText(title, false),
		slack.MsgOptionAttachments(attachment),
		slack.MsgOptionAsUser(true),
	)
	if err != nil {
		logger.Println(err.Error())

		return
	}

	logger.Printf("Message successfully sent to channel %s at %s", channelID, timestamp)
}
