// Package atomicfile provides the ability to write a file with an eventual
// rename on Close (using osRename). This allows for a file to always be in a
// consistent state and never represent an in-progress write.
//
// NOTE: `osRename` may not be atomic on your operating system.
package atomicfile

import (
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// File behaves like os.File, but does an atomic rename operation at Close.
type File struct {
	afero.File
	fs   afero.Fs
	path string
}

// New creates a new temporary file that will replace the file at the given
// path when Closed.
func New(fs afero.Fs, path string, mode os.FileMode) (*File, error) {
	f, err := afero.TempFile(fs, filepath.Dir(path), filepath.Base(path))
	if err != nil {
		return nil, err
	}
	if err := fs.Chmod(f.Name(), mode); err != nil {
		f.Close()
		fs.Remove(f.Name())
		return nil, err
	}
	return &File{File: f, path: path, fs: fs}, nil
}

// Close the file replacing the configured file.
func (f *File) Close() error {
	if err := f.File.Close(); err != nil {
		f.fs.Remove(f.File.Name())
		return err
	}
	if err := f.fs.Rename(f.Name(), f.path); err != nil {
		return err
	}
	return nil
}

// Abort closes the file and removes it instead of replacing the configured
// file. This is useful if after starting to write to the file you decide you
// don't want it anymore.
func (f *File) Abort() error {
	if err := f.File.Close(); err != nil {
		f.fs.Remove(f.Name())
		return err
	}
	if err := f.fs.Remove(f.Name()); err != nil {
		return err
	}
	return nil
}
