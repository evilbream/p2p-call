package discovery

import (
	"context"
	"fmt"
	myhost "p2p-call/internal/p2p/myhost"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"

	dht "github.com/libp2p/go-libp2p-kad-dht"
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

type StreamHandler func(stream network.Stream)

// Discover
type Discover struct {
	ProtocolId       string
	RendezvousString string
	ListenAddresses  []multiaddr.Multiaddr
	BootstrapPeers   []multiaddr.Multiaddr
	outgoingHandler  func(stream network.Stream)
	incomingHandler  func(stream network.Stream)
}

func NewDiscover(outStream, inStream StreamHandler) (*Discover, error) {
	if outStream == nil || inStream == nil {
		return nil, fmt.Errorf("stream handlers cannot be nil")
	}
	return &Discover{
		outgoingHandler:  outStream,
		incomingHandler:  inStream,
		ProtocolId:       ProtocolID,
		RendezvousString: RendezvousString,
		ListenAddresses:  ListenAddresses,
		BootstrapPeers:   BootstrapPeers,
	}, nil
}

func (d *Discover) StartDiscovery(ctx context.Context) error {
	host, err := myhost.New(ctx, myhost.Options{ListenAddresses: d.ListenAddresses})
	if err != nil {
		return err
	}
	log.Info().
		Str("host", host.ID().String()).
		Any("address", host.Addrs()).
		Msg("Host created.")

	// set function that will be called when a peer initiates a connection and starts a stream with this peer
	host.SetStreamHandler(protocol.ID(ProtocolID), d.incomingHandler)

	bootstrapPeers := make([]peer.AddrInfo, len(BootstrapPeers))
	for i, addr := range BootstrapPeers {
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
	if err := d.Run(ctx, host, kademliaDHT); err != nil {
		return err
	}
	return nil
}

func (d *Discover) Run(ctx context.Context, host host.Host, kademliaDHT *dht.IpfsDHT) error {
	log.Debug().Msg("Announcing presence...")
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, RendezvousString)
	log.Debug().Msg("Successfully announced!")

	log.Debug().Msg("Searching for other peers...")

	anyConnected := false
	for !anyConnected {
		log.Info().
			Int("rt_size", kademliaDHT.RoutingTable().Size()).
			Msg("Waiting for peers to connect...")

		peerChan, err := routingDiscovery.FindPeers(ctx, RendezvousString)
		if err != nil {
			return err
		}

		for peer := range peerChan {
			if peer.ID == host.ID() {
				continue
			}

			log.Debug().
				Str("peer", peer.String()).
				Msg("Discovered peer: ")

			log.Debug().
				Str("peer", peer.String()).
				Msg("Connecting to: ")

			stream, err := host.NewStream(ctx, peer.ID, protocol.ID(ProtocolID))

			if err != nil {
				log.Warn().Str("peer", peer.String()).Err(err).Msg("Connection failed")
				continue
			} else {
				go d.outgoingHandler(stream) //  обработка исходящего стрима, скорее всего отправка приглаения
			}

			log.Info().
				Str("peer", peer.String()).
				Msg("Connected to peer")

			anyConnected = true // remove this so other peers can be connected there
			return nil
		}
	}
	return nil

}
