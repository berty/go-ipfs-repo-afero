package main

import (
	config "github.com/ipfs/go-ipfs-config"
	"github.com/spf13/afero"
)

const specFn = "datastore_spec"

// Init initializes a new FSRepo at the given path with the provided config.
// TODO add support for custom datastores.
func Init(fs afero.Fs, repoPath string, conf *config.Config) error {
	/*
		FIXME
		// packageLock must be held to ensure that the repo is not initialized more
		// than once.
		packageLock.Lock()
		defer packageLock.Unlock()
	*/

	if isInitializedUnsynced(fs, repoPath) {
		return nil
	}

	if err := initConfig(fs, repoPath, conf); err != nil {
		return err
	}

	if err := initSpec(fs, repoPath, conf.Datastore.Spec); err != nil {
		return err
	}

	if err := WriteRepoVersion(fs, repoPath, RepoVersion); err != nil {
		return err
	}

	return nil
}

func initConfig(fs afero.Fs, path string, conf *config.Config) error {
	if configIsInitialized(fs, path) {
		return nil
	}
	configFilename, err := config.Filename(path)
	if err != nil {
		return err
	}
	// initialization is the one time when it's okay to write to the config
	// without reading the config from disk and merging any user-provided keys
	// that may exist.
	if err := WriteConfigFile(fs, configFilename, conf); err != nil {
		return err
	}

	return nil
}

func initSpec(fs afero.Fs, path string, conf map[string]interface{}) error {
	fn, err := config.Path(path, specFn)
	if err != nil {
		return err
	}

	if FileExists(fs, fn) {
		return nil
	}

	dsc, err := AnyDatastoreConfig(conf)
	if err != nil {
		return err
	}
	bytes := dsc.DiskSpec().Bytes()

	return afero.WriteFile(fs, fn, bytes, 0600)
}
