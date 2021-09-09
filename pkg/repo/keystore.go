package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	base32 "encoding/base32"

	"github.com/apex/log"
	keystore "github.com/ipfs/go-ipfs/keystore"
	ci "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/spf13/afero"
)

var codec = base32.StdEncoding.WithPadding(base32.NoPadding)

const keyFilenamePrefix = "key_"

// AferoKeystore is a keystore backed by files in a given directory stored on disk.
type AferoKeystore struct {
	dir string
	fs  afero.Fs
}

// NewAferoKeystore returns a new filesystem-backed keystore.
func NewAferoKeystore(fs afero.Fs, dir string) (*AferoKeystore, error) {
	err := fs.Mkdir(dir, 0700)
	switch {
	case os.IsExist(err):
	case err == nil:
	default:
		return nil, err
	}
	return &AferoKeystore{dir, fs}, nil
}

// Has returns whether or not a key exists in the Keystore
func (ks *AferoKeystore) Has(name string) (bool, error) {
	name, err := keystoreEncode(name)
	if err != nil {
		return false, err
	}

	kp := filepath.Join(ks.dir, name)

	_, err = ks.fs.Stat(kp)

	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

// Put stores a key in the Keystore, if a key with the same name already exists, returns ErrKeyExists
func (ks *AferoKeystore) Put(name string, k ci.PrivKey) error {
	name, err := keystoreEncode(name)
	if err != nil {
		return err
	}

	b, err := ci.MarshalPrivateKey(k)
	if err != nil {
		return err
	}

	kp := filepath.Join(ks.dir, name)

	fi, err := ks.fs.OpenFile(kp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0400)
	if err != nil {
		if os.IsExist(err) {
			err = keystore.ErrKeyExists
		}
		return err
	}
	defer fi.Close()

	_, err = fi.Write(b)

	return err
}

// Get retrieves a key from the Keystore if it exists, and returns ErrNoSuchKey
// otherwise.
func (ks *AferoKeystore) Get(name string) (ci.PrivKey, error) {
	name, err := keystoreEncode(name)
	if err != nil {
		return nil, err
	}

	kp := filepath.Join(ks.dir, name)

	data, err := afero.ReadFile(ks.fs, kp)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, keystore.ErrNoSuchKey
		}
		return nil, err
	}

	return ci.UnmarshalPrivateKey(data)
}

// Delete removes a key from the Keystore
func (ks *AferoKeystore) Delete(name string) error {
	name, err := keystoreEncode(name)
	if err != nil {
		return err
	}

	kp := filepath.Join(ks.dir, name)

	return ks.fs.Remove(kp)
}

// List return a list of key identifier
func (ks *AferoKeystore) List() ([]string, error) {
	dir, err := ks.fs.Open(ks.dir)
	if err != nil {
		return nil, err
	}

	dirs, err := dir.Readdirnames(0)
	if err != nil {
		return nil, err
	}

	list := make([]string, 0, len(dirs))

	for _, name := range dirs {
		decodedName, err := decode(name)
		if err == nil {
			list = append(list, decodedName)
		} else {
			log.Errorf("Ignoring keyfile with invalid encoded filename: %s", name)
		}
	}

	return list, nil
}

func keystoreEncode(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("key name must be at least one character")
	}

	encodedName := codec.EncodeToString([]byte(name))
	log.Debugf("Encoded key name: %s to: %s", name, encodedName)

	return keyFilenamePrefix + strings.ToLower(encodedName), nil
}

func decode(name string) (string, error) {
	if !strings.HasPrefix(name, keyFilenamePrefix) {
		return "", fmt.Errorf("key's filename has unexpected format")
	}

	nameWithoutPrefix := strings.ToUpper(name[len(keyFilenamePrefix):])
	decodedName, err := codec.DecodeString(nameWithoutPrefix)
	if err != nil {
		return "", err
	}

	log.Debugf("Decoded key name: %s to: %s", name, decodedName)

	return string(decodedName), nil
}
