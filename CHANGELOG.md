### 1.3.6 release 2025-06-11

- Dependency updates and minor improvements

### 1.3.5 release 2025-06-07

- Environment variables can be loaded from files using the `_FILE` suffix

### 1.3.4 release 2025-05-26

- Introduce delay between GitHub API calls to avoid rate limiting

### 1.3.3 release 2025-05-11

- Minor fixes

### 1.3.2 release 2025-04-27

- Minor fixes

### 1.3.1 release 2025-01-27

- Support cron syntax for backup interval

### 1.2.20 release 2024-10-08

- Add Telegram notifications

### 1.2.19 release 2024-09-02

- Let user define the request timeout

### 1.2.18 release 2024-08-24

- Increase clone timeout to allow for larger repos and slower connections

### 1.2.17 release 2024-07-29

- Performance improvement

### 1.2.16 release 2024-06-09

- Add Slack notifications

### 1.2.15 release 2024-05-09

- Minor fixes and updates

### 1.2.14 release 2024-03-15

- Fix bug introduced in 1.2.13 where daemon exits on run error

### 1.2.13 release 2024-03-15

- Improved error handling to catch and report provider errors
- Return non-zero exit code for runs with failures
- Remove pause after run if not daemonized

### 1.2.12 release 2024-03-13

- Enable limiting GitHub repo backups to user owned

### 1.2.11 release 2024-03-10

- Add support for Azure DevOps respositories

### 1.2.10 release 2024-03-04

- Bugfix for notification error handling

### 1.2.9 release 2024-03-03

- Adds new feature to enable publishing to [ntfy](https://ntfy.sh/) topic on completion

### 1.2.8 release 2024-02-14

- Adds new feature to enable sending webhooks on completion

### 1.2.7 release 2024-01-16

- Improve feedback for invalid BitBucket authentication

### 1.2.6 release 2024-01-11

- Minor fixes and security updates

### 1.2.5 release 2024-01-01

- Dependency updates

### 1.2.4 release 2023-11-11

- Minor updates

### 1.2.3 release 2023-08-25

- Backup interval can now be specified in minutes

### 1.2.2 release 2023-08-19

- Minor output improvements

### 1.2.1 release 2023-08-19

- GitHub user repositories can now be skipped by setting GITHUB_SKIP_USER_REPOS=true

### 1.2.0 release 2023-07-02

- All GitHub Organizations can now be backed up by specifying * instead of individual names
- GitLab API calls will now be retried if they initially fail

### 1.1.12 released 2023-06-25

- Add support for Gitea

### 1.1.11 released 2023-03-26

- Fix goreleaser to build and distribute docker release

### 1.1.10 released 2023-03-26

- Notarize binaries produced for MacOS to remove unknown developer warning
- note: missing docker release

### 1.1.9 released 2023-03-11

- Improve refs comparison mode

### 1.1.8 released 2023-03-11

- Fixes edge case where refs with spaces returned an error and forced clone in when in refs mode

### 1.1.7 released 2023-03-10

- Fix for refs comparison for BitBucket
- Minor logging improvements

### 1.1.6 released 2023-03-05

- Fixes bug where only first 100 GitHub Organisation repositories are backed up
- Introduce retries for BitBucket API calls and cloning

### 1.1.5 released 2022-12-20

- Maintenance release

### 1.1.4 released 2022-11-12

- Adds new feature to prevent having to clone a repository before comparing with the latest local backup.
- Some minor tweaks and output improvements.

### 1.1.3 released 2022-10-12

Fixes issues that resulted in only a subset of GitLab Projects being backed up.  
All Projects across GitLab will now be returned, based on the user's minimum access level. The default level is 'Reporter' and can be overriden by setting environment variable:
`GITLAB_PROJECT_MIN_ACCESS_LEVEL` to the numeric value associated with the level shown [here](https://docs.gitlab.com/ee/api/members.html#valid-access-levels).  
Thanks to [@drummingdemon](https://github.com/drummingdemon) for all their help in testing this release.

### 1.1.2 released 2022-06-03

[Add feature](https://github.com/jonhadfield/soba/issues/9) to enable backup of project repos in GitLab groups

### 1.1.1 released 2022-03-13

[Add feature](https://github.com/jonhadfield/soba/issues/7) to enable backup of GitHub organisations' repositories

### 1.1.0 released 2021-10-27

Resolve exit on backup failure issue
