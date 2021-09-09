package repo

import (
	"strings"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/spf13/afero"
)

func checkInitialized(fs afero.Fs, path string) error {
	if !isInitializedUnsynced(fs, path) {
		alt := strings.Replace(path, ".ipfs", ".go-ipfs", 1)
		if isInitializedUnsynced(fs, alt) {
			return fsrepo.ErrOldRepo
		}
		return fsrepo.NoRepoError{Path: path}
	}
	return nil
}

// isInitializedUnsynced reports whether the repo is initialized. Caller must
// hold the packageLock.
func isInitializedUnsynced(fs afero.Fs, repoPath string) bool {
	return configIsInitialized(fs, repoPath)
}

func configIsInitialized(fs afero.Fs, path string) bool {
	configFilename, err := config.Filename(path)
	if err != nil {
		return false
	}
	exists, err := afero.Exists(fs, configFilename)
	if err != nil {
		return false
	}
	return exists
}
