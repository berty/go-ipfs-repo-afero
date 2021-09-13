module github.com/berty/go-ipfs-repo-afero

go 1.15

require (
	github.com/apex/log v1.9.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-datastore v0.4.5
	github.com/ipfs/go-ds-measure v0.1.0
	github.com/ipfs/go-filestore v0.0.3
	github.com/ipfs/go-ipfs v0.4.20
	github.com/ipfs/go-ipfs-config v0.14.0
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-log/v2 v2.1.1
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/libp2p/go-libp2p v0.13.0 // indirect
	github.com/libp2p/go-libp2p-core v0.8.5
	github.com/libp2p/go-libp2p-quic-transport v0.10.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/pkg/errors v0.9.1
	github.com/spf13/afero v1.1.2
	github.com/stretchr/testify v1.7.0
	go.uber.org/multierr v1.5.0
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20210511113859-b0526f3d8744 // indirect
)

replace (
	github.com/ipfs/go-ipfs => github.com/Jorropo/go-ipfs v0.4.20-0.20201127133049-9632069f4448 // temporary, see https://github.com/ipfs/go-ipfs/issues/7791
	github.com/libp2p/go-libp2p-rendezvous => github.com/berty/go-libp2p-rendezvous v0.0.0-20201028141428-5b2e7e8ff19a // use berty fork of go-libp2p-rendezvous
	github.com/libp2p/go-libp2p-swarm => github.com/Jorropo/go-libp2p-swarm v0.4.2 // temporary, see https://github.com/libp2p/go-libp2p-swarm/pull/227
)
