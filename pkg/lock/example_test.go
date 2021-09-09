package lock_test

import (
	"errors"
	"fmt"

	fslock "github.com/n0izn0iz/go-ipfs-repo-afero/pkg/lock"
	"github.com/spf13/afero"
)

func ExampleLockedError() {
	fs := afero.NewMemMapFs()

	tempdir, err := afero.TempDir(fs, "/tmp", "")
	if err != nil {
		panic(err)
	}

	_, err = fslock.Lock(fs, tempdir, "foo.lock")
	fmt.Println("locked:", errors.As(err, new(fslock.LockedError)))

	_, err = fslock.Lock(fs, tempdir, "foo.lock")
	fmt.Println("locked:", errors.As(err, new(fslock.LockedError)))
	// Output:
	// locked: false
	// locked: true
}
