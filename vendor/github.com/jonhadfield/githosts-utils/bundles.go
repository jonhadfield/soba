package githosts

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/tozd/go/errors"
)

const (
	bundleExtension = ".bundle"
	// invalidBundleStringCheck checks for a portion of the following in the command output
	// to determine if valid: "does not look like a v2 or v3 bundle file".
	invalidBundleStringCheck = "does not look like"
	bundleTimestampChars     = 14
	minBundleFileNameTokens  = 3
)

func getLatestBundlePath(backupPath string) (string, error) {
	bFiles, err := getBundleFiles(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to get bundle files: %w", err)
	}

	if len(bFiles) == 0 {
		return "", errors.New("no bundle files found in path")
	}

	// get timestamps in filenames for sorting
	fNameTimes := map[string]int{}

	for _, f := range bFiles {
		var ts int
		if ts, err = getTimeStampPartFromFileName(f.info.Name()); err == nil {
			fNameTimes[f.info.Name()] = ts

			continue
		}
		// ignoring error output
	}

	type kv struct {
		Key   string
		Value int
	}

	ss := make([]kv, 0, len(fNameTimes))

	for k, v := range fNameTimes {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	return filepath.Join(backupPath, ss[0].Key), nil
}

func getBundleRefs(bundlePath string) (gitRefs, error) {
	bundleRefsCmd := exec.Command("git", "bundle", "list-heads", bundlePath)

	out, bundleRefsCmdErr := bundleRefsCmd.CombinedOutput()
	if bundleRefsCmdErr != nil {
		return nil, errors.New(string(out))
	}

	refs, err := generateMapFromRefsCmdOutput(out)
	if err != nil {
		return nil, fmt.Errorf("failed to generate map from refs cmd output: %w", err)
	}

	return refs, nil
}

func dirHasBundles(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		return false
	}

	defer func() {
		if err = f.Close(); err != nil {
			logger.Print(err.Error())
		}
	}()

	// TODO: why limit to 1?
	names, err := f.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return false
	}

	if err != nil {
		logger.Printf("failed to read bundle directory contents: %s", err.Error())
	}

	for _, name := range names {
		if strings.HasSuffix(name, bundleExtension) {
			return true
		}
	}

	return false
}

func getLatestBundleRefs(backupPath string) (gitRefs, error) {
	// if we encounter an invalid bundle, then we need to repeat until we find a valid one or run out
	for {
		path, err := getLatestBundlePath(backupPath)
		if err != nil {
			return nil, err
		}

		// get refs for bundle
		var refs gitRefs

		if refs, err = getBundleRefs(path); err != nil {
			// failed to get refs
			if strings.Contains(err.Error(), invalidBundleStringCheck) {
				// rename the invalid bundle
				logger.Printf("renaming invalid bundle to %s.invalid",
					path)

				if err = os.Rename(path,
					path+".invalid"); err != nil {
					// failed to rename, meaning a filesystem or permissions issue
					return nil, fmt.Errorf("failed to rename invalid bundle %w", err)
				}

				// invalid bundle rename, so continue to check for the next latest bundle
				continue
			}
		}

		// otherwise return the refs
		return refs, nil
	}
}

func createBundle(logLevel int, workingPath, backupPath string, repo repository) errors.E {
	objectsPath := filepath.Join(workingPath, "objects")

	dirs, readErr := os.ReadDir(objectsPath)
	if readErr != nil {
		return errors.Errorf("failed to read objectsPath: %s: %s", objectsPath, readErr)
	}

	emptyClone, err := isEmpty(workingPath)
	if err != nil {
		return errors.Errorf("failed to check if clone is empty: %s", err)
	}

	if len(dirs) == 2 && emptyClone {
		return errors.Errorf("%s is empty", repo.PathWithNameSpace)
	}

	backupFile := repo.Name + "." + getTimestamp() + bundleExtension
	backupFilePath := filepath.Join(backupPath, backupFile)

	createErr := createDirIfAbsent(backupPath)
	if createErr != nil {
		return errors.Errorf("failed to create backup path: %s: %s", backupPath, createErr)
	}

	logger.Printf("creating bundle for: %s", repo.Name)

	bundleCmd := exec.Command("git", "bundle", "create", backupFilePath, "--all")
	bundleCmd.Dir = workingPath

	var bundleOut bytes.Buffer

	bundleCmd.Stdout = &bundleOut
	bundleCmd.Stderr = &bundleOut

	startBundle := time.Now()

	if bundleErr := bundleCmd.Run(); bundleErr != nil {
		return errors.Errorf("failed to create bundle: %s: %s", repo.Name, bundleErr)
	}

	if logLevel > 0 {
		logger.Printf("git bundle create time for %s %s: %s", repo.Domain, repo.Name, time.Since(startBundle).String())
	}

	return nil
}

