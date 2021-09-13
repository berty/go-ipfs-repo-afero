package repo

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ipfs/go-ipfs/repo"
	"github.com/spf13/afero"

	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/mount"
	dsq "github.com/ipfs/go-datastore/query"
	dssync "github.com/ipfs/go-datastore/sync"
	measure "github.com/ipfs/go-ds-measure"
)

// ConfigFromMap creates a new datastore config from a map
type ConfigFromMap func(map[string]interface{}) (DatastoreConfig, error)

// DatastoreConfig is an abstraction of a datastore config.  A "spec"
// is first converted to a DatastoreConfig and then Create() is called
// to instantiate a new datastore
type DatastoreConfig interface {
	// DiskSpec returns a minimal configuration of the datastore
	// represting what is stored on disk.  Run time values are
	// excluded.
	DiskSpec() DiskSpec

	// Create instantiate a new datastore from this config
	Create(path string) (repo.Datastore, error)
}

// DiskSpec is a minimal representation of the characteristic values of the
// datastore. If two diskspecs are the same, the loader assumes that they refer
// to exactly the same datastore. If they differ at all, it is assumed they are
// completely different datastores and a migration will be performed. Runtime
// values such as cache options or concurrency options should not be added
// here.
type DiskSpec map[string]interface{}

// Bytes returns a minimal JSON encoding of the DiskSpec
func (spec DiskSpec) Bytes() []byte {
	b, err := json.Marshal(spec)
	if err != nil {
		// should not happen
		panic(err)
	}
	return bytes.TrimSpace(b)
}

// String returns a minimal JSON encoding of the DiskSpec
func (spec DiskSpec) String() string {
	return string(spec.Bytes())
}

var datastores map[string]ConfigFromMap

func init() {
	datastores = map[string]ConfigFromMap{
		"mount":   MountDatastoreConfig,
		"mem":     MemDatastoreConfig,
		"log":     LogDatastoreConfig,
		"measure": MeasureDatastoreConfig,
		"afero":   AferoDatastoreConfig,
	}
}

func AddDatastoreConfigHandler(name string, dsc ConfigFromMap) error {
	_, ok := datastores[name]
	if ok {
		return fmt.Errorf("already have a datastore named %q", name)
	}

	datastores[name] = dsc
	return nil
}

// AnyDatastoreConfig returns a DatastoreConfig from a spec based on
// the "type" parameter
func AnyDatastoreConfig(params map[string]interface{}) (DatastoreConfig, error) {
	which, ok := params["type"].(string)
	if !ok {
		return nil, fmt.Errorf("'type' field missing or not a string")
	}
	fun, ok := datastores[which]
	if !ok {
		return nil, fmt.Errorf("unknown datastore type: %s", which)
	}
	return fun(params)
}

type mountDatastoreConfig struct {
	mounts []premount
}

type premount struct {
	ds     DatastoreConfig
	prefix ds.Key
}

// MountDatastoreConfig returns a mount DatastoreConfig from a spec
func MountDatastoreConfig(params map[string]interface{}) (DatastoreConfig, error) {
	var res mountDatastoreConfig
	mounts, ok := params["mounts"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("'mounts' field is missing or not an array")
	}
	for _, iface := range mounts {
		cfg, ok := iface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected map for mountpoint")
		}

		child, err := AnyDatastoreConfig(cfg)
		if err != nil {
			return nil, err
		}

		prefix, found := cfg["mountpoint"]
		if !found {
			return nil, fmt.Errorf("no 'mountpoint' on mount")
		}

		res.mounts = append(res.mounts, premount{
			ds:     child,
			prefix: ds.NewKey(prefix.(string)),
		})
	}
	sort.Slice(res.mounts,
		func(i, j int) bool {
			return res.mounts[i].prefix.String() > res.mounts[j].prefix.String()
		})

	return &res, nil
}

func (c *mountDatastoreConfig) DiskSpec() DiskSpec {
	cfg := map[string]interface{}{"type": "mount"}
	mounts := make([]interface{}, len(c.mounts))
	for i, m := range c.mounts {
		c := m.ds.DiskSpec()
		if c == nil {
			c = make(map[string]interface{})
		}
		c["mountpoint"] = m.prefix.String()
		mounts[i] = c
	}
	cfg["mounts"] = mounts
	return cfg
}

func (c *mountDatastoreConfig) Create(path string) (repo.Datastore, error) {
	mounts := make([]mount.Mount, len(c.mounts))
	for i, m := range c.mounts {
		ds, err := m.ds.Create(path)
		if err != nil {
			return nil, err
		}
		mounts[i].Datastore = ds
		mounts[i].Prefix = m.prefix
	}
	return mount.New(mounts), nil
}

type memDatastoreConfig struct {
	cfg map[string]interface{}
}

// MemDatastoreConfig returns a memory DatastoreConfig from a spec
func MemDatastoreConfig(params map[string]interface{}) (DatastoreConfig, error) {
	return &memDatastoreConfig{params}, nil
}

func (c *memDatastoreConfig) DiskSpec() DiskSpec {
	return nil
}

func (c *memDatastoreConfig) Create(string) (repo.Datastore, error) {
	return dssync.MutexWrap(ds.NewMapDatastore()), nil
}

type logDatastoreConfig struct {
	child DatastoreConfig
	name  string
}

// LogDatastoreConfig returns a log DatastoreConfig from a spec
func LogDatastoreConfig(params map[string]interface{}) (DatastoreConfig, error) {
	childField, ok := params["child"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'child' field is missing or not a map")
	}
	child, err := AnyDatastoreConfig(childField)
	if err != nil {
		return nil, err
	}
	name, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("'name' field was missing or not a string")
	}
	return &logDatastoreConfig{child, name}, nil

}

