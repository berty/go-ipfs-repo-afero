package lock

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/afero"

	lock "github.com/n0izn0iz/go-ipfs-repo-afero/pkg/go4lock"
)

// LockedError is returned as the inner error type when the lock is already
// taken.
type LockedError string

func (e LockedError) Error() string {
	return string(e)
}

// Lock creates the lock.
func Lock(fs afero.Fs, confdir, lockFileName string) (io.Closer, error) {
	lockFilePath := confdir + "/" + lockFileName
	//fmt.Println("locking", lockFilePath)
	lk, err := lock.Lock(fs, lockFilePath)
	if err != nil {
		switch {
		case lockedByOthers(err):
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
				Err:  LockedError("lock is already held by us: " + err.Error()),
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

// Locked checks if there is a lock already set.
func Locked(fs afero.Fs, confdir, lockFile string) (bool, error) {
	//fmt.Println("Checking lock")
	if exists, err := afero.Exists(fs, confdir+"/"+lockFile); err != nil || !exists {
		//fmt.Printf("File doesn't exist \"%s\":%v\n", confdir+"/"+lockFile, err)
		return false, err
	}

	lk, err := Lock(fs, confdir, lockFile)
	if err == nil {
		//fmt.Println("No one has a lock")
		lk.Close()
		return false, nil
	}

	//fmt.Println(err)

	if errors.As(err, new(LockedError)) {
		return true, nil
	}
	return false, err
}

func isLockCreatePermFail(err error) bool {
	s := err.Error()
	return strings.Contains(s, "Lock Create of") && strings.Contains(s, "permission denied")
}
