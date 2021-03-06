/*
Copyright 2013 The Go Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package lock is a file locking library using afero ported from go4.org/lock.
package go4lock

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/spf13/afero"
	"go.uber.org/zap"
)

// Lock locks the given file, creating the file if necessary. If the
// file already exists, it must have zero size or an error is returned.
// The lock is an exclusive lock (a write lock), but locked files
// should neither be read from nor written to. Such files should have
// zero size and only exist to co-ordinate ownership across processes.
//
// A nil Closer is returned if an error occurred. Otherwise, close that
// Closer to release the lock.
//
// On Linux, FreeBSD and OSX, a lock has the same semantics as fcntl(2)'s
// advisory locks.  In particular, closing any other file descriptor for the
// same file will release the lock prematurely.
//
// Attempting to lock a file that is already locked by the current process
// has undefined behavior.
//
// On other operating systems, lock will fallback to using the presence and
// content of a file named name + '.lock' to implement locking behavior.
func Lock(fs afero.Fs, name string) (io.Closer, error) {
	abs := name
	lockmu.Lock()
	defer lockmu.Unlock()
	if locked[abs] {
		return nil, fmt.Errorf("file %q already locked by us", abs)
	}

	c, err := lockFn(fs, abs)
	if err != nil {
		return nil, fmt.Errorf("cannot acquire lock: %v", err)
	}
	locked[abs] = true
	return c, nil
}

var lockFn = lockPortable

// lockPortable is a portable version not using fcntl. Doesn't handle crashes as gracefully,
// since it can leave stale lock files.
func lockPortable(fs afero.Fs, name string) (io.Closer, error) {
	/*lf, err := os.CreateTemp("", "testlog")
	if err != nil {
		return nil, errors.Wrap(err, "open log file")
	}
	fileEncoder := zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
	l := zap.New(zapcore.NewCore(fileEncoder, zapcore.AddSync(lf), zap.DebugLevel))*/
	l := zap.NewNop()

	l.Debug("locking")

	fi, err := fs.Stat(name)
	if err == nil && fi.Size() > 0 {
		st := portableLockStatus(l, fs, name)
		switch st {
		case statusLocked:
			return nil, fmt.Errorf("file %q already locked", name)
		case statusLockedByOther:
			return nil, fmt.Errorf("file %q locked by other", name)
		case statusStale:
			fs.Remove(name)
		case statusInvalid:
			return nil, fmt.Errorf("can't Lock file %q: has invalid contents", name)
		}
	}
	f, err := fs.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file %s %v", name, err)
	}
	//fmt.Println("created file", name)
	if err := json.NewEncoder(f).Encode(&pidLockMeta{OwnerPID: os.Getpid()}); err != nil {
		return nil, fmt.Errorf("cannot write owner pid: %v", err)
	}
	return &unlocker{
		l:        l,
		f:        f,
		fs:       fs,
		abs:      name,
		portable: true,
	}, nil
}

type lockStatus int

const (
	statusInvalid lockStatus = iota
	statusLocked
	statusUnlocked
	statusStale
	statusLockedByOther
)

type pidLockMeta struct {
	OwnerPID int
}

func portableLockStatus(l *zap.Logger, fs afero.Fs, path string) lockStatus {
	f, err := fs.Open(path)
	if err != nil {
		return statusUnlocked
	}
	defer f.Close()
	var meta pidLockMeta
	if json.NewDecoder(f).Decode(&meta) != nil {
		return statusInvalid
	}
	if meta.OwnerPID == 0 {
		return statusInvalid
	}
	p, err := os.FindProcess(meta.OwnerPID)
	if err != nil {
		// e.g. on Windows
		return statusStale
	}
	// On unix, os.FindProcess always is true, so we have to send
	// it a signal to see if it's alive.

	l.Debug("sending signal")
	if signalZero != nil {
		if p.Signal(signalZero) != nil {
			return statusStale
		} else {
			return statusLockedByOther
		}
	}

	l.Debug("already locked")
	return statusLocked
}

var signalZero os.Signal // nil or set by lock_sigzero.go

var (
	lockmu sync.Mutex
	locked = map[string]bool{} // abs path -> true
)

type unlocker struct {
	l        *zap.Logger
	portable bool
	fs       afero.Fs
	f        afero.File
	abs      string
	// once guards the close method call.
	once sync.Once
	// err holds the error returned by Close.
	err error
}

func (u *unlocker) Close() error {
	u.l.Debug("removing lock")

	u.once.Do(u.close)
	return u.err
}

func (u *unlocker) close() {
	lockmu.Lock()
	defer lockmu.Unlock()
	delete(locked, u.abs)

	if u.portable {
		// In the portable lock implementation, it's
		// important to close before removing because
		// Windows won't allow us to remove an open
		// file.
		if err := u.f.Close(); err != nil {
			u.err = err
		}
		if err := u.fs.Remove(u.abs); err != nil {
			// Note that if both Close and Remove fail,
			// we care more about the latter than the former
			// so we'll return that error.
			u.err = err
		}
		u.l.Debug("removed lock")
		return
	}
	// In other implementatioons, it's nice for us to clean up.
	// If we do do this, though, it needs to be before the
	// u.f.Close below.
	u.fs.Remove(u.abs)
	u.err = u.f.Close()
}
