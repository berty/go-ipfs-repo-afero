package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	measure "github.com/ipfs/go-ds-measure"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

const programTooLowMessage = `your programs version (%d) is lower than your repos (%d)`

func newAferoRepo(fs afero.Fs, rpath string) (*AferoRepo, error) {
	expPath, err := homedir.Expand(filepath.Clean(rpath))
	if err != nil {
		return nil, err
	}

	return &AferoRepo{fs: fs, path: expPath}, nil
}

// openConfig returns an error if the config file is not present.
func (r *AferoRepo) openConfig() error {
	configFilename, err := config.Filename(r.path)
	if err != nil {
		return errors.Wrap(err, "get config filename")
	}
	conf, err := Load(r.fs, configFilename)
	if err != nil {
		return errors.Wrap(err, "load config")
	}
	r.config = conf
	return nil
}

// openDatastore returns an error if the config file is not present.
func (r *AferoRepo) openDatastore() error {
	if r.config.Datastore.Type != "" || r.config.Datastore.Path != "" {
		return fmt.Errorf("old style datatstore config detected")
	} else if r.config.Datastore.Spec == nil {
		return fmt.Errorf("required Datastore.Spec entry missing from config file")
	}
	if r.config.Datastore.NoSync {
		log.Warn("NoSync is now deprecated in favor of datastore specific settings. If you want to disable fsync on flatfs set 'sync' to false. See https://github.com/ipfs/go-ipfs/blob/master/docs/datastores.md#flatfs.")
	}

	dsc, err := AnyDatastoreConfig(r.config.Datastore.Spec)
	if err != nil {
		return errors.Wrap(err, "get datastore config")
	}
	spec := dsc.DiskSpec()

	oldSpec, err := r.readSpec()
	if err != nil {
		return err
	}
	if oldSpec != spec.String() {
		return fmt.Errorf("datastore configuration of '%s' does not match what is on disk '%s'",
			oldSpec, spec.String())
	}

	d, err := dsc.Create(r.path)
	if err != nil {
		return errors.Wrap(err, "create datastore")
	}
	r.ds = d

	// Wrap it with metrics gathering
	prefix := "ipfs.fsrepo.datastore"
	r.ds = measure.New(prefix, r.ds)

	return nil
}

func (r *AferoRepo) readSpec() (string, error) {
	fn, err := config.Path(r.path, specFn)
	if err != nil {
		return "", err
	}
	b, err := afero.ReadFile(r.fs, fn)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}
