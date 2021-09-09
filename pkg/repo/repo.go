package repo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	filestore "github.com/ipfs/go-filestore"
	keystore "github.com/ipfs/go-ipfs-keystore"
	repo "github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/pkg/errors"
	"github.com/spf13/afero"

	config "github.com/ipfs/go-ipfs-config"
	ma "github.com/multiformats/go-multiaddr"

	lockfile "github.com/n0izn0iz/go-ipfs-repo-afero/pkg/lock"
)

// FIXME: port to berty repo template

type AferoRepo struct {
	fs       afero.Fs
	path     string
	config   *config.Config
	ds       repo.Datastore
	keystore keystore.Keystore
	closed   bool
	lockfile io.Closer
}

var _ repo.Repo = (*AferoRepo)(nil)

// version number that we are currently expecting to see
const RepoVersion = 11
const apiFile = "api"
const repoLock = "repo.lock"

var (

	// packageLock must be held to while performing any operation that modifies an
	// FSRepo's state field. This includes Init, Open, Close, and Remove.
	packageLock sync.Mutex

	// onlyOne keeps track of open FSRepo instances.
	//
	// TODO: once command Context / Repo integration is cleaned up,
	// this can be removed. Right now, this makes ConfigCmd.Run
	// function try to open the repo twice:
	//
	//     $ ipfs daemon &
	//     $ ipfs config foo
	//
	// The reason for the above is that in standalone mode without the
	// daemon, `ipfs config` tries to save work by not building the
	// full IpfsNode, but accessing the Repo directly.
	//onlyOne repo.OnlyOne
)

func Open(fs afero.Fs, repoPath string) (repo.Repo, error) {
	packageLock.Lock()
	defer packageLock.Unlock()

	r, err := newAferoRepo(fs, repoPath)
	if err != nil {
		return nil, errors.Wrap(err, "instanciate afero repo")
	}

	if err := checkInitialized(r.fs, r.path); err != nil {
		return nil, errors.Wrap(err, "check repo init")
	}

	r.lockfile, err = lockfile.Lock(r.fs, r.path, repoLock)
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

	ver, err := GetRepoVersion(r.fs, r.path)
	if err != nil {
		return nil, errors.Wrap(err, "get repo version")
	}

	if RepoVersion > ver {
		return nil, fsrepo.ErrNeedMigration
	} else if ver > RepoVersion {
		// program version too low for existing repo
		return nil, fmt.Errorf(programTooLowMessage, RepoVersion, ver)
	}

	// check repo path, then check all constituent parts.
	if err := Writable(r.fs, r.path); err != nil {
		return nil, errors.Wrap(err, "check if repo is writable")
	}

	if err := r.openConfig(); err != nil {
		return nil, errors.Wrap(err, "open repo config")
	}

	if err := r.openDatastore(); err != nil {
		return nil, errors.Wrap(err, "open datastore config")
	}

	if err := r.openKeystore(); err != nil {
		return nil, err
	}

	/*
		if r.config.Experimental.FilestoreEnabled || r.config.Experimental.UrlstoreEnabled {
			r.filemgr = filestore.NewFileManager(r.ds, filepath.Dir(r.path))
			r.filemgr.AllowFiles = r.config.Experimental.FilestoreEnabled
			r.filemgr.AllowUrls = r.config.Experimental.UrlstoreEnabled
		}
	*/

	keepLocked = true
	return r, nil
}

// Config returns the ipfs configuration file from the repo. Changes made
// to the returned config are not automatically persisted.
func (r *AferoRepo) Config() (*config.Config, error) {
	packageLock.Lock()
	defer packageLock.Unlock()
	if r.closed {
		return nil, errors.New("cannot access config, repo not open")
	}

	return r.config, nil
}

// BackupConfig creates a backup of the current configuration file using
// the given prefix for naming.
func (r *AferoRepo) BackupConfig(prefix string) (string, error) {
	return "", errors.New("AferoRepo.BackupConfig not implemented")
}

// SetConfig persists the given configuration struct to storage.
func (r *AferoRepo) SetConfig(*config.Config) error {
	return errors.New("AferoRepo.SetConfig not implemented")
}

// SetConfigKey sets the given key-value pair within the config and persists it to storage.
func (r *AferoRepo) SetConfigKey(key string, value interface{}) error {
	return errors.New("AferoRepo.SetConfigKey not implemented")
}

// GetConfigKey reads the value for the given key from the configuration in storage.
func (r *AferoRepo) GetConfigKey(key string) (interface{}, error) {
	return nil, errors.New("AferoRepo.GetConfigKey not implemented")
}

// Datastore returns a reference to the configured data storage backend.
func (r *AferoRepo) Datastore() repo.Datastore {
	packageLock.Lock()
	d := r.ds
	packageLock.Unlock()
	return d
}

// GetStorageUsage returns the number of bytes stored.
func (r *AferoRepo) GetStorageUsage() (uint64, error) {
	return 0, errors.New("AferoRepo.GetStorageUsage not implemented")
}

// Keystore returns a reference to the key management interface.
func (r *AferoRepo) Keystore() keystore.Keystore {
	return r.keystore
}

// FileManager returns a reference to the filestore file manager.
func (r *AferoRepo) FileManager() *filestore.FileManager {
	return nil
}

// SetAPIAddr sets the API address in the repo.
func (r *AferoRepo) SetAPIAddr(addr ma.Multiaddr) error {
	return errors.New("AferoRepo.SetAPIAddr not implemented")
}

const swarmKeyFile = "swarm.key"

// SwarmKey returns the configured shared symmetric key for the private networks feature.
func (r *AferoRepo) SwarmKey() ([]byte, error) {
	// FIXME: filepath usage will produce different paths on different platforms
	// So if we have a virtual fs used on *NIX then transferred on windows it will not work

	repoPath := filepath.Clean(r.path)
	spath := filepath.Join(repoPath, swarmKeyFile)

	f, err := r.fs.Open(spath)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}
	defer f.Close()

	return afero.ReadAll(f)
}

// Close see io.Closer
func (r *AferoRepo) Close() error {
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return errors.New("repo is closed")
	}

	err := r.fs.Remove(filepath.Join(r.path, apiFile))
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("error removing api file: ", err)
	}

	if err := r.ds.Close(); err != nil {
		return err
	}

	// This code existed in the previous versions, but
	// EventlogComponent.Close was never called. Preserving here
	// pending further discussion.
	//
	// TODO It isn't part of the current contract, but callers may like for us
	// to disable logging once the component is closed.
	// logging.Configure(logging.Output(os.Stderr))

	r.closed = true
	return r.lockfile.Close()
}

// IsInitialized returns true if the repo is initialized at provided |path|.
func IsInitialized(fs afero.Fs, path string) bool {
	// packageLock is held to ensure that another caller doesn't attempt to
	// Init or Remove the repo while this call is in progress.
	packageLock.Lock()
	defer packageLock.Unlock()

	return isInitializedUnsynced(fs, path)
}
