package repo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	config "github.com/ipfs/go-ipfs-config"
	serialize "github.com/ipfs/go-ipfs-config/serialize"
	"github.com/n0izn0iz/go-ipfs-repo-afero/pkg/atomicfile"
	"github.com/spf13/afero"
)

// WriteConfigFile writes the config from `cfg` into `filename`.
func WriteConfigFile(fs afero.Fs, filename string, cfg interface{}) error {
	err := fs.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return err
	}

	f, err := atomicfile.New(fs, filename, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return encode(f, cfg)
}

// encode configuration with JSON
func encode(w io.Writer, value interface{}) error {
	// need to prettyprint, hence MarshalIndent, instead of Encoder
	buf, err := config.Marshal(value)
	if err != nil {
		return err
	}
	_, err = w.Write(buf)
	return err
}

// ReadConfigFile reads the config from `filename` into `cfg`.
func ReadConfigFile(fs afero.Fs, filename string, cfg interface{}) error {
	f, err := fs.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			err = serialize.ErrNotInitialized
		}
		return err
	}
	defer f.Close()
	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return fmt.Errorf("failure to decode config: %s", err)
	}
	return nil
}

// Load reads given file and returns the read config, or error.
func Load(fs afero.Fs, filename string) (*config.Config, error) {
	var cfg config.Config
	err := ReadConfigFile(fs, filename, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, err
}
