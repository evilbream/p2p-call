package mdns

import (
	"context"
	"crypto/rand"
	"fmt"
	"p2p-call/internal/p2p/base"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

type MDNSDiscovery struct {
	base.Discover
}

func initMDNS(peerhost host.Host, rendezvous string) chan peer.AddrInfo {
	// register with service so that we get notified about peer discovery
	n := &discoveryNotifee{}
	n.PeerChan = make(chan peer.AddrInfo)

	// An hour might be a long long period in practical applications. But this is fine for us
	ser := mdns.NewMdnsService(peerhost, rendezvous, n)
	if err := ser.Start(); err != nil {
		log.Error().Err(err).Msg("Failed to start mDNS service")
	}
	return n.PeerChan
}

func (m *MDNSDiscovery) Start(ctx context.Context) error {
	log.Info().Msg("Start mdns discovery")
	r := rand.Reader
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return err
	}
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", m.Cfg.ListenHost, m.Cfg.ListenPort))
	log.Info().Msgf("Source multiaddr: %s", sourceMultiAddr.String())

	host, err := libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		return err
	}
	host.SetStreamHandler(protocol.ID(m.Cfg.ProtocolId), m.StreamHandler)
	log.Info().Msgf("mDNS Host ID: %s", host.ID().String())

	peerChan := initMDNS(host, m.Cfg.RendezvousString)

	for {
		log.Info().Msg("Waiting for peers to connect...")
		select {
		case <-ctx.Done():
			log.Info().Msg("mDNS discovery stopped")
			return ctx.Err()
		case peer := <-peerChan:
			if m.ProcessOnePeer(ctx, host, peer) {
				return nil
			}
		}
	}
}
