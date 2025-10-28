package host

import (
	"context"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
)

type Options struct {
	ListenAddresses []multiaddr.Multiaddr
	//PrivKey         p2pcrypto.PrivKey
}

// New creates a new libp2p Host with the given options.
func New(ctx context.Context, opts Options) (host.Host, error) {
	host, err := libp2p.New(libp2p.ListenAddrs(opts.ListenAddresses...))
	if err != nil {
		return nil, err
	}
	return host, nil
}
