package atomicfile_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/n0izn0iz/go-ipfs-repo-afero/pkg/atomicfile"
	"github.com/spf13/afero"
)

func test(t *testing.T, dir, prefix string) {
	t.Parallel()

	fs := afero.NewMemMapFs()

	tmpfile, err := afero.TempFile(fs, dir, prefix)
	if err != nil {
		t.Fatal(err)
	}
	name := tmpfile.Name()

	if err := fs.Remove(name); err != nil {
		t.Fatal(err)
	}

	defer fs.Remove(name)
	f, err := atomicfile.New(fs, name, os.FileMode(0666))
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("foo"))
	if _, err := fs.Stat(name); !os.IsNotExist(err) {
		t.Fatal("did not expect file to exist")
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(name); err != nil {
		t.Fatalf("expected file to exist: %s", err)
	}
}

func TestCurrentDir(t *testing.T) {
	test(t, "/", "atomicfile-current-dir-")
}

func TestRootTmpDir(t *testing.T) {
	test(t, "/tmp", "atomicfile-root-tmp-dir-")
}

func TestDefaultTmpDir(t *testing.T) {
	test(t, "", "atomicfile-default-tmp-dir-")
}

func TestAbort(t *testing.T) {
	contents := []byte("the answer is 42")
	t.Parallel()

	fs := afero.NewMemMapFs()

	tmpfile, err := afero.TempFile(fs, "", "atomicfile-abort-")
	if err != nil {
		t.Fatal(err)
	}
	name := tmpfile.Name()
	if _, err := tmpfile.Write(contents); err != nil {
		t.Fatal(err)
	}
	defer fs.Remove(name)

	f, err := atomicfile.New(fs, name, os.FileMode(0666))
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("foo"))
	if err := f.Abort(); err != nil {
		t.Fatal(err)
	}
	if _, err := fs.Stat(name); err != nil {
		t.Fatalf("expected file to exist: %s", err)
	}
	actual, err := afero.ReadFile(fs, name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, actual) {
		t.Fatalf(`did not find expected "%s" instead found "%s"`, contents, actual)
	}
}
