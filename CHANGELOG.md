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