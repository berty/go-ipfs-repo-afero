package main

import (
	"fmt"
	"net/http"
	"os"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/interface-go-ipfs-core/options"
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
	// fs
	testFs(afero.NewOsFs(), "./test-repo")

	// mem
	memFS := afero.NewMemMapFs()
	testFs(memFS, "/mem-repo")
	serveFS(memFS)
}

func testFs(fs afero.Fs, rpath string) {
	fmt.Printf("Testing %s:%s\n", fs.Name(), rpath)

	defer func() {
		if a := recover(); a != nil {
			fmt.Printf("Panic: %v\nSpinning http server to examine fs", a)
			serveFS(fs)
		}
	}()

	if IsInitialized(fs, rpath) {
		panic(fmt.Errorf("repo %s:%s already initialized", fs.Name(), rpath))
	}

	conf, err := genConfig()
	if err != nil {
		panic(err)
	}

	if err := Init(fs, rpath, conf); err != nil {
		panic(err)
	}

	if _, err := Open(fs, rpath); err != nil {
		panic(err)
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
