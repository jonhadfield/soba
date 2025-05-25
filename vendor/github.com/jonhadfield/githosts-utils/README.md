# githosts-utils

A Go library that simplifies backing up repositories from several popular Git hosting providers. The package is used by [soba](https://github.com/jonhadfield/soba) and can be embedded in other tools or used directly.

## Features

- Minimal dependencies and portable code
- Supports GitHub, GitLab, Bitbucket, Azure DevOps and Gitea
- Repositories are cloned using `git --mirror` and stored as timestamped bundle files
- Optional reference comparison to skip cloning when refs have not changed
- Ability to keep a configurable number of previous bundles
- Pluggable HTTP client and simple logging control via the `GITHOSTS_LOG` environment variable

## Installation

```bash
go get github.com/jonhadfield/githosts-utils
```

The library requires Go 1.22 or later.

## Quick Start

Create a host for the provider you wish to back up and call `Backup()` on it. Each provider has an input struct with the required options. The example below backs up a set of GitHub repositories:

```go
package main

import (
    "log"
    "os"

    "github.com/jonhadfield/githosts-utils"
)

func main() {
    backupDir := "/path/to/backups"

    host, err := githosts.NewGitHubHost(githosts.NewGitHubHostInput{
        Caller:    "example",
        BackupDir: backupDir,
        Token:     os.Getenv("GITHUB_TOKEN"),
    })
    if err != nil {
        log.Fatal(err)
    }

    results := host.Backup()
    for _, r := range results.BackupResults {
        log.Printf("%s: %s", r.Repo, r.Status)
    }
}
```

`Backup()` returns a `ProviderBackupResult` which contains the status for each repository. Bundles are written beneath `<backupDir>/<provider>/<owner>/<repo>/`.

### Diff Remote Method

Each host accepts a `DiffRemoteMethod` value of either `"clone"` or `"refs"`:

- `clone` (default) – always clone and create a new bundle.
- `refs` – fetch remote references first and skip cloning when the refs match the latest bundle.

### Retaining Bundles

Set `BackupsToRetain` to keep only the most recent _n_ bundle files per repository. Older bundles are automatically deleted after a successful backup.

## Environment Variables

The library reads the following variables where relevant:

- `GITHOSTS_LOG` – set to `debug` to emit verbose logs.
- `GIT_BACKUP_DIR` – used by the tests to determine the backup location.

Provider specific tests require credentials via environment variables such as `GITHUB_TOKEN`, `GITLAB_TOKEN`, `BITBUCKET_KEY`, `BITBUCKET_SECRET`, `AZURE_DEVOPS_USERNAME`, `AZURE_DEVOPS_PAT` and `GITEA_TOKEN`.

## Running Tests

```bash
export GIT_BACKUP_DIR=$(mktemp -d)
go test ./...
```

Integration tests are skipped unless the corresponding provider credentials are present.

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
