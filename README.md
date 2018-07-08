
# soba: backup hosted git repositories

- [about](#about)
- [configuration](#configuration)
- [running using command line](#run-using-command-line)
- [running using docker](#run-using-docker)

## about

soba is tool for backing up private and public git repositories hosted on the most popular hosting providers. It generates a [git bundle](https://git-scm.com/book/en/v2/Git-Tools-Bundling) that stores a backup of each repository as a single file. 

An unchanged git repository will create an identical bundle file so bundles will only be stored if a change has been made and will not produce duplicates.


## configuration

soba can be run from the command line or as docker container. In both cases the only configuration required is an environment variable with the directory in which to create backups and one with a token for each of the providers. For example, the following would create a directory for each provider in existing directory /repo-backups/:
``` bash
GIT_BACKUP_DIR="/repo-backups/"
```

On Linux and MacOS you would set this using:

```bash
$ export GIT_BACKUP_DIR="/repo-backups/"
```

To set provider tokens see [below](#setting-provider-tokens).

### run using command line

Download the latest release [here](https://github.com/jonhadfield/soba/releases) and then install:
```
$ install <soba binary> /usr/local/bin/soba
```

After setting GIT_BACKUP_DIR, set your provider token(s) as detailed [here](#setting-provider-tokens).

and then run:

```bash
$ soba
```

### run using docker

Using docker enables you to run soba without anything else installed.

Docker requires you pass environment variables to the container using the '-e' option and that you mount your preferred backup directory. For example:

```bash
$ docker run --rm -t quay.io/jonhadfield/soba \
             -v <your backup dir>:/backup \
             -e GIT_BACKUP_DIR='MYBACKUPDIR' \
             -e GITHUB_TOKEN='MYGITHUBTOKEN' \
             -e GITLAB_TOKEN='MYGITLABTOKEN'
```

To hide credentials, you can instead use exported environment variables and specify using this syntax:

```bash
$ docker run --rm -t quay.io/jonhadfield/soba \
            -v <your backup dir>:/backup \
             -e GIT_BACKUP_DIR='MYBACKUPDIR' \
             -e GITHUB_TOKEN=$GITHUB_TOKEN \
             -e GITLAB_TOKEN=$GITLAB_TOKEN
```


## scheduling backups

Backups can be scheduled to run by setting an additional environment variable: GIT_BACKUP_INTERVAL. The value is the number of hours between backups. For example, this will run the backup daily:

```bash
$ export GIT_BACKUP_INTERVAL=24
```

if using docker then add:

```bash
-e GIT_BACKUP_INTERVAL=24
```

Note: the interval is added to the start of the last backup and not the time it finished. Therefore, ensure the interval is greater than the duration of a backup.

## setting provider tokens

On Linux and MacOS you can set environment variables manually before each time you run soba:

```bash
$ export NAME='VALUE'
```
    
or by defining in a startup file for your shell so they are automatically set and available when you need them. For example, if using the bash shell and running soba as your user, add the relevant export statements to the following file: 

```
/home/<your-user-id>/.bashrc
```

and run:

```bash
$ source /home/<your-user-id>/.bashrc
```

| Provider | Environment Variable | Generating token |
|:---------|:---------------------|:-----------------|
| GitHub   | GITHUB_TOKEN         | [instructions](https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/)
| GitLab   | GITLAB_TOKEN         | [instructions](https://gitlab.com/profile/personal_access_tokens)

_additional providers coming soon_  

