package lock

import (
	"errors"
	"fmt"

	"github.com/spf13/afero"
)

func ExampleLockedError() {
	fs := afero.NewMemMapFs()

	tmpDir, err := afero.TempDir(fs, "", "")
	if err != nil {
		panic(err)
	}

	_, err = Lock(fs, tmpDir, "foo.lock")
	fmt.Println("locked:", errors.As(err, new(LockedError)))

	_, err = Lock(fs, tmpDir, "foo.lock")
	fmt.Println("locked:", errors.As(err, new(LockedError)))

	// Output:
	// locked: false
	// locked: true
}
