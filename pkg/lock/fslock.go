package lock

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	lock "github.com/berty/go-ipfs-repo-afero/pkg/go4lock"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/afero"
)

// log is the fsrepo logger
var log = logging.Logger("lock")

// LockedError is returned as the inner error type when the lock is already
// taken.
type LockedError string

func (e LockedError) Error() string {
	return string(e)
}

// Lock creates the lock.
func Lock(fs afero.Fs, confdir, lockFileName string) (io.Closer, error) {
	lockFilePath := filepath.Join(confdir, lockFileName)
	lk, err := lock.Lock(fs, lockFilePath)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "locked by other"):
			return lk, &os.PathError{
				Op:   "lock",
				Path: lockFilePath,
				Err:  LockedError("someone else has the lock"),
			}
		case strings.Contains(err.Error(), "already locked"):
			// we hold the lock ourselves
			return lk, &os.PathError{
				Op:   "lock",
				Path: lockFilePath,
				Err:  LockedError("lock is already held by us"),
			}
		case os.IsPermission(err) || isLockCreatePermFail(err):
			// lock fails on permissions error

			// Using a path error like this ensures that
			// os.IsPermission works on the returned error.
			return lk, &os.PathError{
				Op:   "lock",
				Path: lockFilePath,
				Err:  os.ErrPermission,
			}
		}
	}
	return lk, err
}

// FileExists check if the file with the given path exits.
func FileExists(fs afero.Fs, filename string) bool {
	fi, err := fs.Stat(filename)
	if fi != nil || (err != nil && !os.IsNotExist(err)) {
		return true
	}
	return false
}

// Locked checks if there is a lock already set.
func Locked(fs afero.Fs, confdir, lockFile string) (bool, error) {
	log.Debugf("Checking lock")
	if !FileExists(fs, filepath.Join(confdir, lockFile)) {
		log.Debugf("File doesn't exist: %s", filepath.Join(confdir, lockFile))
		return false, nil
	}

	lk, err := Lock(fs, confdir, lockFile)
	if err == nil {
		log.Debugf("No one has a lock")
		lk.Close()
		return false, nil
	}

	log.Debug(err)

	if errors.As(err, new(LockedError)) {
		return true, nil
	}
	return false, err
}

func isLockCreatePermFail(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Lock Create of") && strings.Contains(s, "permission denied")
}
