package githosts

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type repository struct {
	Name          string
	NameWithOwner string
	Domain        string
	HTTPSUrl      string
	SSHUrl        string
	URLWithToken  string
}

type describeReposOutput struct {
	Repos []repository
}

type gitProvider interface {
	getAPIURL() string
	describeRepos() describeReposOutput
	Backup(string)
}

type newHostInput struct {
	Domain       string
	ProviderName string
	APIURL       string
}

func createHost(input newHostInput) (gitProvider, error) {
	var hostErr error
	switch strings.ToLower(input.ProviderName) {
	case "github":
		return githubHost{
			Provider: "Github",
			APIURL:   input.APIURL,
		}, nil
	case "gitlab":
		return gitlabHost{
			Provider: "Gitlab",
			APIURL:   input.APIURL,
		}, nil
	default:
		hostErr = errors.New("provider invalid or not implemented")
	}

	return nil, hostErr
}

func processBackup(repo repository, backupDIR string) {
	// CREATE BACKUP PATH
	workingPath := backupDIR + string(os.PathSeparator) + workingDIRName + string(os.PathSeparator) + repo.Domain + string(os.PathSeparator) + repo.NameWithOwner
	backupPath := backupDIR + string(os.PathSeparator) + repo.Domain + string(os.PathSeparator) + repo.NameWithOwner
	// DELETE EXISTING CLONE
	delErr := deleteDirIfExists(workingPath)
	if delErr != nil {
		logger.Fatal(delErr)
	}
	// CLONE REPO
	logger.Printf("cloning repo '%s'", repo.HTTPSUrl)
	cloneCmd := exec.Command("git", "clone", "--mirror", repo.URLWithToken, workingPath)
	cloneCmd.Dir = backupDIR
	var cloneOut bytes.Buffer
	cloneCmd.Stdout = &cloneOut
	cloneErr := cloneCmd.Run()
	if cloneErr != nil {
		logger.Fatal(cloneErr)
	}
	// CREATE BUNDLE
	objectsPath := workingPath + string(os.PathSeparator) + "objects"
	dirs, _ := ioutil.ReadDir(objectsPath)
	emptyPack, _ := IsEmpty(objectsPath + string(os.PathSeparator) + "pack")
	if len(dirs) == 2 && emptyPack {
		logger.Printf("repo %s is empty, so not creating bundle", repo.Name)
	} else {
		logger.Printf("creating bundle for '%s'", repo.Name)
		backupFile := repo.Name + "." + getTimestamp() + ".bundle"
		backupFilePath := backupPath + string(os.PathSeparator) + backupFile
		createErr := createDirIfAbsent(backupPath)
		if createErr != nil {
			logger.Fatal(createErr)
		}
		bundleCmd := exec.Command("git", "bundle", "create", backupFilePath, "--all")
		bundleCmd.Dir = workingPath
		var bundleOut bytes.Buffer
		bundleCmd.Stdout = &bundleOut
		bundleCmd.Stderr = &bundleOut
		bundleErr := bundleCmd.Run()
		if bundleErr != nil {
			logger.Fatal(bundleErr)
		}
	}
}
