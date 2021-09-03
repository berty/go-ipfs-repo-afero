package main

import (
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	serialize "github.com/ipfs/go-ipfs-config/serialize"
	"github.com/mitchellh/go-homedir"
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
		return err
	}
	conf, err := serialize.Load(configFilename)
	if err != nil {
		return err
	}
	r.config = conf
	return nil
}
