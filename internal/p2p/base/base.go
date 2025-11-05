package base

import (
	"context"
	"fmt"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

var (
	RendezvousString string                = "p2p-meet-example-000cdfb2-7055-4c36-87a7-94a646eaf57e"
	BootstrapPeers   []multiaddr.Multiaddr = dht.DefaultBootstrapPeers
	ListenAddresses  []multiaddr.Multiaddr = []multiaddr.Multiaddr{}
	ProtocolID       string                = "/p2p-call/connection/1.1.0"
	//allowedProtocols map[protocol.ID]struct{} = map[protocol.ID]struct{}{ релаизовать отказ в подключении без нужной реалиазции протокола
	//	protocol.ID(ProtocolID): {},
	//}
)

type DiscoverInterface interface {
	Start(ctx context.Context) error
}

type DiscoverConfig struct {
	ProtocolId       string
	RendezvousString string
	ListenAddresses  []multiaddr.Multiaddr
	BootstrapPeers   []multiaddr.Multiaddr
	ListenHost       string
	ListenPort       int
}

func NewDefaultDiscoverConfig() *DiscoverConfig {
	return &DiscoverConfig{
		ProtocolId:       ProtocolID,
		RendezvousString: RendezvousString,
		ListenAddresses:  ListenAddresses,
		BootstrapPeers:   dht.DefaultBootstrapPeers,
		ListenHost:       "0.0.0.0",
		ListenPort:       0,
	}
}

type StreamHandler func(stream network.Stream)

// Discover
type Discover struct {
	Cfg       *DiscoverConfig
	OutStream func(stream network.Stream)
	InStream  func(stream network.Stream)
}

func NewDiscoverWithDefaultCfg(outStream, inStream StreamHandler) (*Discover, error) {
	if outStream == nil || inStream == nil {
		return nil, fmt.Errorf("stream handlers cannot be nil")
	}
	cfg := NewDefaultDiscoverConfig()
	return &Discover{Cfg: cfg, OutStream: outStream, InStream: inStream}, nil
}

// processOnePeer tries to connect to one peer found via mDNS
func (d *Discover) ProcessOnePeer(ctx context.Context, host host.Host, peer peer.AddrInfo) (shouldExit bool) {

	if peer.ID == host.ID() { // make one channel per peer only by peer.ID >= host.ID()
		return shouldExit
	}

	log.Debug().Str("peer", peer.String()).Msg("found peer via mDNS")
	if err := host.Connect(ctx, peer); err != nil {
		log.Warn().Str("peer", peer.String()).Err(err).Msg("Connection failed")
		return shouldExit
	}

	stream, err := host.NewStream(ctx, peer.ID, protocol.ID(d.Cfg.ProtocolId))
	if err != nil {
		log.Warn().Str("peer", peer.String()).Err(err).Msg("Connection failed")
		return shouldExit
	} else {
		go d.OutStream(stream) // process outgoing stream
	}

	log.Info().Str("peer", peer.String()).Msg("Connected to peer")

	shouldExit = true
	return shouldExit
}
