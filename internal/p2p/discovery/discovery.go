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

func NewDiscover(outStream, inStream base.StreamHandler) (*DiscoverManager, error) {
	baseDiscover, err := base.NewDiscoverWithDefaultCfg(outStream, inStream)
	if err != nil {
		return nil, err
	}
	return &DiscoverManager{*baseDiscover}, nil
}

func (d *DiscoverManager) StartDiscovery(ctx context.Context) error {
	// dont wait in real app, start use dht at once and cancel mDNS when it connects
	mdnsCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	mdnsDiscover := mdns.MDNSDiscovery{Discover: d.baseDicover}
	err := mdnsDiscover.Start(mdnsCtx)
	if err == nil {
		return nil
	}
	cancel()
	log.Warn().Err(err).Msg("mDNS discovery failed, falling back to DHT")
	dhtDiscover := dht.DhtDiscover{Discover: d.baseDicover}
	return dhtDiscover.Start(ctx)

}
