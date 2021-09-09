package repo

import (
	"os"

	"github.com/spf13/afero"
)

// FileExists check if the file with the given path exits.
func FileExists(fs afero.Fs, filename string) bool {
	fi, err := fs.Stat(filename) // FIXME: use afero.Lstater if possible
	if fi != nil || (err != nil && !os.IsNotExist(err)) {
		return true
	}
	return false
}
