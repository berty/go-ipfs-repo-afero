package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// Writable ensures the directory exists and is writable
func Writable(fs afero.Fs, path string) error {
	// Construct the path if missing
	if err := fs.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
	// Check the directory is writable
	if f, err := fs.Create(filepath.Join(path, "._check_writable")); err == nil {
		f.Close()
		fs.Remove(f.Name())
	} else {
		return errors.New("'" + path + "' is not writable")
	}
	return nil
}
