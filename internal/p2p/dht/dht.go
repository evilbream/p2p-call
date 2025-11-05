package dht

import (
	"context"
	"p2p-call/internal/p2p/base"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

type DhtDiscover struct {
	base.Discover
}

func (d *DhtDiscover) Start(ctx context.Context) error {
	host, err := libp2p.New(libp2p.ListenAddrs([]multiaddr.Multiaddr(d.Cfg.ListenAddresses)...))
	if err != nil {
		return err
	}
	log.Info().
		Str("host", host.ID().String()).
		Any("address", host.Addrs()).
		Msg("Host created.")

	// set function that will be called when a peer initiates a connection and starts a stream with this peer
	host.SetStreamHandler(protocol.ID(d.Cfg.ProtocolId), d.InStream)

	bootstrapPeers := make([]peer.AddrInfo, len(d.Cfg.BootstrapPeers))
	for i, addr := range d.Cfg.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(addr)
		bootstrapPeers[i] = *peerinfo
	}
	kademliaDHT, err := dht.New(ctx, host, dht.BootstrapPeers(bootstrapPeers...))
	if err != nil {
		return err
	}

	log.Debug().Msg("Bootstrapping the DHT...")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return err
	}

	// Wait a bit to let bootstrapping finish (really bootstrap should block until it's ready, but that isn't the case yet.)
	time.Sleep(1 * time.Second)

	log.Debug().Msg("Announcing presence...")
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, d.Cfg.RendezvousString)
	log.Debug().Msg("Successfully announced!")

	log.Debug().Msg("Searching for other peers...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			log.Info().Int("rt_size", kademliaDHT.RoutingTable().Size()).Msg("Waiting for peers to connect...")

			peerChan, err := routingDiscovery.FindPeers(ctx, d.Cfg.RendezvousString)
			if err != nil {
				return err
			}

			for peer := range peerChan {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if d.ProcessOnePeer(ctx, host, peer) {
						return nil
					}
				}
			}
		}
	}

}
