package datachannel

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

// DataChannelManager manages a WebRTC data channel for sending and receiving messages.
type DataChannelManager struct {
	dataChannel *webrtc.DataChannel
	mu          sync.RWMutex

	OnMessage     func(msg DataChannelMessage)
	OnStateChange func(state webrtc.DataChannelState)
	messageQueue  chan DataChannelMessage
	sendQueue     chan []byte
}

func NewDataChannelManager() *DataChannelManager {
	return &DataChannelManager{
		messageQueue: make(chan DataChannelMessage, 100),
		sendQueue:    make(chan []byte, 100),
	}
}

// CreateDataChannel creates a new data channel with the given label.
func (dcm *DataChannelManager) CreateDataChannel(pc *webrtc.PeerConnection, label string) error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if dcm.dataChannel != nil {
		return fmt.Errorf("datachannel already created")
	}

	ordered := true
	maxRetransmits := uint16(15)

	dc, err := pc.CreateDataChannel(label, &webrtc.DataChannelInit{
		Ordered:        &ordered,
		MaxRetransmits: &maxRetransmits,
	})
	if err != nil {
		return fmt.Errorf("failed to create data channel: %w", err)
	}

	dcm.dataChannel = dc
	dcm.setupHandlers()

	log.Info().Msg("Data channel created")
	return nil
}

func (dcm *DataChannelManager) SendTypingIndicator() error {
	return dcm.send(DataChannelMessage{
		Type: MessageTypeTyping,
	})
}

// SendMessage sends a text message over the data channel.
func (dcm *DataChannelManager) SendMessage(text string) error {
	return dcm.send(DataChannelMessage{
		Type:    MessageTypeText,
		Payload: text,
	})
}

func (dcm *DataChannelManager) send(msg DataChannelMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}
	select {
	case dcm.sendQueue <- data:
		return nil
	default:
		return fmt.Errorf("send queue is full, cannot send message")
	}
}

func (dcm *DataChannelManager) SetDataChannel(dc *webrtc.DataChannel) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	dcm.dataChannel = dc
	dcm.setupHandlers()

	log.Info().Msg("Data channel set")
}

// processSendQueue handles sending messages from the sendQueue to the data channel
func (dcm *DataChannelManager) processSendQueue() {
	for msg := range dcm.sendQueue {
		dcm.mu.RLock()
		dc := dcm.dataChannel
		dcm.mu.RUnlock()

		if dc == nil || dc.ReadyState() != webrtc.DataChannelStateOpen {
			log.Warn().Msg("Data channel not open, cannot send message")
			continue
		}

		if err := dc.Send(msg); err != nil {
			log.Error().Err(err).Msg("Failed to send message over data channel")
			continue
		}
		log.Debug().Msgf("Sent message over data channel: %d bytes", len(msg))
	}
}

func (dcm *DataChannelManager) Close() error {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()

	if dcm.dataChannel != nil {
		return dcm.dataChannel.Close()
	}
	return nil
}