func getBundleFiles(backupPath string) (bundleFiles, error) {
	files, err := os.ReadDir(backupPath)
	if err != nil {
		return nil, errors.Wrap(err, "backup path read failed")
	}

	var bfs bundleFiles

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), bundleExtension) {
			continue
		}

		var ts time.Time

		ts, err = timeStampFromBundleName(f.Name())
		if err != nil {
			return nil, err
		}

		var info os.FileInfo

		info, err = f.Info()
		if err != nil {
			return nil, err
		}

		bfs = append(bfs, bundleFile{
			info:    info,
			created: ts,
		})
	}

	sort.Sort(bfs)

	return bfs, err
}

func pruneBackups(backupPath string, keep int) errors.E {
	files, readErr := os.ReadDir(backupPath)
	if readErr != nil {
		return errors.Wrap(readErr, "backup path read failed")
	}

	if len(files) > 0 {
		logger.Printf("pruning %s to keep %d newest only", backupPath, keep)
	}

	var bfs bundleFiles

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), bundleExtension) {
			logger.Printf("skipping non bundle file '%s'", f.Name())

			continue
		}

		var ts time.Time

		ts, err := timeStampFromBundleName(f.Name())
		if err != nil {
			return err
		}

		var info os.FileInfo

		info, infoErr := f.Info()
		if infoErr != nil {
			return errors.Wrap(infoErr, "failed to get file info")
		}

		bfs = append(bfs, bundleFile{
			info:    info,
			created: ts,
		})
	}

	sort.Sort(bfs)

	firstFilesToDelete := len(bfs) - keep

	var err errors.E

	for x, f := range files {
		if x < firstFilesToDelete {
			if removeErr := os.Remove(filepath.Join(backupPath, f.Name())); err != nil {
				return errors.Wrap(removeErr, "failed to remove file")
			}

			continue
		}

		break
	}

	return nil
}

type bundleFile struct {
	info    os.FileInfo
	created time.Time
}

type bundleFiles []bundleFile

func (b bundleFiles) Len() int {
	return len(b)
}

func (b bundleFiles) Less(i, j int) bool {
	return b[i].created.Before(b[j].created)
}

func (b bundleFiles) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func timeStampFromBundleName(i string) (time.Time, errors.E) {
	tokens := strings.Split(i, ".")
	if len(tokens) < minBundleFileNameTokens {
		return time.Time{}, errors.New("invalid bundle name")
	}

	sTime := tokens[len(tokens)-2]
	if len(sTime) != bundleTimestampChars {
		return time.Time{}, errors.Errorf("bundle '%s' has an invalid timestamp", i)
	}

	return timeStampToTime(sTime)
}

func getTimeStampPartFromFileName(name string) (int, error) {
	if strings.Count(name, ".") >= minBundleFileNameTokens-1 {
		parts := strings.Split(name, ".")

		strTimestamp := parts[len(parts)-2]

		return strconv.Atoi(strTimestamp)
	}

	return 0, fmt.Errorf("filename '%s' does not match bundle format <repo-name>.<timestamp>.bundle",
		name)
}

func filesIdentical(path1, path2 string) bool {
	// check if file sizes are same
	latestBundleSize := getFileSize(path1)

	previousBundleSize := getFileSize(path2)

	if latestBundleSize == previousBundleSize {
		// check if hashes match
		latestBundleHash, latestHashErr := getSHA2Hash(path1)
		if latestHashErr != nil {
			logger.Printf("failed to get sha2 hash for: %s", path1)
		}

		previousBundleHash, previousHashErr := getSHA2Hash(path2)

		if previousHashErr != nil {
			logger.Printf("failed to get sha2 hash for: %s", path2)
		}

		if reflect.DeepEqual(latestBundleHash, previousBundleHash) {
			return true
		}
	}

	return false
}

func removeBundleIfDuplicate(dir string) {
	files, err := getBundleFiles(dir)
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
		var ts int
		if ts, err = getTimeStampPartFromFileName(f.info.Name()); err == nil {
			fNameTimes[f.info.Name()] = ts
		}
	}

	type kv struct {
		Key   string
		Value int
	}

	ss := make([]kv, 0, len(fNameTimes))

	for k, v := range fNameTimes {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	latestBundleFilePath := filepath.Join(dir, ss[0].Key)
	previousBundleFilePath := filepath.Join(dir, ss[1].Key)

	if filesIdentical(latestBundleFilePath, previousBundleFilePath) {
		logger.Printf("no change since previous bundle: %s", ss[1].Key)
		logger.Printf("deleting duplicate bundle: %s", ss[0].Key)

		if deleteFile(filepath.Join(dir, ss[0].Key)) != nil {
			logger.Println("failed to remove duplicate bundle")
		}
	}
}

func deleteFile(path string) error {
	if err := os.Remove(path); err != nil {
		return errors.Wrap(err, "failed to remove file")
	}

	return nil
}

func getSHA2Hash(filePath string) ([]byte, error) {
	var result []byte

	file, err := os.Open(filePath)
	if err != nil {
		return result, errors.Wrap(err, "failed to open file")
	}

	defer func() {
		if err = file.Close(); err != nil {
			logger.Printf("warn: failed to close: %s", filePath)
		}
	}()

	hash := sha256.New()
	if _, err = io.Copy(hash, file); err != nil {
		return result, errors.Wrap(err, "failed to get hash")
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
