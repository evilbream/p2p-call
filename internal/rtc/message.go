package rtc

import (
	"encoding/json"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type SignalMessageType string

const (
	Handshake SignalMessageType = "handshake"
	Ack       SignalMessageType = "ack"
	Offer     SignalMessageType = "offer"
	Answer    SignalMessageType = "answer"
	SimpleMsg SignalMessageType = "simple_msg"
	ErrorMsg  SignalMessageType = "error_msg"
)

type Message struct {
	Type      SignalMessageType          `json:"type"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidate       `json:"candidate,omitempty"`
	SessionID string                     `json:"session_id"`
}

func (msg *Message) ToBytes() []byte {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Error().Err(err).Msg("Error marshaling message")
		return nil
	}
	return data
}
