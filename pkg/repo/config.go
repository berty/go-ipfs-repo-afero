package repo

import (
	"strings"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/spf13/afero"
)

// DefaultDatastoreConfig is an internal function exported to aid in testing.
func DefaultDatastoreConfig() config.Datastore {
	return config.Datastore{
		StorageMax:         "10GB",
		StorageGCWatermark: 90, // 90%
		GCPeriod:           "1h",
		BloomFilterSize:    0,
		Spec:               flatfsSpec(),
	}
}

func flatfsSpec() map[string]interface{} {
	return map[string]interface{}{
		"type": "mount",
		"mounts": []interface{}{
			map[string]interface{}{
				"mountpoint": "/blocks",
				"type":       "measure",
				"prefix":     "afero.blocks",
				"child": map[string]interface{}{
					"type": "afero",
					"path": "blocks",
					/*"sync":      true,
					"shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",*/
				},
			},
			map[string]interface{}{
				"mountpoint": "/",
				"type":       "measure",
				"prefix":     "afero.datastore",
				"child": map[string]interface{}{
					"type": "afero",
					"path": "datastore",
					//"compression": "none",
				},
			},
		},
	}
}

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
