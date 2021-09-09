package lock_test

import (
	"bufio"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	lock "github.com/n0izn0iz/go-ipfs-repo-afero/pkg/lock"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func assertLock(t *testing.T, fs afero.Fs, confdir, lockFile string, expected bool) {
	t.Helper()

	isLocked, err := lock.Locked(fs, confdir, lockFile)
	if err != nil {
		t.Fatal(err)
	}

	if isLocked != expected {
		t.Fatalf("expected %t to be %t at %s/%s", isLocked, expected, confdir, lockFile)
	}
}

func TestLockSimple(t *testing.T) {
	lockFile := "my-test.lock"

	fs := afero.NewMemMapFs()

	confdir := "./test"

	assertLock(t, fs, confdir, lockFile, false)

	lockfile, err := lock.Lock(fs, confdir, lockFile)
	if err != nil {
		t.Fatal(err)
	}

	assertLock(t, fs, confdir, lockFile, true)

	if err := lockfile.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, fs, confdir, lockFile, false)

	// second round of locking

	lockfile, err = lock.Lock(fs, confdir, lockFile)
	if err != nil {
		t.Fatal(err)
	}

	assertLock(t, fs, confdir, lockFile, true)

	if err := lockfile.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, fs, confdir, lockFile, false)
}

func TestLockMultiple(t *testing.T) {
	lockFile1 := "test-1.lock"
	lockFile2 := "test-2.lock"

	fs := afero.NewMemMapFs()

	confdir, err := afero.TempDir(fs, "/tmp", "test")
	require.NoError(t, err)

	// make sure we start clean
	_ = fs.Remove(path.Join(confdir, lockFile1))
	_ = fs.Remove(path.Join(confdir, lockFile2))

	lock1, err := lock.Lock(fs, confdir, lockFile1)
	if err != nil {
		t.Fatal(err)
	}
	lock2, err := lock.Lock(fs, confdir, lockFile2)
	if err != nil {
		t.Fatal(err)
	}

	assertLock(t, fs, confdir, lockFile1, true)
	assertLock(t, fs, confdir, lockFile2, true)

	if err := lock1.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, fs, confdir, lockFile1, false)
	assertLock(t, fs, confdir, lockFile2, true)

	if err := lock2.Close(); err != nil {
		t.Fatal(err)
	}

	assertLock(t, fs, confdir, lockFile1, false)
	assertLock(t, fs, confdir, lockFile2, false)
}

func TestLockedByOthers(t *testing.T) {
	const (
		lockFile = "my-test.lock"
		wantErr  = "someone else has the lock"
	)

	fs := afero.NewOsFs()

	confdir := t.TempDir()
	lockedMsg := "locking " + confdir + "/" + lockFile + "\n"

	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" { // child process
		confdir := os.Args[3]
		lockedMsg := "locking " + confdir + "/" + lockFile + "\n"
		lock, err := lock.Lock(fs, confdir, lockFile)
		if err != nil {
			t.Fatalf("child lock: %v", err)
		}
		defer func() {
			require.NoError(t, lock.Close())
		}()
		os.Stdout.WriteString(lockedMsg)
		time.Sleep(10 * time.Minute)
		return
	}

	// Execute a child process that locks the file.
	cmd := exec.Command(os.Args[0], "-test.run=TestLockedByOthers", "--", confdir)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("cmd.StdoutPipe: %v", err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatalf("cmd.Start: %v", err)
	}
	defer cmd.Process.Kill()

	// Wait for the child to lock the file.
	b := bufio.NewReader(stdout)
	line, err := b.ReadString('\n')
	if err != nil {
		t.Fatalf("read from child: %v", err)
	}
	if got, want := line, lockedMsg; got != want {
		t.Fatalf("got %q from child; want %q", got, want)
	}

	// Parent should not be able to lock the file.
	_, err = lock.Lock(fs, confdir, lockFile)
	if err == nil {
		t.Fatalf("parent successfully acquired the lock")
	}
	pe, ok := err.(*os.PathError)
	if !ok {
		t.Fatalf("wrong error type %T: %s", err, err.Error())
	}
	if got := pe.Error(); !strings.Contains(got, wantErr) {
		t.Fatalf("error %q does not contain %q", got, wantErr)
	}
}
