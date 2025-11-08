package rtc

import (
	"context"
	"fmt"
	"p2p-call/internal/p2p/discovery"
	"p2p-call/internal/p2p/signaling"
	"p2p-call/internal/rtc/negotiator"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type Signal struct {
	sessionID  string
	handshake  *signaling.HandshakeManager
	stream     *negotiator.StreamHandler
	negotiator *negotiator.Negotiator
	pc         *webrtc.PeerConnection
	hostID     peer.ID
	peerID     peer.ID
}

func NewSignal(sessionID string, pc *webrtc.PeerConnection) *Signal {
	handshake := signaling.NewHandshake()
	stream := negotiator.NewStreamHandler(sessionID, handshake.MarkReady)
	negotiator := negotiator.NewNegotiator(pc, stream)

	return &Signal{
		sessionID:  sessionID,
		pc:         pc,
		stream:     stream,
		negotiator: negotiator,
		handshake:  handshake,
	}
}

func (s *Signal) StartWebrtcCon(ctx context.Context) error {
	// Setup callbacks
	s.negotiator.SetupCallbacks()

	// Discovery
	dscvr, err := discovery.NewDiscover(s.handleStream)
	if err != nil {
		return fmt.Errorf("failed to create discovery: %w", err)
	}

	if err := dscvr.StartDiscovery(ctx, s.handshake.Ready); err != nil {
		return fmt.Errorf("failed to start discovery: %w", err)
	}

	// Wait for handshake
	s.handshake.Wait()
	log.Info().Msg("Handshake completed")

	// Negotiate WebRTC
	return s.negotiate(ctx)
}

func (s *Signal) handleStream(stream network.Stream) {
	s.hostID, s.peerID = s.stream.HandleStream(stream)
}

func (s *Signal) negotiate(ctx context.Context) error {
	if s.hostID < s.peerID {
		log.Info().Msg("Acting as offerer")
		return s.negotiator.CreateOffer(ctx)
	}

	log.Info().Msg("Acting as answerer")
	return s.negotiator.AcceptOffer(ctx)
}
