# soba

> Automated, encrypted backups for your Git repositories

<img src="docs/soba.png" alt="logo" width="200"/>

[![GitHub Release][release-img]][release]
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/1bd46b99467c45d99e4903b44a16f874)](https://app.codacy.com/gh/jonhadfield/soba/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)
[![CodeQL](https://github.com/jonhadfield/soba/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/jonhadfield/soba/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jonhadfield/soba)](https://goreportcard.com/report/github.com/jonhadfield/soba)

soba backs up your Git repositories from GitHub, GitLab, Bitbucket, Azure DevOps, Gitea, and Sourcehut. Each repository is saved as a single-file [git bundle](https://git-scm.com/book/en/v2/Git-Tools-Bundling), and only new bundles are stored when changes are detected. Bundles can optionally be encrypted with [age](https://age-encryption.org/) for secure offsite storage.

## Features

| | |
|---|---|
| **Multi-provider** | GitHub, GitLab, Bitbucket, Azure DevOps, Gitea, Sourcehut |
| **Efficient storage** | Git bundles with change detection &mdash; unchanged repos are skipped |
| **Encryption** | Optional [age encryption](https://age-encryption.org/) for bundles, manifests, and LFS archives |
| **Built-in scheduler** | Interval (`24h`, `45m`) or cron (`0 3 * * *`) scheduling |
| **Smart rotation** | Keep only the _n_ most recent backups per repo |
| **Git LFS** | Back up large file storage objects alongside repo bundles |
| **Notifications** | Slack, Telegram, webhooks, and [ntfy](https://ntfy.sh/) alerts |
| **Runs anywhere** | Binary, Docker, Kubernetes, or Synology NAS |

## Quick Start

Create git bundles of all repositories in your GitHub account:

```bash
mkdir soba-backups
docker run --rm \
  -v ./soba-backups:/backups \
  -e GIT_BACKUP_DIR=/backups \
  -e GITHUB_TOKEN=<your-token> \
  ghcr.io/jonhadfield/soba
```

## Installation

### Binary

Download the latest release from the [releases page](https://github.com/jonhadfield/soba/releases), then:

```bash
install <soba binary> /usr/local/bin/soba
```

Set `GIT_BACKUP_DIR` and your [provider credentials](docs/providers.md#provider-reference), then run:

```bash
soba
```

### Docker

```bash
docker run --rm -t \
  -v /path/to/backups:/backup \
  -e GIT_BACKUP_DIR=/backup \
  -e GITHUB_TOKEN=$GITHUB_TOKEN \
  -e GITLAB_TOKEN=$GITLAB_TOKEN \
  ghcr.io/jonhadfield/soba
```

### Kubernetes

Deploy soba as a CronJob. See the [Kubernetes guide](kubernetes/README.md) for manifests and instructions.

### Synology NAS

Run soba via the Docker GUI on your NAS. See the [Synology guide](docs/providers.md#run-on-synology-nas) for step-by-step instructions.

## Supported Providers

| Provider | Token Docs | Key Variables |
|:---------|:-----------|:--------------|
| [GitHub](docs/providers.md#github) | [Create token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#personal-access-tokens-classic) | `GITHUB_TOKEN` |
| [GitLab](docs/providers.md#gitlab) | [Create token](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html) | `GITLAB_TOKEN` |
| [Bitbucket](docs/providers.md#bitbucket) | [API tokens](https://id.atlassian.com/manage-profile/security/api-tokens) / [OAuth2](https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/) | `BITBUCKET_EMAIL` + `BITBUCKET_API_TOKEN` |
| [Azure DevOps](docs/providers.md#azure-devops) | [Create PAT](https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate?view=azure-devops) | `AZURE_DEVOPS_USERNAME` + `AZURE_DEVOPS_PAT` + `AZURE_DEVOPS_ORGS` |
| [Gitea](docs/providers.md#gitea) | [Create token](https://docs.gitea.com/development/api-usage#generating-and-listing-api-tokens) | `GITEA_APIURL` + `GITEA_TOKEN` |
| [Sourcehut](docs/providers.md#sourcehut) | [Create PAT](https://man.sr.ht/accounts.md#api) | `SOURCEHUT_PAT` |

For full provider configuration, organisation filtering, comparison modes, and self-hosted endpoints, see the [provider documentation](docs/providers.md).

## Configuration

All configuration is via environment variables. Set `GIT_BACKUP_DIR` for the backup destination and add credentials for each provider you want to back up.

```bash
export GIT_BACKUP_DIR="/repo-backups/"
export GITHUB_TOKEN="ghp_..."
```

Secrets can also be loaded from files using the `_FILE` suffix:

```bash
export GITHUB_TOKEN_FILE=/run/secrets/github_token
```

If both the variable and `_FILE` version are set, the variable takes precedence.

## Scheduling

soba includes a built-in scheduler so it can run continuously. Set an interval or cron expression:

```bash
# Run every 24 hours
export GIT_BACKUP_INTERVAL=24h

# Run every 45 minutes
export GIT_BACKUP_INTERVAL=45m

# Run daily at 3am (cron syntax)
export GIT_BACKUP_CRON='0 3 * * *'
```

soba can also be triggered by external schedulers like cron or systemd. See the [logging and persistence guide](docs/providers.md#logging) for cron examples.

## Backup Rotation

Keep only the _n_ most recent backups per provider by setting the relevant variable:

```bash
export GITHUB_BACKUPS=7
export GITLAB_BACKUPS=7
export BITBUCKET_BACKUPS=7
export GITEA_BACKUPS=7
export AZURE_DEVOPS_BACKUPS=7
export SOURCEHUT_BACKUPS=7
```

## Git LFS

To include Git LFS objects in your backups, enable it per provider:

```bash
export GITHUB_BACKUP_LFS=yes
export GITLAB_BACKUP_LFS=yes
```

LFS content is stored in a `*.lfs.tar.gz` file alongside the repository bundle. The Docker image includes `git-lfs`.

## Encryption

Encrypt bundles, manifests, and LFS archives with [age encryption](https://age-encryption.org/) by setting a passphrase:

```bash
export BUNDLE_PASSPHRASE="your-secure-passphrase"
```

When enabled:
- Bundles are saved as `.bundle.age`
- Manifests are saved as `.manifest.age`
- LFS archives are saved as `.lfs.tar.gz.age`

Store the passphrase securely &mdash; without it, your backups cannot be decrypted.

### Decrypting backups

Install the [age CLI](https://github.com/FiloSottile/age/releases), then:

```bash
age -d -o repo.bundle repo.bundle.age
```

You'll be prompted for the passphrase. For batch decryption:

```bash
#!/bin/bash
read -s -p "Enter passphrase: " PASSPHRASE
echo

for file in *.age; do
    output="${file%.age}"
    echo "Decrypting $file to $output..."
    echo "$PASSPHRASE" | age -d -o "$output" "$file"
done
```

## Notifications

Get notified when backups complete or fail. To reduce noise on scheduled runs, send notifications only on failure:

```bash
export SOBA_NOTIFY_ON_FAILURE_ONLY=true
```

| Channel | Variables |
|:--------|:----------|
| **Slack** | `SLACK_CHANNEL_ID`, `SLACK_API_TOKEN` |
| **Telegram** | `SOBA_TELEGRAM_BOT_TOKEN`, `SOBA_TELEGRAM_CHAT_ID` |
| **Webhooks** | `SOBA_WEBHOOK_URL`, `SOBA_WEBHOOK_FORMAT` (`long` or `short`) |
| **ntfy** | `SOBA_NTFY_URL` |

Webhook payload examples: [long format](examples/webhook.json), [short format](examples/webhook-short.json).

## Restoring Backups

A git bundle is a portable archive of a repository. Clone it like any remote:

```bash
git clone soba.20180708153107.bundle my-repo
cd my-repo
git remote set-url origin <original-repo-url>
```

If the bundle is encrypted, [decrypt it first](#decrypting-backups).

## Supported Platforms

Tested on Windows 10, macOS, and Linux (amd64).
Should also work on Linux (386, arm, arm64), FreeBSD, NetBSD, and OpenBSD.

## Changelog

See [CHANGELOG.md](docs/CHANGELOG.md) for release history.

[release]: https://github.com/jonhadfield/soba/releases
[release-img]: https://img.shields.io/github/release/jonhadfield/soba.svg?logo=github
