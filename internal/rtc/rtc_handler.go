package rtc

import (
	"fmt"
	"p2p-call/internal/audio/pipeline"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type EventHandlers struct {
	sm       *StateManager
	pipeline *pipeline.AudioPipeline
}

// handleIceCandidate processes new ICE candidates
func (h EventHandlers) handleIceCandidate(candidate *webrtc.ICECandidate) {
	if candidate != nil {
		var connType string
		switch candidate.Typ.String() {
		case "host":
			connType = "Direct" // local network or public ip
		case "srflx":
			connType = "STUN" // via stun server
		case "relay":
			connType = "TURN" // via turn server (relay)
		case "prflx":
			connType = "Peer" // addition peer reflexive candidate
		default:
			connType = "Undefined"
		}

		log.Debug().
			Str("type", connType).
			Str("protocol", candidate.Protocol.String()).
			Str("address", candidate.Address).
			Uint16("port", candidate.Port).
			Uint32("priority", candidate.Priority).
			Msg("New ICE candidate gathered")
	}
}

func (h EventHandlers) handleIceConnectionStateChange(state webrtc.ICEConnectionState) {
	log.Info().Str("state", state.String()).Msg("ICE state changed")
	// process other connection result
	switch state {
	case webrtc.ICEConnectionStateChecking:
		h.sm.UpdateState(StateChecking, "ice state checking", nil)
	case webrtc.ICEConnectionStateConnected:
		h.sm.UpdateState(StateConnected, "ice connected", nil)
	case webrtc.ICEConnectionStateCompleted:
		h.sm.UpdateState(StateConnected, "ice completed", nil)
	case webrtc.ICEConnectionStateFailed:
		h.sm.UpdateState(StateFailed, "ice connection failed", fmt.Errorf("ice connection failed"))
	case webrtc.ICEConnectionStateDisconnected:
		h.sm.UpdateState(StateDisconnected, "ice disconnected", fmt.Errorf("ice disconnected"))
	case webrtc.ICEConnectionStateClosed:
		h.sm.UpdateState(StateClosed, "ice connection closed", fmt.Errorf("ice connection closed"))

	}
}

func (h EventHandlers) handleConnectionStateChange(state webrtc.PeerConnectionState) {
	log.Info().Str("state", state.String()).Msg("Peer connection state changed")
	switch state {
	case webrtc.PeerConnectionStateConnected:
		h.sm.UpdateState(StateConnected, "connected", nil)
	case webrtc.PeerConnectionStateFailed:
		h.sm.UpdateState(StateFailed, "peer connection failed", fmt.Errorf("peer connection failed"))
	case webrtc.PeerConnectionStateClosed:
		h.sm.UpdateState(StateClosed, "peer connection closed", fmt.Errorf("peer connection closed"))
	}
}

func (h EventHandlers) handleTrackEvent(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
	log.Info().Str("track_id", track.ID()).Str("type", track.Kind().String()).Msg("Received track")

	if track.Kind() == webrtc.RTPCodecTypeAudio {
		log.Info().Msg("Audio track received from peer")
		go h.pipeline.StartReceiving(track)
	}
}

// setupEventHandlers sets up the necessary event handlers for the peer connection
func (h EventHandlers) setupEventHandlers(pc *webrtc.PeerConnection) {
	pc.OnICECandidate(h.handleIceCandidate)
	pc.OnICEConnectionStateChange(h.handleIceConnectionStateChange)
	pc.OnConnectionStateChange(h.handleConnectionStateChange)
	pc.OnTrack(h.handleTrackEvent)
	// start logging stats
	go logStat(pc)
}

// logStat periodically logs connection statistics
func logStat(pc *webrtc.PeerConnection) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := pc.GetStats()
		for _, stat := range stats {
			if inbound, ok := stat.(webrtc.InboundRTPStreamStats); ok {
				log.Debug().
					Uint32("packets", inbound.PacketsReceived).
					Uint64("bytes", inbound.BytesReceived).
					Int32("lost", inbound.PacketsLost).
					Float64("jitter", inbound.Jitter).
					Msg("Inbound RTP stats")
			}
			if outbound, ok := stat.(webrtc.OutboundRTPStreamStats); ok {
				log.Debug().
					Uint32("packets", outbound.PacketsSent).
					Uint64("bytes", outbound.BytesSent).
					Msg("Outbound RTP stats")
			}
		}
	}
}
