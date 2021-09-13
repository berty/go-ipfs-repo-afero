package repo

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/ipfs/go-ipfs/thirdparty/assert"
	"github.com/spf13/afero"

	datastore "github.com/ipfs/go-datastore"
	config "github.com/ipfs/go-ipfs-config"
)

// swap arg order
func testRepoPath(fs afero.Fs, p string, t *testing.T) string {
	name, err := afero.TempDir(fs, "", p)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func TestInitIdempotence(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Parallel()
	path := testRepoPath(fs, "", t)
	for i := 0; i < 10; i++ {
		assert.Nil(Init(fs, path, &config.Config{Datastore: DefaultDatastoreConfig()}), t, "multiple calls to init should succeed")
	}
}

func Remove(fs afero.Fs, repoPath string) error {
	repoPath = filepath.Clean(repoPath)
	return fs.RemoveAll(repoPath)
}

func TestCanManageReposIndependently(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()

	pathA := testRepoPath(fs, "a", t)
	pathB := testRepoPath(fs, "b", t)

	t.Log("initialize two repos")
	assert.Nil(Init(fs, pathA, &config.Config{Datastore: DefaultDatastoreConfig()}), t, "a", "should initialize successfully")
	assert.Nil(Init(fs, pathB, &config.Config{Datastore: DefaultDatastoreConfig()}), t, "b", "should initialize successfully")

	t.Log("ensure repos initialized")
	assert.True(IsInitialized(fs, pathA), t, "a should be initialized")
	assert.True(IsInitialized(fs, pathB), t, "b should be initialized")

	t.Log("open the two repos")
	repoA, err := Open(fs, pathA)
	assert.Nil(err, t, "a")
	repoB, err := Open(fs, pathB)
	assert.Nil(err, t, "b")

	t.Log("close and remove b while a is open")
	assert.Nil(repoB.Close(), t, "close b")
	assert.Nil(Remove(fs, pathB), t, "remove b")

	t.Log("close and remove a")
	assert.Nil(repoA.Close(), t)
	assert.Nil(Remove(fs, pathA), t)
}

func TestDatastoreGetNotAllowedAfterClose(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()

	path := testRepoPath(fs, "test", t)

	assert.True(!IsInitialized(fs, path), t, "should NOT be initialized")
	assert.Nil(Init(fs, path, &config.Config{Datastore: DefaultDatastoreConfig()}), t, "should initialize successfully")
	r, err := Open(fs, path)
	assert.Nil(err, t, "should open successfully")

	k := "key"
	data := []byte(k)
	assert.Nil(r.Datastore().Put(datastore.NewKey(k), data), t, "Put should be successful")

	assert.Nil(r.Close(), t)
	_, err = r.Datastore().Get(datastore.NewKey(k))
	assert.Err(err, t, "after closer, Get should be fail")
}

func TestDatastorePersistsFromRepoToRepo(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()

	path := testRepoPath(fs, "test", t)

	assert.Nil(Init(fs, path, &config.Config{Datastore: DefaultDatastoreConfig()}), t)
	r1, err := Open(fs, path)
	assert.Nil(err, t)

	k := "key"
	expected := []byte(k)
	assert.Nil(r1.Datastore().Put(datastore.NewKey(k), expected), t, "using first repo, Put should be successful")
	assert.Nil(r1.Close(), t)

	r2, err := Open(fs, path)
	assert.Nil(err, t)
	actual, err := r2.Datastore().Get(datastore.NewKey(k))
	assert.Nil(err, t, "using second repo, Get should be successful")
	assert.Nil(r2.Close(), t)
	assert.True(bytes.Equal(expected, actual), t, "data should match")
}

func TestOpenMoreThanOnceInSameProcess(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()

	path := testRepoPath(fs, "", t)
	assert.Nil(Init(fs, path, &config.Config{Datastore: DefaultDatastoreConfig()}), t)

	r1, err := Open(fs, path)
	assert.Nil(err, t, "first repo should open successfully")
	r2, err := Open(fs, path)
	assert.Nil(err, t, "second repo should open successfully")
	assert.True(r1 == r2, t, "second open returns same value")

	assert.Nil(r1.Close(), t)
	assert.Nil(r2.Close(), t)
}
