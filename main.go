package main

import (
	"fmt"
	"os"

	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/spf13/afero"
)

func main() {
	testFs(afero.NewOsFs())
	testFs(afero.NewMemMapFs())
}

func testFs(fs afero.Fs) {
	fmt.Println("Testing", fs.Name())

	conf, err := genConfig()
	if err != nil {
		panic(err)
	}

	rpath := "./test-repo"

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
	return config.InitWithIdentity(identity)
}
