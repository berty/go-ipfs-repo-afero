package repo_test

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/berty/go-ipfs-repo-afero/pkg/repo"
	"github.com/ipfs/go-cid"
	config "github.com/ipfs/go-ipfs-config"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	ipfs_core "github.com/ipfs/go-ipfs/core"
	coreapi "github.com/ipfs/go-ipfs/core/coreapi"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func serveFS(fs afero.Fs) {
	httpFs := afero.NewHttpFs(fs)
	fileserver := http.FileServer(httpFs.Dir("/"))
	http.Handle("/", fileserver)
	http.ListenAndServe("localhost:8080", nil)
}

func TestMemMapRepo(t *testing.T) {
	testFs(t, afero.NewMemMapFs(), "/mem-repo")
}

func TestOsRepo(t *testing.T) {
	testFs(t, afero.NewOsFs(), t.TempDir())
}

func ipfsGet(t *testing.T, ctx context.Context, api iface.CoreAPI, cid cid.Cid) []byte {
	ipfsNode, err := api.Unixfs().Get(ctx, path.IpfsPath(cid))
	require.NoError(t, err)
	defer closeOrPanic(t, ipfsNode)

	file := ipfsfiles.ToFile(ipfsNode)
	defer closeOrPanic(t, file)

	fileBytes, err := ioutil.ReadAll(file)
	require.NoError(t, err)

	return fileBytes
}

func testFs(t *testing.T, fs afero.Fs, rpath string) {
	t.Helper()

	t.Logf("Testing %s:%s", fs.Name(), rpath)

	/*defer func() {
		if a := recover(); a != nil {
			fmt.Printf("Panic: %v\n", a)
			debug.PrintStack()
			fmt.Printf("Spinning http server to examine fs")
			serveFS(fs)
		}
	}()*/

	repo.DsFs = fs

	initialized := repo.IsInitialized(fs, rpath)
	require.False(t, initialized)

	conf, err := genConfig()
	require.NoError(t, err)

	repo.Init(fs, rpath, conf)
	require.NoError(t, err)

	repo, err := repo.Open(fs, rpath)
	require.NoError(t, err)
	defer func() {
		if err := repo.Close(); err != nil && err.Error() != "repo is closed" {
			require.NoError(t, err)
		}
	}()

	// create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create ipfs node
	buildcfg := &ipfs_core.BuildCfg{
		Online: true,
		Repo:   repo,
	}
	inode, err := ipfs_core.NewNode(ctx, buildcfg)
	require.NoError(t, err)
	defer closeOrPanic(t, inode)

	// create ipfs api
	api, err := coreapi.NewCoreAPI(inode, options.Api.FetchBlocks(true))
	require.NoError(t, err)

	// cast cid
	cid, err := cid.Decode("QmPZ9gcCEpqKTo6aq61g2nXGUhM4iCL3ewB6LDXZCtioEB")
	require.NoError(t, err)

	// cat ipfs readme
	fileBytes := ipfsGet(t, ctx, api, cid)
	fmt.Println(string(fileBytes)) // FIXME: check that the file is correct

	// declare test data
	testData := []byte("Hello world!")

	// pin test data
	ipath, err := api.Unixfs().Add(ctx, ipfsfiles.NewBytesFile(testData))
	if err != nil {
		panic(errors.Wrap(err, "ipfs add"))
	}
	fmt.Println("added", ipath)

	// get test data
	testBytes := ipfsGet(t, ctx, api, ipath.Cid())
	require.Equal(t, testData, testBytes)
}

func closeOrPanic(t *testing.T, closer io.Closer) {
	err := closer.Close()
	require.NoError(t, err)
}

func genConfig() (*config.Config, error) {
	var err error
	var identity config.Identity

	identity, err = config.CreateIdentity(os.Stdout, []options.KeyGenerateOption{
		options.Key.Type(options.Ed25519Key),
	})
	if err != nil {
		return nil, err
	}

	conf, err := config.InitWithIdentity(identity)
	if err != nil {
		return nil, errors.Wrap(err, "init config")
	}

	conf.Datastore = DefaultDatastoreConfig()

	return conf, nil
}

// DefaultDatastoreConfig is an internal function exported to aid in testing.
func DefaultDatastoreConfig() config.Datastore {
	return config.Datastore{
		StorageMax:         "10GB",
		StorageGCWatermark: 90, // 90%
		GCPeriod:           "1h",
		BloomFilterSize:    0,
		Spec:               flatfsSpec(),
	}
}

func flatfsSpec() map[string]interface{} {
	return map[string]interface{}{
		"type": "mount",
		"mounts": []interface{}{
			map[string]interface{}{
				"mountpoint": "/blocks",
				"type":       "measure",
				"prefix":     "afero.blocks",
				"child": map[string]interface{}{
					"type": "afero",
					"path": "blocks",
					/*"sync":      true,
					"shardFunc": "/repo/flatfs/shard/v1/next-to-last/2",*/
				},
			},
			map[string]interface{}{
				"mountpoint": "/",
				"type":       "measure",
				"prefix":     "afero.datastore",
				"child": map[string]interface{}{
					"type": "afero",
					"path": "datastore",
					//"compression": "none",
				},
			},
		},
	}
}
