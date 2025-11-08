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
	Cfg           *DiscoverConfig
	StreamHandler func(stream network.Stream)
}

func NewDiscoverWithDefaultCfg(streamHandler StreamHandler) (*Discover, error) {
	if streamHandler == nil {
		return nil, fmt.Errorf("stream handler cannot be nil")
	}
	cfg := NewDefaultDiscoverConfig()
	return &Discover{Cfg: cfg, StreamHandler: streamHandler}, nil
}

// processOnePeer tries to connect to one peer found via mDNS
func (d *Discover) ProcessOnePeer(ctx context.Context, host host.Host, peer peer.AddrInfo) (shouldExit bool) {

	if peer.ID == host.ID() { // make one channel per peer only by peer.ID >= host.ID(). использовать двойной выход только если обоим пирам обязательно открывать отдельный канал
		return shouldExit
	}

	if peer.ID > host.ID() {
		log.Info().Str("peer", peer.String()).Msg("Peer ID greater than host ID, waiting for incoming connection")
		return shouldExit // wait for the other peer to connect
	}

	if host.Network().Connectedness(peer.ID) == network.Connected {
		log.Info().Str("peer", peer.String()).Msg("Already connected")
		return true // already connected
	}

	if err := host.Connect(ctx, peer); err != nil {
		log.Warn().Str("peer", peer.String()).Err(err).Msg("Connection failed")
		return shouldExit
	}

	stream, err := host.NewStream(ctx, peer.ID, protocol.ID(d.Cfg.ProtocolId))
	if err != nil {
		log.Warn().Str("peer", peer.String()).Err(err).Msg("Connection failed")
		return shouldExit
	} else {
		go d.StreamHandler(stream) // process outgoing stream
	}

	log.Info().Str("peer", peer.String()).Msg("Connected to peer")

	shouldExit = true
	return shouldExit
}
