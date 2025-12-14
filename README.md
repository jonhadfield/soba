# soba

> 🔒 **Reliable, automated backup solution for your Git repositories**

<img src="docs/soba.png" alt="logo" width="200"/>

Soba ensures your code is never lost by automatically backing up repositories from GitHub, GitLab, Bitbucket, and other major Git hosting providers. It creates space-efficient git bundles, detecting and storing only changed repositories to minimize storage usage.

**Key features:**
- 🌐 **Multi-provider support** - Back up from GitHub, GitLab, Bitbucket, Azure DevOps, Gitea, and Sourcehut
- 💾 **Efficient storage** - Only stores new bundles when changes are detected
- 🔒 **Encryption support** - Optionally encrypt bundles with age encryption for security
- ⏰ **Built-in scheduler** - Run backups automatically at custom intervals
- 🔄 **Smart rotation** - Keep only the backups you need
- 📦 **Git LFS support** - Back up large file storage objects
- 📢 **Notifications** - Get alerts via Slack, Telegram, webhooks, and more
- 🐳 **Runs anywhere** - Deploy as a binary, Docker container, or on Kubernetes

---

[![GitHub Release][release-img]][release]
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/1bd46b99467c45d99e4903b44a16f874)](https://app.codacy.com/gh/jonhadfield/soba/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)[![CodeQL](https://github.com/jonhadfield/soba/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/jonhadfield/soba/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jonhadfield/soba)](https://goreportcard.com/report/github.com/jonhadfield/soba)

- [about](#about)
- [configuration](#configuration)
- [run using the binary](#run-using-the-binary)
- [run with Docker](#run-with-docker)
- [run on Synology NAS](#run-on-synology-nas)
- [run on Kubernetes](kubernetes/README.md)
- [scheduling backups](#scheduling-backups)
- [rotating backups](#rotating-backups)
- [git lfs backups](#git-lfs-backups)
- [encryption](#encryption)
- [notifications](#notifications)
- [logging](#logging)
- [setting provider credentials](#setting-provider-credentials)
- [additional options](#additional-options)
- [restoring backups](#restoring-backups)

## about

soba is tool for backing up private and public git repositories hosted on the
most popular [hosting providers](#supported-providers). It generates a [git bundle](https://git-scm.com/book/en/v2/Git-Tools-Bundling) that stores a backup of each repository as a single file.

As unchanged git repositories create identical bundle files, new bundles will only be stored if changes to the repository have been made. This can be done by re-cloning each repository every time soba runs, or by [comparing refs without cloning](#comparing-remote-repository-with-local-backup).

soba includes its [own scheduler](#scheduling-backups) that triggers it to run every specified number of hours, or it can be run with other schedulers such as cron.

## quick start
soba can [run as a binary](#run-using-the-binary) or [using docker](#run-using-docker) with the prebuilt image distributed with each release.
For example, the following will create git bundles of all repositories in your GitHub user's account in the soba-backups directory:

```
$ mkdir soba-backups
$ docker run --rm -v ./soba-backups:/backups -e GITHUB_TOKEN=<token-here> -e GIT_BACKUP_DIR=/backups jonhadfield/soba:latest
```

## latest updates

### 1.3.9 release 2025-07-05

- Add Sourcehut provider support
- Improve clone reliability and error handling
- Increase default HTTP request timeout to 10 minutes

### 1.3.8 release 2025-06-29

- Fix Docker image reference

### 1.3.7 release 2025-06-26
- Add Sourcehut as a supported provider

- Add support for Git LFS
- Changes BitBucket auth to API Keys (OAuth2 will be supported also in next release)

See full changelog [here](docs/CHANGELOG.md).

## supported OSes

Tested on Windows 10, MacOS, and Linux (amd64).
Not tested, but should also work on builds for: Linux (386, arm386 and arm64), FreeBSD, NetBSD, and OpenBSD.

## supported providers

- Azure DevOps
- BitBucket
- Gitea
- GitHub
- GitLab
- Sourcehut

## configuration

soba can be run from the command line or as a container. In both cases the only configuration required is an
environment variable with the directory in which to create backups, and additional to define credentials for each the
providers.

On Windows 10:

- search for 'environment variables' and choose 'Edit environment variables for your account'
- choose 'New...' under the top pane and enter the name/key and value for each of the settings

On Linux and MacOS you would set these using:

```bash
export GIT_BACKUP_DIR="/repo-backups/"
```

You can also source values from files by specifying the same variable name with
the suffix `_FILE`. soba will read the contents of the referenced file and use
that as the value. For example:

```bash
export GIT_BACKUP_DIR_FILE=/run/secrets/backup_dir
```

If both the variable and `_FILE` version are set, the variable value takes
precedence.

To set provider credentials see [below](#setting-provider-credentials).

## run using the binary

Download the latest release [here](https://github.com/jonhadfield/soba/releases) and then install:

```bash
install <soba binary> /usr/local/bin/soba
```

After setting `GIT_BACKUP_DIR`, set your provider token(s) as detailed [here](#setting-provider-credentials).

and then run:

```bash
soba
```

## run with Docker

Using Docker enables you to run soba without anything else installed.

Docker requires you pass environment variables to the container using the '-e' option and that you mount your preferred
backup directory. For example:

```bash
docker run --rm -t \
             -v <your backup dir>:/backup \
             -e GIT_BACKUP_DIR='/backup' \
             -e GITHUB_TOKEN='MYGITHUBTOKEN' \
             -e GITLAB_TOKEN='MYGITLABTOKEN' \
             ghcr.io/jonhadfield/soba
```

To hide credentials, you can instead use exported environment variables and specify using this syntax:

```bash
docker run --rm -t \
             -v <your backup dir>:/backup \
             -e GIT_BACKUP_DIR='/backup' \
             -e GITHUB_TOKEN=$GITHUB_TOKEN \
             -e GITLAB_TOKEN=$GITLAB_TOKEN \
             ghcr.io/jonhadfield/soba
```

## run on Kubernetes
For instructions on how to run soba on Kubernetes, see [here](kubernetes/README.md).

## scheduling backups

Backups can be scheduled to run by setting an interval or by using a cron syntax.

### interval syntax
Environment variable: `GIT_BACKUP_INTERVAL` can be specified in hours or minutes. For example, this will run the backup daily:

```bash
export GIT_BACKUP_INTERVAL=24h
```

and this will run the backup every 45 minutes:

```bash
export GIT_BACKUP_INTERVAL=45m
```

note: if you don't specify the trailing 'm' or 'h' then hours are assumed.

### cron syntax
Alternatively, you can schedule backups using a cron syntax. For example, to run every day at 3am:

```bash
export GIT_BACKUP_CRON='0 3 * * *'
```

## rotating backups

A new bundle is created every time a change is detected in the repository. To keep only the _x_ most recent, use the
following provider specific environment variables:
`GITEA_BACKUPS=x`
`GITHUB_BACKUPS=x`
`GITLAB_BACKUPS=x`
`BITBUCKET_BACKUPS=x`

## git lfs backups

To back up Git LFS objects, set the environment variable for your provider to `y` or `yes`:
`GITHUB_BACKUP_LFS`, `GITLAB_BACKUP_LFS`, `GITEA_BACKUP_LFS`, `BITBUCKET_BACKUP_LFS`, `SOURCEHUT_BACKUP_LFS`,
and `AZURE_DEVOPS_BACKUP_LFS`.
When enabled, soba stores LFS content in a `*.lfs.tar.gz` file alongside the repository bundle.
The provided Docker image already includes `git-lfs`.

## encryption

soba supports encrypting git bundles and manifests using [age encryption](https://age-encryption.org/). When enabled, backups are stored with the `.age` extension and are protected by a passphrase.

### enabling encryption

To enable encryption, set the `BUNDLE_PASSPHRASE` environment variable:

```bash
export BUNDLE_PASSPHRASE="your-secure-passphrase"
```

When this environment variable is set:
- Git bundles are encrypted and saved with `.bundle.age` extension instead of `.bundle`
- Manifest files are encrypted and saved with `.manifest.age` extension instead of `.manifest`
- LFS archives are encrypted and saved with `.lfs.tar.gz.age` extension instead of `.lfs.tar.gz`

### using with Docker

```bash
docker run --rm -t \
             -v <your backup dir>:/backup \
             -e GIT_BACKUP_DIR='/backup' \
             -e GITHUB_TOKEN=$GITHUB_TOKEN \
             -e BUNDLE_PASSPHRASE='your-secure-passphrase' \
             ghcr.io/jonhadfield/soba
```

### security considerations

- Store the passphrase securely - without it, your backups cannot be decrypted
- Consider using the `_FILE` pattern to read the passphrase from a secure file:
  ```bash
  export BUNDLE_PASSPHRASE_FILE=/run/secrets/bundle_passphrase
  ```
- Encrypted backups are protected with age encryption, which uses modern cryptographic standards

### restoring encrypted backups

Encrypted bundles must be decrypted before they can be used. First, install the `age` command-line tool:

#### installing age

**On macOS:**
```bash
brew install age
```

**On Ubuntu/Debian:**
```bash
sudo apt install age
```

**On other systems:**
Download from the [age releases page](https://github.com/FiloSottile/age/releases)

#### decrypting files

**Decrypt a git bundle:**
```bash
age -d -o repository.bundle repository.bundle.age
```
You'll be prompted to enter the passphrase used during encryption.

**Decrypt a manifest file:**
```bash
age -d -o repository.manifest repository.manifest.age
```

**Decrypt an LFS archive:**
```bash
age -d -o repository.lfs.tar.gz repository.lfs.tar.gz.age
```

#### restoring from decrypted bundle

Once decrypted, you can clone the bundle as a normal git repository:

```bash
git clone repository.bundle repository
cd repository
git remote set-url origin <original-repo-url>
```

#### batch decryption

To decrypt multiple files at once, you can use a script:

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

## setting the request timeout

By default, soba will wait up to ten minutes for a response to complete. This could be anything from an API call to discover repositories to a clone of a large repository.
If you have a slow connection or very large repositories, you may want to increase this. To do so, set the environment variable `GIT_REQUEST_TIMEOUT` to the number of seconds you wish to wait. For example, to wait up to ten minutes:
```bash
export GIT_REQUEST_TIMEOUT=600
```

## notifications

### Notification Control

By default, soba sends notifications on every backup run. To reduce noise when running on a schedule, you can configure soba to only send notifications when backups fail or complete with errors:

```bash
export SOBA_NOTIFY_ON_FAILURE_ONLY=true
```

When enabled, successful backup runs will skip all notifications (Telegram, Slack, webhooks, and ntfy).

### Telegram
*(since release 1.2.20)*
To send a Telegram message on completion, set the environment variables:
`SOBA_TELEGRAM_BOT_TOKEN` with the bot token
`SOBA_TELEGRAM_CHAT_ID` with the chat/group id

To get the bot token:
- send a message to @BotFather of /newbot
- submit a name, e.g. soba-notifier
- submit a username for the bot
- record bot token

To get the chat id:
- add the bot user to the group (get group info and click Add)
- run command:`curl -s -X POST https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
- record the chat id in the response

### Slack
*(since release 1.2.16)*
To send a Slack message on completion, set the environment variables:
`SLACK_CHANNEL_ID` with the channel id
`SLACK_API_TOKEN` with the token for the Slack app
For example:
`$ export SLACK_CHANNEL_ID=C12345678`
`$ export SLACK_API_TOKEN=xoxb-***********-************-************************`

#### note
- channel id can be in `About` section at bottom of the channel details
- the app needs to be added under `Apps` in the `Integrations` section of channel details
- use the token starting with `xoxb-` and not the one starting with `xoxp-`

### webhooks
*(since release 1.2.8)*
To send a webhook on completion of a run: set the environment variable `SOBA_WEBHOOK_URL` with the url of the endpoint.
For example:
`$ export SOBA_WEBHOOK_URL=https://api.example.com/webhook`

#### webhook payload
The payload is a JSON document containing details of the backup run.  The default format lists each repository and the success or failure of its backup.  You can see an example [here](examples/webhook.json).
For a shorter format, with just stats on the success and failure counts, use the environment variable `SOBA_WEBHOOK_FORMAT`.
For example:
`$ export SOBA_WEBHOOK_FORMAT=short`
You can see a sample [here](examples/webhook-short.json).
*The default format (if not specified) is `long`*

> NOTE: The long format webhook will contain a list of your repos and, if there's an error, may contain other details including URLs. Please keep this in mind when sending to endpoints that may be insecure.

### ntfy
*(since release 1.2.10)*
ntfy is a popular service that enables push notifications for desktop and mobile apps.
To send a message on completion of a run: set the environment variable `SOBA_NTFY_URL` with the url of the endpoint.
For example:
`$ export SOBA_NTFY_URL=https://ntfy.sh/example-topic`

## logging

### persistence

Messages are written to stdout and can be persisted by directing to a file, e.g.
`soba > soba.log`

#### logging to /var/log/soba

create a user called soba:
`sudo adduser soba`
create a log directory:
`sudo mkdir /var/log/soba`
set user permissions:
`sudo chown soba /var/log/soba && sudo chmod 700 /var/log/soba`
switch to soba user:
`sudo su - soba`
run soba and direct output:
`soba > /var/log/soba/soba.log`

### rotation

[Logrotate](https://linux.die.net/man/8/logrotate) is a utility that comes with most Linux distributions and removes and/or compresses messages older than a certain number of hours or days.
This example assumes you persist the log file to /var/log/soba/soba.log
create a file in /etc/logrotate.d/soba with the following content:

    /var/log/soba/soba.log {
      rotate 7      # remove backups older than seven days
      daily         # process log file each day
      compress      # compress the backup
      copytruncate  # don't delete the file after backup, but instread truncate
    }

Each day, this copy the latest logs to a new file that is then compressed. The existing log file is then truncated. Any backups older than seven days are then removed.
### log level
Set `SOBA_LOG` to a number to control verbosity. Higher values increase output.


### keep running after reboot

In case the computer is rebooted or the process ends for another reason, you can ensure it automatically restarts with a simple script and cron job.

#### script

For example:

    #!/bin/bash -e
    export GIT_BACKUP_DIR=/backup-dir
    export GITHUB_TOKEN=xxxxxxx   # avoid hard-coding if possible
    export GITHUB_BACKUPS=7
    export GIT_BACKUP_INTERVAL=12
    export GITHUB_COMPARE=refs
    /usr/local/bin/soba

#### cron job

ensure the user running soba has an entry in `/etc/cron.allow`.

run `crontab -e`

add the following (assuming you have a user called soba with a script to run it called backup in their home directory):
`* * * * * /usr/bin/flock -n /tmp/soba.lockfile /home/soba/backup >> /var/log/soba/soba.log 2>&1`

note: A useful tool for testing cron jobs is [crontab guru](https://crontab.guru/).

## setting provider credentials

On Linux and MacOS you can set environment variables manually before each time you run soba:

```bash
export NAME='VALUE'
```

or by defining in a startup file for your shell so they are automatically set and available when you need them. For
example, if using the bash shell and running soba as your user, add the relevant export statements to the following
file:

```bash
/home/<your-user-id>/.bashrc
```

and run:

```bash
source /home/<your-user-id>/.bashrc
```

| Provider  | Environment Variable(s)         | Generating token                                                                                         |
|:----------|:--------------------------------|:---------------------------------------------------------------------------------------------------------|
| Azure DevOps | AZURE\_DEVOPS\_USERNAME      | [instructions](https://learn.microsoft.com/en-us/azure/devops/organizations/accounts/use-personal-access-tokens-to-authenticate?view=azure-devops)       |
|           | AZURE\_DEVOPS\_PAT              |                                                                                                     |
|           | AZURE\_DEVOPS\_ORGS             |                                                                                                          |
|           | AZURE_DEVOPS_BACKUPS            |
| BitBucket (API tokens)  | BITBUCKET_EMAIL   | [instructions](https://id.atlassian.com/manage-profile/security/api-tokens)         |
|           | BITBUCKET_API_TOKEN             |                                                                                                          |
| BitBucket (OAuth2)      | BITBUCKET_USER    | [instructions](https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/)         |
|           | BITBUCKET_KEY                   |                                                                                                          |
|           | BITBUCKET_SECRET                |                                                                                                          |                                                                                          |
| Gitea     | GITEA_APIURL                    | [instructions](#gitea-instructions) |
|           | GITEA_TOKEN                     | |
| GitHub    | GITHUB_TOKEN                    | [instructions](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#personal-access-tokens-classic) |
| GitLab    | GITLAB_TOKEN                    | [instructions](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)                                       |
|           | GITLAB\_PROJECT\_MIN\_ACCESS\_LEVEL | [instructions](https://docs.gitlab.com/ee/user/permissions.html)                                       |
| Sourcehut | SOURCEHUT_PAT                   | [instructions](https://man.sr.ht/accounts.md#api) |
|           | SOURCEHUT_APIURL                | |
|           | SOURCEHUT_BACKUPS               | |
|           | SOURCEHUT_BACKUP_LFS               | |

You can now also provide these credentials via files using the *_FILE environment variable pattern. For example:

```bash
GITHUB_TOKEN_FILE=/run/secrets/my_github_token
```

If both the variable and *_FILE are set, the variable takes precedence.

## additional options

### Azure DevOps

#### Returning Organisations' repositories (available since soba 1.2.11)

An organisation must be specified using environment variable AZURE\_DEVOPS\_ORGS in order for soba to discover the projects and their repos.
_Note: Only a single organisation is currently supported._

#### Repo/Bundle comparison method

Environment variable: AZURE\_DEVOPS\_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                                |
|:----------------|:---------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                     |
| refs            | Compare refs without downloading (available since soba 1.1.4)  |


### BitBucket

#### Repo/Bundle comparison method

Environment variable: BITBUCKET_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                                |
|:----------------|:---------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                     |
| refs            | Compare refs without downloading (available since soba 1.1.4)  |


To use Bitbucket Server or another custom endpoint, set `BITBUCKET_APIURL` with the API URL.

When using API tokens as your auth method, only the following scopes are required:
- `read:project:bitbucket`
- `read:repository:bitbucket`

### Gitea

#### Gitea instructions

[Official documentation](https://docs.gitea.com/development/api-usage#generating-and-listing-api-tokens)

The value for GITEA_APIURL needs to be in the format: https://[domain]/api/v1, where domain is something like gitea.example.com.

GITEA_TOKEN is the secret you need to generate using the API (see official documentation above), or via the web GUI:

- Login to Gitea
- Select your user icon in the top right-hand corner and choose `Settings` from the dropdown
- Select `Applications`
- Enter a Token Name, e.g. "soba backups"
- Select `Public only` or `All` depending on use-case
- Expand the `Select permissions` menu
- Select `read:organization` and `read:repository`.
- Click on `Generate Token` and the value will appear at the top of the page

#### Returning Organisations' repositories

Repositories in Gitea organisations are not backed up by default. To back these up, specify a comma separated
list of organisations in the environment variable: GITEA_ORGS. To include "all" organisations, set to `*`.

#### Gitea Repo/Bundle comparison method

Environment variable: GITEA_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                               |
|:----------------|:--------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                    |
| refs            | Compare refs without downloading (available since soba 1.1.4) |


### GitHub

#### Returning Organisations' repositories

Repositories in GitHub organisations are not backed up by default. To back these up, specify a comma separated
list of organisations in the environment variable: GITHUB_ORGS.

#### Skipping User repository backups

By default, all users' repositories will be backed up, even when specifying organisations.
To skip user repositories set environment variable: GITHUB\_SKIP\_USER\_REPOS to true.

#### Limit user repo backups to those owned by the user

By default, all repositories a user is affiliated with, e.g. a collaborator on, are included for backup.
To limit these to only those owned by the user, set environment variable: GITHUB\_LIMIT\_USER\_OWNED to true.

#### GitHub Repo/Bundle comparison method

Environment variable: GITHUB_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                               |
|:----------------|:--------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                    |
| refs            | Compare refs without downloading (available since soba 1.1.4) |
#### Adjust GitHub call behaviour

Environment variables:
- `GITHUB_CALL_SIZE` - number of repositories returned per API call (default 100)
- `GITHUB_WORKER_DELAY` - delay in milliseconds between API workers starting (default 500)

To use GitHub Enterprise or other API endpoints, set `GITHUB_APIURL`.

### GitLab

#### filtering Projects by access level (available since soba 1.1.3)

The way in which a user's GitLab Projects are returned. By default, every Project a user has at
least `Reporter` access to will be returned. New environment variable GITLAB\_PROJECT\_MIN\_ACCESS\_LEVEL can be set to
override this, by specifying the number matching the desired access level shown [here](https://docs.gitlab.com/ee/api/members.html#valid-access-levels) and here:

| Access Level | Value |
|:-------------|:------|
| Guest        | 10    |
| Reporter     | 20    |
| Developer    | 30    |                                                                                          |
| Maintainer   | 40    |
| Owner        | 50    |

#### GitLab Repo/Bundle comparison method

Environment variable: GITLAB_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                               |
|:----------------|:--------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                    |
| refs            | Compare refs without downloading (available since soba 1.1.4) |
To use a self-hosted GitLab instance, set `GITLAB_APIURL` with the API URL.
### Sourcehut

#### Repo/Bundle comparison method

Environment variable: SOURCEHUT_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |   |
|:----------------|:------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle |
| refs            | Compare refs without downloading |

To use a custom Sourcehut instance, set `SOURCEHUT_APIURL` with the API URL.


### Comparing remote repository with local backup

By default, each repository will be cloned, bundled, and that bundle compared with the latest local bundle to check if it should be kept or discarded.
When processing many large repositories, this can be a lengthy process.
Alternatively, you can now compare the Git refs of the latest local bundle with the remote repository without having to clone.
This is carried out using native commands `git bundle list-heads <bundle file>` and `git ls-remote <remote repository>`.
This process is far quicker than cloning but should only be used if the following is understood: Comparing refs means comparing the tips of, and not the entire history of, the repository. [This post on Stack Overflow](https://stackoverflow.com/questions/74281792/git-comparing-local-bundle-with-remote-repository-using-refs-only) goes into additional detail.

### run on Synology NAS

#### _The following was tested on DS916+_

1. Create a directory on your NAS for backing up Git repositories to
2. Install Docker from the Synology Package Center
3. Open Docker and select 'Image'
4. Select 'Add' from the top menu and choose 'Add From URL'
5. In 'Repository URL' enter 'jonhadfield/soba', leave other options as default and click 'Add'
6. When it asks to 'Choose Tag' accept the default 'latest' by pressing 'Select'
7. Select image 'jonhadfield/soba:latest' from the list and click 'Launch' from the top menu
8. Set 'Container Name' to 'soba' and select 'Advanced Settings'
9. Check 'Enable auto-restart'
10. Under 'Volume' select 'Add folder' and choose the directory created in step 1. Set the 'Mount Path' to '/backup'
11. Under 'Network' check 'Use the same network as Docker Host'
12. Under 'Environment' click '+' to add the common configuration:
    - **variable** GIT\_BACKUP\_DIR **Value** /backup
    - **variable** GIT\_BACKUP\_INTERVAL **Value** (hours between backups)
13. Also under 'Environment' click '+' to add the relevant provider specific configuration:
    - **variable** AZURE_DEVOPS_USERNAME **Value**
    - **variable** AZURE_DEVOPS_PAT **Value**
    - **variable** AZURE_DEVOPS_ORGS **Value**
    - **variable** AZURE_DEVOPS_BACKUPS **Value** (Number of backups to keep for each repo)
    - **variable** BITBUCKET_USER **Value**
    - **variable** BITBUCKET_KEY **Value**
    - **variable** BITBUCKET_SECRET **Value**
    - **variable** BITBUCKET_BACKUPS **Value** (Number of backups to keep for each repo)
    - **variable** GITEA_APIURL **Value**
    - **variable** GITEA_TOKEN **Value**
    - **variable** GITEA_ORGS **Value**
    - **variable** GITEA_BACKUPS **Value**
    - **variable** GITHUB_TOKEN **Value**
    - **variable** GITHUB_ORGS **Value** (Optional - comma separated list of organisations)
    - **variable** GITHUB\_SKIP\_USER\_REPOS **Value** (Optional - defaults to false)
    - **variable** GITHUB\_LIMIT\_USER\_OWNED **Value** (Optional - defaults to false)
    - **variable** GITHUB_BACKUPS **Value** (Number of backups to keep for each repo)
    - **variable** GITLAB_TOKEN **Value**
    - **variable** GITLAB_BACKUPS **Value** (Number of backups to keep for each repo)
    - **variable** GITLAB\_PROJECT\_MIN\_ACCESS\_LEVEL **Value** (Optional - scope of repos to backup)
    - **variable** SOURCEHUT_PAT **Value**
    - **variable** SOURCEHUT_APIURL **Value** (Optional)
    - **variable** SOURCEHUT_BACKUPS **Value** (Number of backups to keep for each repo)
    - **variable** SOURCEHUT_BACKUP_LFS **Value** (y/yes to also back up LFS objects)
14. Click 'Apply'
15. Leave settings as default and select 'Next'
16. Check 'Run this container after the wizard is finished' and click 'Apply'

The container should launch in a few seconds. You can view progress by choosing 'Container' in the left-hand menu,
select 'soba', choose 'details' and then click on 'Log'

## restoring backups

A Git bundle is an archive of a Git repository. The simplest way to restore is to clone it like a remote repository.

```bash
git clone soba.20180708153107.bundle soba
```
[release]: https://github.com/jonhadfield/soba/releases
[release-img]: https://img.shields.io/github/release/jonhadfield/soba.svg?logo=github
