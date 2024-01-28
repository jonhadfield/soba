# soba on Kubernetes

back to [README](../README.md).

## run as a cron job on Kubernetes

The recommended way to run soba on Kubernetes is as a cron job.  
A set of example manifest files have been provided in the this directory.  

### persistent volume claim
[pvc.yaml](pvc.yaml)  
The persistent storage where soba will store the backups.  
_note: an example of a persistent volume is omitted due to the number of valid possibilities. I'd recommend a highly available setup such as RAID1 (or higher), or multi-availability zone (or multi-geo) storage on a cloud provider, exported as an NFS share._

### config map
[configmap.yaml](configmap.yaml)   
The non-secret key/value pairs, e.g. the path for:`GIT_BACKUP_DIR`.  
_note: if you prefer to also keep these secret, simply define them as Kubernetes Secrets (below) instead and soba will pick them up from there._

### secrets
[secrets.yaml](secrets.yaml)   
The secret key/value pairs, e.g. your GitHub token defined in:`GITHUB_TOKEN`.  

### cron job
[cron.yaml](cron.yaml)   
The schedule for soba to run and where its configuration is found.  
The schedule is defined using the same cron syntax as can be used with the binary and docker image distributions.



