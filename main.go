package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"runtime/debug"

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
)

func serveFS(fs afero.Fs) {
	httpFs := afero.NewHttpFs(fs)
	fileserver := http.FileServer(httpFs.Dir("/"))
	http.Handle("/", fileserver)
	http.ListenAndServe("localhost:8080", nil)
}

func main() {
	// mem
	memFS := afero.NewMemMapFs()
	testFs(memFS, "/mem-repo")

	// fs
	testFs(afero.NewOsFs(), "./test-repo")

	// serve mem for inspection
	serveFS(memFS)
}

func ipfsGet(ctx context.Context, api iface.CoreAPI, cid cid.Cid) []byte {
	ipfsNode, err := api.Unixfs().Get(ctx, path.IpfsPath(cid))
	if err != nil {
		panic(errors.Wrap(err, "ipfs get"))
	}
	defer closeOrPanic(ipfsNode)

	file := ipfsfiles.ToFile(ipfsNode)
	defer closeOrPanic(file)

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		panic(errors.Wrap(err, "read file"))
	}

	return fileBytes
}

func testFs(fs afero.Fs, rpath string) {
	fmt.Printf("Testing %s:%s\n", fs.Name(), rpath)

	defer func() {
		if a := recover(); a != nil {
			fmt.Printf("Panic: %v\n", a)
			debug.PrintStack()
			fmt.Printf("Spinning http server to examine fs")
			serveFS(fs)
		}
	}()

	dsfs = fs

	if IsInitialized(fs, rpath) {
		panic(fmt.Errorf("repo %s:%s already initialized", fs.Name(), rpath))
	}

	conf, err := genConfig()
	if err != nil {
		panic(errors.Wrap(err, "gen config"))
	}

	if err := Init(fs, rpath, conf); err != nil {
		panic(errors.Wrap(err, "init repo"))
	}

	repo, err := Open(fs, rpath)
	if err != nil {
		panic(errors.Wrap(err, "open repo"))
	}
	defer func() {
		if err := repo.Close(); err != nil && err.Error() != "repo is closed" {
			panic(errors.Wrap(err, "close repo"))
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
	if err != nil {
		panic(errors.Wrap(err, "new ipfs node"))
	}
	defer closeOrPanic(inode)

	// create ipfs api
	api, err := coreapi.NewCoreAPI(inode, options.Api.FetchBlocks(true))
	if err != nil {
		panic(errors.Wrap(err, "new ipfs api"))
	}

	// cast cid
	cid, err := cid.Decode("QmPZ9gcCEpqKTo6aq61g2nXGUhM4iCL3ewB6LDXZCtioEB")
	if err != nil {
		panic(errors.Wrap(err, "decode cid"))
	}

	// cat ipfs readme
	fileBytes := ipfsGet(ctx, api, cid)
	fmt.Println(string(fileBytes))

	// declare test data
	testData := []byte("Hello world!")

	// pin test data
	ipath, err := api.Unixfs().Add(ctx, ipfsfiles.NewBytesFile(testData))
	if err != nil {
		panic(errors.Wrap(err, "ipfs add"))
	}
	fmt.Println("added", ipath)

	// get test data
	testBytes := ipfsGet(ctx, api, ipath.Cid())
	if !bytes.Equal(testData, testBytes) {
		panic(fmt.Errorf(`data diff, expected "%s", got "%s"`, testData, testBytes))
	}
}

func closeOrPanic(closer io.Closer) {
	if err := closer.Close(); err != nil {
		panic(errors.Wrap(err, "close"))
	}
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
