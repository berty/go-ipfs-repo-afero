package main

import (
	"fmt"
	"os"

	filestore "github.com/ipfs/go-filestore"
	keystore "github.com/ipfs/go-ipfs-keystore"
	repo "github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/spf13/afero"

	config "github.com/ipfs/go-ipfs-config"
	ma "github.com/multiformats/go-multiaddr"
)

// FIXME: port to berty repo template

type AferoRepo struct {
	fs     afero.Fs
	path   string
	config *config.Config
}

var _ repo.Repo = (*AferoRepo)(nil)

// version number that we are currently expecting to see
const RepoVersion = 11

func Open(fs afero.Fs, repoPath string) (repo.Repo, error) {
	/*
		packageLock.Lock()
		defer packageLock.Unlock()
	*/

	r, err := newAferoRepo(fs, repoPath)
	if err != nil {
		return nil, err
	}

	if err := checkInitialized(r.fs, r.path); err != nil {
		return nil, err
	}

	/*
		r.lockfile, err = lockfile.Lock(r.path, LockFile)
		if err != nil {
			return nil, err
		}
		keepLocked := false
		defer func() {
			// unlock on error, leave it locked on success
			if !keepLocked {
				r.lockfile.Close()
			}
		}()
	*/

	ver, err := GetRepoVersion(r.fs, r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fsrepo.ErrNoVersion
		}
		return nil, err
	}

	if RepoVersion > ver {
		return nil, fsrepo.ErrNeedMigration
	} else if ver > RepoVersion {
		// program version too low for existing repo
		return nil, fmt.Errorf(programTooLowMessage, RepoVersion, ver)
	}

	// check repo path, then check all constituent parts.
	if err := Writable(r.fs, r.path); err != nil {
		return nil, err
	}

	if err := r.openConfig(); err != nil {
		return nil, err
	}

	/*
		if err := r.openDatastore(); err != nil {
			return nil, err
		}

		if err := r.openKeystore(); err != nil {
			return nil, err
		}

		if r.config.Experimental.FilestoreEnabled || r.config.Experimental.UrlstoreEnabled {
			r.filemgr = filestore.NewFileManager(r.ds, filepath.Dir(r.path))
			r.filemgr.AllowFiles = r.config.Experimental.FilestoreEnabled
			r.filemgr.AllowUrls = r.config.Experimental.UrlstoreEnabled
		}

		keepLocked = true
	*/
	return r, nil
}

// Config returns the ipfs configuration file from the repo. Changes made
// to the returned config are not automatically persisted.
func (r *AferoRepo) Config() (*config.Config, error) {
	panic("not implemented")
}

// BackupConfig creates a backup of the current configuration file using
// the given prefix for naming.
func (r *AferoRepo) BackupConfig(prefix string) (string, error) {
	panic("not implemented")
}

// SetConfig persists the given configuration struct to storage.
func (r *AferoRepo) SetConfig(*config.Config) error {
	panic("not implemented")
}

// SetConfigKey sets the given key-value pair within the config and persists it to storage.
func (r *AferoRepo) SetConfigKey(key string, value interface{}) error {
	panic("not implemented")
}

// GetConfigKey reads the value for the given key from the configuration in storage.
func (r *AferoRepo) GetConfigKey(key string) (interface{}, error) {
	panic("not implemented")
}

// Datastore returns a reference to the configured data storage backend.
func (r *AferoRepo) Datastore() repo.Datastore {
	panic("not implemented")
}

// GetStorageUsage returns the number of bytes stored.
func (r *AferoRepo) GetStorageUsage() (uint64, error) {
	panic("not implemented")
}

// Keystore returns a reference to the key management interface.
func (r *AferoRepo) Keystore() keystore.Keystore {
	panic("not implemented")
}

// FileManager returns a reference to the filestore file manager.
func (r *AferoRepo) FileManager() *filestore.FileManager {
	panic("not implemented")
}

// SetAPIAddr sets the API address in the repo.
func (r *AferoRepo) SetAPIAddr(addr ma.Multiaddr) error {
	panic("not implemented")
}

// SwarmKey returns the configured shared symmetric key for the private networks feature.
func (r *AferoRepo) SwarmKey() ([]byte, error) {
	panic("not implemented")
}

// Close see io.Closer
func (r *AferoRepo) Close() error {
	panic("not implemented")
}
