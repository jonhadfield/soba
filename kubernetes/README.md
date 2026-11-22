# soba on Kubernetes

back to [README](../README.md).

## run as a cron job on Kubernetes

The recommended way to run soba on Kubernetes is as a cron job.  
A set of example manifest files have been provided in this directory.

### persistent volume claim
[pvc.yaml](pvc.yaml)  
The persistent storage where soba will store the backups.  
_note: an example of a persistent volume is omitted due to the number of valid possibilities. I'd recommend a highly available setup such as RAID1 (or higher), or multi-availability zone (or multi-geo) storage on a cloud provider, exported as an NFS share._

### config map
[configmap.yaml](configmap.yaml)   
The non-secret key/value pairs, e.g. the path for `GIT_BACKUP_DIR`, and optional provider settings such as `GITHUB_ORGS` or `CODEBERG_ORGS`.  
_note: if you prefer to also keep these secret, simply define them as Kubernetes Secrets (below) instead and soba will pick them up from there._

### secrets
[secrets.yaml](secrets.yaml)   
The secret key/value pairs for provider credentials, for example `GITHUB_TOKEN`, `GITLAB_TOKEN`, `GITEA_TOKEN`, or `CODEBERG_TOKEN`. Omit any provider you are not using.

The CronJob loads these as environment variables via `envFrom` / `secretRef`. You can instead mount a Secret (or external secret store) as files and point soba at them with the `_FILE` suffix — for example set `GITHUB_TOKEN_FILE=/run/secrets/github_token`. If both the variable and its `_FILE` counterpart are set, the variable takes precedence. See [credentials from files](../docs/providers.md#credentials-from-files).

### cron job
[cron.yaml](cron.yaml)   
The schedule for soba to run and where its configuration is found.  
The schedule is defined using the same cron syntax as can be used with the binary and docker image distributions.
