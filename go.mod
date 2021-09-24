module github.com/berty/go-ipfs-repo-afero

go 1.16

require (
	bazil.org/fuse v0.0.0-20200524192727-fb710f7dfd05 // indirect
	github.com/apex/log v1.9.0
	github.com/dgraph-io/ristretto v0.0.3 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-datastore v0.4.6
	github.com/ipfs/go-ds-measure v0.1.0
	github.com/ipfs/go-filestore v0.0.3
	github.com/ipfs/go-ipfs v0.10.0-rc1
	github.com/ipfs/go-ipfs-config v0.16.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-keystore v0.0.2
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/ipfs/interface-go-ipfs-core v0.5.2
	github.com/kr/text v0.2.0 // indirect
	github.com/libp2p/go-libp2p v0.15.0 // indirect
	github.com/libp2p/go-libp2p-core v0.10.0
	github.com/libp2p/go-libp2p-swarm v0.6.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.4.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.1.2
	github.com/stretchr/testify v1.7.0
	go.uber.org/multierr v1.7.0
	go.uber.org/zap v1.19.0
)

replace (
	bazil.org/fuse => bazil.org/fuse v0.0.0-20200117225306-7b5117fecadc // specific version for iOS building
	github.com/agl/ed25519 => github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // latest commit before the author shutdown the repo; see https://github.com/golang/go/issues/20504

	github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.9.0
	github.com/libp2p/go-libp2p-rendezvous => github.com/berty/go-libp2p-rendezvous v0.0.0-20210915133138-7b54d608d83a // use berty fork of go-libp2p-rendezvous
	github.com/libp2p/go-libp2p-swarm => github.com/libp2p/go-libp2p-swarm v0.5.3
	github.com/libp2p/go-libp2p-tls => github.com/libp2p/go-libp2p-tls v0.2.0

	github.com/peterbourgon/ff/v3 => github.com/moul/ff/v3 v3.0.1 // temporary, see https://github.com/peterbourgon/ff/pull/67, https://github.com/peterbourgon/ff/issues/68
	golang.org/x/mobile => github.com/aeddi/mobile v0.0.3-silicon // temporary, see https://github.com/golang/mobile/pull/58
)
