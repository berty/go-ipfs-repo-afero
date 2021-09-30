package repo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/ipfs/go-datastore"
	filestore "github.com/ipfs/go-filestore"
	keystore "github.com/ipfs/go-ipfs-keystore"
	repo "github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/common"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/multierr"

	config "github.com/ipfs/go-ipfs-config"
	ma "github.com/multiformats/go-multiaddr"

	lockfile "github.com/berty/go-ipfs-repo-afero/pkg/lock"
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
	onlyOne repo.OnlyOne
)

func Open(fs afero.Fs, repoPath string) (repo.Repo, error) {
	fn := func() (repo.Repo, error) {
		return open(fs, repoPath)
	}
	return onlyOne.Open(repoPath, fn)
}

func open(fs afero.Fs, repoPath string) (repo.Repo, error) {
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
		return nil, errors.Wrap(err, "lock repo")
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
		return nil, errors.Wrap(err, "open keystore")
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
	temp, err := afero.TempFile(r.fs, r.path, "config-"+prefix)
	if err != nil {
		return "", err
	}
	defer temp.Close()

	configFilename, err := config.Filename(r.path)
	if err != nil {
		return "", err
	}

	orig, err := r.fs.OpenFile(configFilename, os.O_RDONLY, 0600)
	if err != nil {
		return "", err
	}
	defer orig.Close()

	_, err = io.Copy(temp, orig)
	if err != nil {
		return "", err
	}

	return orig.Name(), nil
}

// SetConfig persists the given configuration struct to storage.
func (r *AferoRepo) SetConfig(updated *config.Config) error {
	// packageLock is held to provide thread-safety.
	packageLock.Lock()
	defer packageLock.Unlock()

	return r.setConfigUnsynced(updated)
}

// SetConfigKey sets the given key-value pair within the config and persists it to storage.
func (r *AferoRepo) SetConfigKey(key string, value interface{}) error {
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return errors.New("repo is closed")
	}

	filename, err := config.Filename(r.path)
	if err != nil {
		return err
	}
	// Load into a map so we don't end up writing any additional defaults to the config file.
	var mapconf map[string]interface{}
	if err := ReadConfigFile(r.fs, filename, &mapconf); err != nil {
		return err
	}

	// Load private key to guard against it being overwritten.
	// NOTE: this is a temporary measure to secure this field until we move
	// keys out of the config file.
	pkval, err := common.MapGetKV(mapconf, config.PrivKeySelector)
	if err != nil {
		return err
	}

	// Set the key in the map.
	if err := common.MapSetKV(mapconf, key, value); err != nil {
		return err
	}

	// replace private key, in case it was overwritten.
	if err := common.MapSetKV(mapconf, config.PrivKeySelector, pkval); err != nil {
		return err
	}

	// This step doubles as to validate the map against the struct
	// before serialization
	conf, err := config.FromMap(mapconf)
	if err != nil {
		return err
	}
	if err := WriteConfigFile(r.fs, filename, mapconf); err != nil {
		return err
	}
	return r.setConfigUnsynced(conf) // TODO roll this into this method
}

// GetConfigKey reads the value for the given key from the configuration in storage.
func (r *AferoRepo) GetConfigKey(key string) (interface{}, error) {
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return nil, errors.New("repo is closed")
	}

	filename, err := config.Filename(r.path)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	if err := ReadConfigFile(r.fs, filename, &cfg); err != nil {
		return nil, err
	}
	return common.MapGetKV(cfg, key)
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
	return datastore.DiskUsage(r.Datastore())
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
	// Create a temp file to write the address, so that we don't leave empty file when the
	// program crashes after creating the file.
	f, err := r.fs.Create(filepath.Join(r.path, "."+apiFile+".tmp"))
	if err != nil {
		return err
	}

	if _, err = f.WriteString(addr.String()); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}

	// Atomically rename the temp file to the correct file name.
	if err = r.fs.Rename(filepath.Join(r.path, "."+apiFile+".tmp"), filepath.Join(r.path,
		apiFile)); err == nil {
		return nil
	}
	// Remove the temp file when rename return error
	if err1 := r.fs.Remove(filepath.Join(r.path, "."+apiFile+".tmp")); err1 != nil {
		return multierr.Append(err, err1)
	}
	return err
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
