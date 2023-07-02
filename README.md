# soba: backup hosted git repositories

[![Build Status](https://travis-ci.org/jonhadfield/soba.svg?branch=master)](https://travis-ci.org/jonhadfield/soba)
[![Codacy Badge](https://app.codacy.com/project/badge/Grade/1bd46b99467c45d99e4903b44a16f874)](https://www.codacy.com/gh/jonhadfield/soba/dashboard?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=jonhadfield/soba&amp;utm_campaign=Badge_Grade)
[![Go Report Card](https://goreportcard.com/badge/github.com/jonhadfield/soba)](https://goreportcard.com/report/github.com/jonhadfield/soba)

- [about](#about)
- [configuration](#configuration)
- [run using command line](#run-using-command-line)
- [scheduling backups](#scheduling-backups)
- [rotating backups](#rotating-backups)
- [logging](#logging)
- [setting provider credentials](#setting-provider-credentials)
- [additional options](#additional-options)
- [run using docker](#run-using-docker)
- [run on Synology NAS](#run-on-synology-nas)
- [restore](#restore)

## about

soba is tool for backing up private and public git repositories hosted on the
most popular hosting providers. It generates a [git bundle](https://git-scm.com/book/en/v2/Git-Tools-Bundling) that stores a backup of each repository as a
single file.

An unchanged git repository will create an identical bundle file so bundles will only be stored if a change has been
made and will not produce duplicates. Since version 1.1.4 you can now [check for changes without cloning](#comparing-remote-repository-with-local-backup).

## latest updates

### 1.2.0 release 2023-07-02

- All GitHub Organizations can now be backed up by specifying * instead of individual names
- GitLab API calls will now be retried if they initially fail

### 1.1.12 released 2023-06-25

- Add support for Gitea

### 1.1.11 released 2023-03-26

- Fix goreleaser to build and distribute docker release

See full changelog [here](./CHANGELOG.md).

## supported OSes

Tested on Windows 10, MacOS, and Linux (amd64).
Not tested, but should also work on builds for: Linux (386, arm386 and arm64), FreeBSD, NetBSD, and OpenBSD.

## supported providers

- BitBucket
- Gitea
- GitHub
- GitLab

## configuration

soba can be run from the command line or as docker container. In both cases the only configuration required is an
environment variable with the directory in which to create backups, and additional to define credentials for each the
providers.

On Windows 10:

- search for 'environment variables' and choose 'Edit environment variables for your account'
- choose 'New...' under the top pane and enter the name/key and value for each of the settings

On Linux and MacOS you would set these using:

```bash
export GIT_BACKUP_DIR="/repo-backups/"
```

To set provider credentials see [below](#setting-provider-credentials).

## run using command line

Download the latest release [here](https://github.com/jonhadfield/soba/releases) and then install:

```bash
install <soba binary> /usr/local/bin/soba
```

After setting `GIT_BACKUP_DIR`, set your provider token(s) as detailed [here](#setting-provider-credentials).

and then run:

```bash
soba
```

## run using docker

Using docker enables you to run soba without anything else installed.

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

## scheduling backups

Backups can be scheduled to run by setting an additional environment variable: GIT_BACKUP_INTERVAL. The value is the
number of hours between backups. For example, this will run the backup daily:

```bash
export GIT_BACKUP_INTERVAL=24
```

if using docker then add:

```bash
-e GIT_BACKUP_INTERVAL=24
```

_Note: the interval is added to the start of the last backup and not the time it finished. Therefore, ensure the interval is greater than the duration of a backup._

## rotating backups

A new bundle is created every time a change is detected in the repository. To keep only the _x_ most recent, use the
following provider specific environment variables:
`GITEA_BACKUPS=x`
`GITHUB_BACKUPS=x`
`GITLAB_BACKUPS=x`
`BITBUCKET_BACKUPS=x`

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
| BitBucket | BITBUCKET_USER                  | [instructions](https://support.atlassian.com/bitbucket-cloud/docs/use-oauth-on-bitbucket-cloud/)       |
|           | BITBUCKET_KEY                   |                                                                                                          |
|           | BITBUCKET_SECRET                |                                                                                                          |                                                                                          |
| Gitea     | GITEA_APIURL                    | e.g. https://gitea.example.com/api/v1|
|           | GITEA_TOKEN                     | [instructions](https://docs.gitea.com/development/api-usage#generating-and-listing-api-tokens)|
| GitHub    | GITHUB_TOKEN                    | [instructions](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token#personal-access-tokens-classic) |
| GitLab    | GITLAB_TOKEN                    | [instructions](https://docs.gitlab.com/ee/user/profile/personal_access_tokens.html)                                       |
|           | GITLAB\_PROJECT\_MIN\_ACCESS\_LEVEL | [instructions](https://docs.gitlab.com/ee/user/permissions.html)                                       |

## additional options

### BitBucket

#### Repo/Bundle comparison method

Environment variable: BITBUCKET_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                                |
|:----------------|:---------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                     |
| refs            | Compare refs without downloading (available since soba 1.1.4)  |

### Gitea

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

#### GitHub Repo/Bundle comparison method

Environment variable: GITHUB_COMPARE

[See explanation below](#comparing-remote-repository-with-local-backup)

| Value           |                                                               |
|:----------------|:--------------------------------------------------------------|
| clone (default) | Clone the remote and compare latest bundle                    |
| refs            | Compare refs without downloading (available since soba 1.1.4) |

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
    - **variable** GITHUB_BACKUPS **Value** (Number of backups to keep for each repo)
    - **variable** GITLAB_TOKEN **Value**
    - **variable** GITLAB_BACKUPS **Value** (Number of backups to keep for each repo)
    - **variable** GITLAB\_PROJECT\_MIN\_ACCESS\_LEVEL **Value** (Optional - scope of repos to backup)
14. Click 'Apply'
15. Leave settings as default and select 'Next'
16. Check 'Run this container after the wizard is finished' and click 'Apply'

The container should launch in a few seconds. You can view progress by choosing 'Container' in the left-hand menu,
select 'soba', choose 'details' and then click on 'Log'

## restore

A Git bundle is an archive of a Git repository. The simplest way to restore is to clone it like a remote repository.

```bash
git clone soba.20180708153107.bundle soba
```
