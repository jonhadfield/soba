package githosts

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type repository struct {
	Name             string
	Owner            string
	NameWithOwner    string
	Domain           string
	HTTPSUrl         string
	SSHUrl           string
	URLWithToken     string
	URLWithBasicAuth string
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
	case "bitbucket":
		return bitbucketHost{
			Provider: "BitBucket",
			APIURL:   input.APIURL,
		}, nil
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

func processBackup(repo repository, backupDIR string) error {
	// CREATE BACKUP PATH
	workingPath := backupDIR + pathSep + workingDIRName + pathSep + repo.Domain + pathSep + repo.NameWithOwner
	backupPath := backupDIR + pathSep + repo.Domain + pathSep + repo.NameWithOwner
	// DELETE EXISTING CLONE
	delErr := os.RemoveAll(workingPath + pathSep)
	if delErr != nil {
		logger.Fatal(delErr)
	}
	// CLONE REPO
	logger.Printf("cloning: %s", repo.HTTPSUrl)
	var cloneURL string
	if repo.URLWithToken != "" {
		cloneURL = repo.URLWithToken
	} else if repo.URLWithBasicAuth != "" {
		cloneURL = repo.URLWithBasicAuth
	}
	cloneCmd := exec.Command("git", "clone", "-v", "--mirror", cloneURL, workingPath)
	cloneCmd.Dir = backupDIR
	var cloneStdErr bytes.Buffer
	cloneCmd.Stderr = &cloneStdErr
	cloneErr := cloneCmd.Run()
	stderr := cloneStdErr.String()
	var errOutString string

	if cloneErr != nil {
		errOutString = cloneErr.Error() + "\n" + stderr
		cloneErr = errors.WithStack(fmt.Errorf(errOutString))
		logger.Fatal(cloneErr)
	}
	// CREATE BUNDLE
	objectsPath := workingPath + pathSep + "objects"
	dirs, _ := ioutil.ReadDir(objectsPath)
	emptyPack, checkEmptyErr := isEmpty(objectsPath + pathSep + "pack")
	if checkEmptyErr != nil {
		logger.Printf("failed to check if: '%s' is empty", objectsPath+pathSep+"pack")
	}
	if len(dirs) == 2 && emptyPack {
		logger.Printf("%s is empty, so not creating bundle", repo.Name)
	} else {

		backupFile := repo.Name + "." + getTimestamp() + bundleExtension
		backupFilePath := backupPath + pathSep + backupFile
		createErr := createDirIfAbsent(backupPath)
		if createErr != nil {
			logger.Fatal(createErr)
		}
		logger.Printf("creating bundle for: %s", repo.Name)
		bundleCmd := exec.Command("git", "bundle", "create", backupFilePath, "--all")
		bundleCmd.Dir = workingPath
		var bundleOut bytes.Buffer
		bundleCmd.Stdout = &bundleOut
		bundleCmd.Stderr = &bundleOut
		bundleErr := bundleCmd.Run()
		if bundleErr != nil {
			logger.Fatal(bundleErr)
		}
		removeBundleIfDuplicate(backupPath)

	}
	return nil
}

func removeBundleIfDuplicate(dir string) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		logger.Println(err)
		return
	}
	if len(files) == 1 {
		return
	}
	// get timestamps in filenames for sorting
	fNameTimes := map[string]int{}
	for _, f := range files {
		if strings.Count(f.Name(), ".") >= 2 {
			parts := strings.Split(f.Name(), ".")
			strTimestamp := parts[len(parts)-2]
			intTimestamp, convErr := strconv.Atoi(strTimestamp)
			if convErr == nil {
				fNameTimes[f.Name()] = intTimestamp
			}
		}

	}
	type kv struct {
		Key   string
		Value int
	}
	var ss []kv
	for k, v := range fNameTimes {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	// check if file sizes are same
	latestBundleSize := getFileSize(dir + pathSep + ss[0].Key)
	previousBundleSize := getFileSize(dir + pathSep + ss[1].Key)
	if latestBundleSize == previousBundleSize {
		// check if hashes match
		latestBundleHash, latestHashErr := getMD5Hash(dir + pathSep + ss[0].Key)
		if latestHashErr != nil {
			logger.Printf("failed to get md5 hash for: %s", dir+pathSep+ss[0].Key)
		}
		previousBundleHash, previousHashErr := getMD5Hash(dir + pathSep + ss[1].Key)
		if previousHashErr != nil {
			logger.Printf("failed to get md5 hash for: %s", dir+pathSep+ss[1].Key)
		}
		if reflect.DeepEqual(latestBundleHash, previousBundleHash) {
			logger.Printf("no change since previous bundle: %s", ss[1].Key)
			logger.Printf("deleting duplicate bundle: %s", ss[0].Key)
			if deleteFile(dir+pathSep+ss[0].Key) != nil {
				logger.Println("failed to remove duplicate bundle")
			}
		}
	}
}

func deleteFile(path string) (err error) {
	err = os.Remove(path)
	return
}

func getMD5Hash(filePath string) ([]byte, error) {
	var result []byte
	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer func() {
		if cErr := file.Close(); cErr != nil {
			logger.Printf("warn: failed to close: %s", filePath)
		}
	}()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return result, err
	}

	return hash.Sum(result), nil
}

func getFileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		logger.Println(err)
		return 0
	}
	return fi.Size()
}