func (c *logDatastoreConfig) Create(path string) (repo.Datastore, error) {
	child, err := c.child.Create(path)
	if err != nil {
		return nil, err
	}
	return ds.NewLogDatastore(child, c.name), nil
}

func (c *logDatastoreConfig) DiskSpec() DiskSpec {
	return c.child.DiskSpec()
}

type measureDatastoreConfig struct {
	child  DatastoreConfig
	prefix string
}

// MeasureDatastoreConfig returns a measure DatastoreConfig from a spec
func MeasureDatastoreConfig(params map[string]interface{}) (DatastoreConfig, error) {
	childField, ok := params["child"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("'child' field is missing or not a map")
	}
	child, err := AnyDatastoreConfig(childField)
	if err != nil {
		return nil, err
	}
	prefix, ok := params["prefix"].(string)
	if !ok {
		return nil, fmt.Errorf("'prefix' field was missing or not a string")
	}
	return &measureDatastoreConfig{child, prefix}, nil
}

func (c *measureDatastoreConfig) DiskSpec() DiskSpec {
	return c.child.DiskSpec()
}

func (c measureDatastoreConfig) Create(path string) (repo.Datastore, error) {
	child, err := c.child.Create(path)
	if err != nil {
		return nil, err
	}
	return measure.New(c.prefix, child), nil
}

type aferoDatastoreConfig struct {
	path string
}

var _ DatastoreConfig = (*aferoDatastoreConfig)(nil)

var DsFs afero.Fs

// AferoDatastoreConfig returns an afero DatastoreConfig from a spec
func AferoDatastoreConfig(params map[string]interface{}) (DatastoreConfig, error) {
	pp, ok := params["path"]
	if !ok {
		return nil, errors.New("no path")
	}

	p, ok := pp.(string)
	if !ok {
		return nil, errors.New("path is not a string")
	}

	return &aferoDatastoreConfig{
		path: p,
	}, nil
}

func (dsc *aferoDatastoreConfig) Create(path string) (repo.Datastore, error) {
	return &aferoDatastore{fs: DsFs, path: path + "/" + dsc.path}, nil
}

func (dsc *aferoDatastoreConfig) DiskSpec() DiskSpec {
	return map[string]interface{}{
		"type": "afero",
		"path": dsc.path,
	}
}

// Afero version of https://github.com/ipfs/go-datastore/blob/master/examples/fs.go

type aferoDatastore struct {
	fs     afero.Fs
	path   string
	closed bool
}

var _ repo.Datastore = (*aferoDatastore)(nil)

func (ads *aferoDatastore) Batch() (ds.Batch, error) {
	return ds.NewBasicBatch(ads), nil
}

func (ads *aferoDatastore) Close() error {
	ads.closed = true
	return nil
}

func (ads *aferoDatastore) Delete(key ds.Key) (err error) {
	fn := ads.KeyFilename(key)
	if !isFile(ads.fs, fn) {
		return nil
	}

	err = ads.fs.Remove(fn)
	if os.IsNotExist(err) {
		err = nil // idempotent
	}
	return err
}

var ErrClosed = errors.New("datastore is closed")

func (ads *aferoDatastore) Get(key ds.Key) ([]byte, error) {
	return ads.get(key)
}

func (ads *aferoDatastore) get(key ds.Key) ([]byte, error) {
	if ads.closed {
		return nil, ErrClosed
	}

	fn := ads.KeyFilename(key)
	if !isFile(ads.fs, fn) {
		return nil, ds.ErrNotFound
	}

	return afero.ReadFile(ads.fs, fn)
}

// isFile returns whether given path is a file
func isFile(fs afero.Fs, path string) bool {
	finfo, err := fs.Stat(path)
	if err != nil {
		return false
	}

	return !finfo.IsDir()
}

func (ads *aferoDatastore) GetSize(key ds.Key) (int, error) {
	return ds.GetBackedSize(ads, key)
}

func (ads *aferoDatastore) Has(key ds.Key) (bool, error) {
	return ds.GetBackedHas(ads, key)
}

func (ads *aferoDatastore) Put(key ds.Key, value []byte) (err error) {
	fn := ads.KeyFilename(key)

	// mkdirall above.
	err = ads.fs.MkdirAll(filepath.Dir(fn), 0755)
	if err != nil {
		return err
	}

	return afero.WriteFile(ads.fs, fn, value, 0666)
}

var ObjectKeySuffix = ".dsobject"

func (ads *aferoDatastore) Query(q dsq.Query) (dsq.Results, error) {

	entries := []dsq.Entry{}

	walkFn := func(path string, info os.FileInfo, _ error) error {
		// remove ds path prefix
		relPath, err := filepath.Rel(ads.path, path)
		if err == nil {
			path = filepath.ToSlash(relPath)
		}

		if info != nil && !info.IsDir() {
			path = strings.TrimSuffix(path, ObjectKeySuffix)
			var result dsq.Entry
			key := ds.NewKey(path)
			result.Key = key.String()
			if !q.KeysOnly {
				result.Value, err = ads.get(key)
				if err != nil {
					return err
				}
			}
			entries = append(entries, result)
		}
		return nil
	}

	if err := afero.Walk(ads.fs, ads.path, walkFn); err != nil {
		return nil, err
	}

	r := dsq.ResultsWithEntries(q, entries)
	r = dsq.NaiveQueryApply(q, r)
	return r, nil
}

func (ads *aferoDatastore) Sync(ds.Key) error {
	return nil
}

func (ads *aferoDatastore) KeyFilename(key ds.Key) string {
	return filepath.Join(ads.path, key.String()+ObjectKeySuffix)
}
