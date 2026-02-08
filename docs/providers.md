# Provider Configuration

Back to [README](../README.md).

This document covers detailed configuration for each supported provider, advanced options, and platform-specific deployment guides.

## Setting Provider Credentials

On Linux and macOS you can set environment variables manually before each time you run soba:

```bash
export NAME='VALUE'
```

or by defining in a startup file for your shell so they are automatically set and available when you need them. For example, if using the bash shell and running soba as your user, add the relevant export statements to:

```bash
/home/<your-user-id>/.bashrc
```

and run:

```bash
source /home/<your-user-id>/.bashrc
```

On Windows 10:

- search for 'environment variables' and choose 'Edit environment variables for your account'
- choose 'New...' under the top pane and enter the name/key and value for each of the settings

### Credentials from files

You can provide credentials via files using the `_FILE` environment variable pattern. For example:

```bash
GITHUB_TOKEN_FILE=/run/secrets/my_github_token
```

If both the variable and `_FILE` version are set, the variable takes precedence.

---

## Provider Reference

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
|           | BITBUCKET_SECRET                |                                                                                                          |
| Gitea     | GITEA_APIURL                    | [instructions](#gitea) |
|           | GITEA_TOKEN                     | |
| GitHub    | GITHUB_TOKEN                    | [instructions](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#personal-access-tokens-classic) |
| GitLab    | GITLAB_TOKEN                    | [instructions](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)                                       |
|           | GITLAB\_PROJECT\_MIN\_ACCESS\_LEVEL | [instructions](https://docs.gitlab.com/ee/user/permissions.html)                                       |
| Sourcehut | SOURCEHUT_PAT                   | [instructions](https://man.sr.ht/accounts.md#api) |
|           | SOURCEHUT_APIURL                | |
|           | SOURCEHUT_BACKUPS               | |
|           | SOURCEHUT_BACKUP_LFS            | |

---

## Azure DevOps

### Returning organisations' repositories

An organisation must be specified using environment variable `AZURE_DEVOPS_ORGS` in order for soba to discover the projects and their repos.
_Note: Only a single organisation is currently supported._

### Repo/Bundle comparison method

Environment variable: `AZURE_DEVOPS_COMPARE`

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                                |
|:----------------|:---------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                     |
| refs            | Compare refs without downloading (available since soba 1.1.4)  |

---

## BitBucket

### Repo/Bundle comparison method

Environment variable: `BITBUCKET_COMPARE`

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                                |
|:----------------|:---------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                     |
| refs            | Compare refs without downloading (available since soba 1.1.4)  |

### Custom endpoints

To use Bitbucket Server or another custom endpoint, set `BITBUCKET_APIURL` with the API URL.

### API token scopes

When using API tokens as your auth method, only the following scopes are required:
- `read:project:bitbucket`
- `read:repository:bitbucket`

---

## Gitea

### Generating a token

[Official documentation](https://docs.gitea.com/development/api-usage#generating-and-listing-api-tokens)

The value for `GITEA_APIURL` needs to be in the format: `https://[domain]/api/v1`, where domain is something like `gitea.example.com`.

`GITEA_TOKEN` is the secret you need to generate using the API (see official documentation above), or via the web GUI:

1. Login to Gitea
2. Select your user icon in the top right-hand corner and choose `Settings` from the dropdown
3. Select `Applications`
4. Enter a Token Name, e.g. "soba backups"
5. Select `Public only` or `All` depending on use-case
6. Expand the `Select permissions` menu
7. Select `read:organization` and `read:repository`
8. Click on `Generate Token` and the value will appear at the top of the page

### Returning organisations' repositories

Repositories in Gitea organisations are not backed up by default. To back these up, specify a comma separated
list of organisations in the environment variable: `GITEA_ORGS`. To include "all" organisations, set to `*`.

### Repo/Bundle comparison method

Environment variable: `GITEA_COMPARE`

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                               |
|:----------------|:--------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                    |
| refs            | Compare refs without downloading (available since soba 1.1.4) |

---

## GitHub

### Returning organisations' repositories

Repositories in GitHub organisations are not backed up by default. To back these up, specify a comma separated
list of organisations in the environment variable: `GITHUB_ORGS`.

### Skipping user repository backups

By default, all users' repositories will be backed up, even when specifying organisations.
To skip user repositories set environment variable: `GITHUB_SKIP_USER_REPOS` to `true`.

### Limit user repo backups to those owned by the user

By default, all repositories a user is affiliated with, e.g. a collaborator on, are included for backup.
To limit these to only those owned by the user, set environment variable: `GITHUB_LIMIT_USER_OWNED` to `true`.

### Repo/Bundle comparison method

Environment variable: `GITHUB_COMPARE`

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                               |
|:----------------|:--------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                    |
| refs            | Compare refs without downloading (available since soba 1.1.4) |

### Adjust GitHub API behaviour

Environment variables:
- `GITHUB_CALL_SIZE` - number of repositories returned per API call (default 100)
- `GITHUB_WORKER_DELAY` - delay in milliseconds between API workers starting (default 500)

### Custom endpoints

To use GitHub Enterprise or other API endpoints, set `GITHUB_APIURL`.

---

## GitLab

### Filtering projects by access level

By default, every project a user has at least `Reporter` access to will be returned. Set `GITLAB_PROJECT_MIN_ACCESS_LEVEL` to override this with the number matching the desired access level:

| Access Level | Value |
|:-------------|:------|
| Guest        | 10    |
| Reporter     | 20    |
| Developer    | 30    |
| Maintainer   | 40    |
| Owner        | 50    |

See [GitLab documentation](https://docs.gitlab.com/ee/api/members.html#valid-access-levels) for details.

### Repo/Bundle comparison method

Environment variable: `GITLAB_COMPARE`

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                               |
|:----------------|:--------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                    |
| refs            | Compare refs without downloading (available since soba 1.1.4) |

### Custom endpoints

To use a self-hosted GitLab instance, set `GITLAB_APIURL` with the API URL.

---

## Sourcehut

### Repo/Bundle comparison method

Environment variable: `SOURCEHUT_COMPARE`

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                            |
|:----------------|:-------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle |
| refs            | Compare refs without downloading           |

### Custom endpoints

To use a custom Sourcehut instance, set `SOURCEHUT_APIURL` with the API URL.

---

## Comparing Remote Repository with Local Backup

By default, each repository will be cloned, bundled, and that bundle compared with the latest local bundle to check if it should be kept or discarded. When processing many large repositories, this can be a lengthy process.

Alternatively, you can compare the Git refs of the latest local bundle with the remote repository without having to clone. This is carried out using native commands `git bundle list-heads <bundle file>` and `git ls-remote <remote repository>`.

This process is far quicker than cloning but should only be used if the following is understood: Comparing refs means comparing the tips of, and not the entire history of, the repository. [This post on Stack Overflow](https://stackoverflow.com/questions/74281792/git-comparing-local-bundle-with-remote-repository-using-refs-only) goes into additional detail.

---

## Logging

### Persistence

Messages are written to stdout and can be persisted by directing to a file, e.g.
`soba > soba.log`

#### Logging to /var/log/soba

Create a user called soba:
`sudo adduser soba`
Create a log directory:
`sudo mkdir /var/log/soba`
Set user permissions:
`sudo chown soba /var/log/soba && sudo chmod 700 /var/log/soba`
Switch to soba user:
`sudo su - soba`
Run soba and direct output:
`soba > /var/log/soba/soba.log`

### Rotation

[Logrotate](https://linux.die.net/man/8/logrotate) is a utility that comes with most Linux distributions and removes and/or compresses messages older than a certain number of hours or days.

This example assumes you persist the log file to `/var/log/soba/soba.log`.
Create a file in `/etc/logrotate.d/soba` with the following content:

    /var/log/soba/soba.log {
      rotate 7      # remove backups older than seven days
      daily         # process log file each day
      compress      # compress the backup
      copytruncate  # don't delete the file after backup, but instead truncate
    }

Each day, this will copy the latest logs to a new file that is then compressed. The existing log file is then truncated. Any backups older than seven days are then removed.

### Log level

Set `SOBA_LOG` to a number to control verbosity. Higher values increase output.

### Keep running after reboot

In case the computer is rebooted or the process ends for another reason, you can ensure it automatically restarts with a simple script and cron job.

#### Script

For example:

    #!/bin/bash -e
    export GIT_BACKUP_DIR=/backup-dir
    export GITHUB_TOKEN=xxxxxxx   # avoid hard-coding if possible
    export GITHUB_BACKUPS=7
    export GIT_BACKUP_INTERVAL=12
    export GITHUB_COMPARE=refs
    /usr/local/bin/soba

#### Cron job

Ensure the user running soba has an entry in `/etc/cron.allow`.

Run `crontab -e`

Add the following (assuming you have a user called soba with a script to run it called backup in their home directory):
`* * * * * /usr/bin/flock -n /tmp/soba.lockfile /home/soba/backup >> /var/log/soba/soba.log 2>&1`

_A useful tool for testing cron jobs is [crontab guru](https://crontab.guru/)._

---

## Run on Synology NAS

### _The following was tested on DS916+_

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
select 'soba', choose 'details' and then click on 'Log'.

### Setting the request timeout

By default, soba will wait up to ten minutes for a response to complete. This could be anything from an API call to discover repositories to a clone of a large repository.
If you have a slow connection or very large repositories, you may want to increase this. To do so, set the environment variable `GIT_REQUEST_TIMEOUT` to the number of seconds you wish to wait. For example, to wait up to ten minutes:
```bash
export GIT_REQUEST_TIMEOUT=600
```
