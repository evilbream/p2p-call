package discovery

import (
	"context"
	"p2p-call/internal/p2p/base"
	"p2p-call/internal/p2p/dht"
	"p2p-call/internal/p2p/mdns"
	"time"

	"github.com/rs/zerolog/log"
)

// Discover
type DiscoverManager struct {
	baseDicover base.Discover
}

func NewDiscover(streamHandler base.StreamHandler) (*DiscoverManager, error) {
	baseDiscover, err := base.NewDiscoverWithDefaultCfg(streamHandler)
	if err != nil {
		return nil, err
	}
	return &DiscoverManager{*baseDiscover}, nil
}

// StartDiscovery starts mDNS and DHT discovery concurrently.
// It returns as soon as one of the methods successfully discovers a peer.
// If both methods fail, it returns an error.
func (d *DiscoverManager) StartDiscovery(ctx context.Context, ready chan struct{}) error {
	peerFound := make(chan struct{})

	mdnsCtx, mdnsCancel := context.WithCancel(ctx)
	dhtCtx, dhtCancel := context.WithCancel(ctx)
	defer dhtCancel()
	defer mdnsCancel()

	go func() {
		log.Info().Msg("Starting mdns discovery...")
		mdnsDiscover := mdns.MDNSDiscovery{Discover: d.baseDicover}
		if err := mdnsDiscover.Start(mdnsCtx); err != nil {
			if err == context.Canceled {
				return
			}
			log.Debug().Err(err).Msg("mDNS discovery error")
			return
		}

		log.Info().Msg("mDNS discovery succeeded")
		peerFound <- struct{}{}
		dhtCancel() // stop dht if running

	}()
	go func() {
		select {
		case <-time.After(3 * time.Second):
			log.Info().Msg("Starting DHT discovery...")
			dhtDiscover := dht.DhtDiscover{Discover: d.baseDicover}
			if err := dhtDiscover.Start(dhtCtx); err != nil {
				if err == context.Canceled {
					return
				}
				log.Debug().Err(err).Msg("DHT discovery error")
				return
			}

			log.Info().Msg("DHT discovery succeeded")
			peerFound <- struct{}{}
			mdnsCancel() // stop mdns if running
			return
		case <-dhtCtx.Done():
			return
		}
	}()

	// wait for results
	for {
		select {
		case <-ready:
			log.Info().Msg("Stream established, stopping discovery")
			return nil
		case <-peerFound:
			log.Info().Msg("Peer discovery succeeded")
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}
