package repo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/afero"
)

const (
	envIpfsPath = "IPFS_PATH"
	defIpfsDir  = ".ipfs"
	versionFile = "version"
)

// GetRepoVersion returns the version of the repo in the ipfs directory.  If the
// ipfs directory is not specified then the default location is used.
func GetRepoVersion(fs afero.Fs, ipfsDir string) (int, error) {
	ipfsDir, err := CheckIpfsDir(fs, ipfsDir)
	if err != nil {
		return 0, err
	}
	return repoVersion(fs, ipfsDir)
}

func repoVersion(fs afero.Fs, ipfsDir string) (int, error) {
	c, err := afero.ReadFile(fs, filepath.Join(ipfsDir, versionFile))
	if err != nil {
		return 0, err
	}

	ver, err := strconv.Atoi(strings.TrimSpace(string(c)))
	if err != nil {
		return 0, errors.New("invalid data in repo version file")
	}
	return ver, nil
}

// CheckIpfsDir gets the ipfs directory and checks that the directory exists.
func CheckIpfsDir(fs afero.Fs, dir string) (string, error) {
	var err error
	dir, err = IpfsDir(dir)
	if err != nil {
		return "", err
	}

	_, err = fs.Stat(dir)
	if err != nil {
		return "", err
	}

	return dir, nil
}

// IpfsDir returns the path of the ipfs directory.  If dir specified, then
// returns the expanded version dir.  If dir is "", then return the directory
// set by IPFS_PATH, or if IPFS_PATH is not set, then return the default
// location in the home directory.
func IpfsDir(dir string) (string, error) {
	var err error
	if dir == "" {
		dir = os.Getenv(envIpfsPath)
	}
	if dir != "" {
		dir, err = homedir.Expand(dir)
		if err != nil {
			return "", err
		}
		return dir, nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("could not determine IPFS_PATH, home dir not set")
	}

	return filepath.Join(home, defIpfsDir), nil
}

// WriteRepoVersion writes the specified repo version to the repo located in
// ipfsDir. If ipfsDir is not specified, then the default location is used.
func WriteRepoVersion(fs afero.Fs, ipfsDir string, version int) error {
	ipfsDir, err := IpfsDir(ipfsDir)
	if err != nil {
		return err
	}

	vFilePath := filepath.Join(ipfsDir, versionFile)
	return afero.WriteFile(fs, vFilePath, []byte(fmt.Sprintf("%d\n", version)), 0644)
}
