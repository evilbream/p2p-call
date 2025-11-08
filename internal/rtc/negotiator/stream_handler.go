package negotiator

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

type HandshakeCallBack func()
type OfferCallBack func(msg Message)
type AnswerCallBack func(msg Message)

type StreamHandler struct {
	incomingChan chan Message      // channel for incoming messages via p2plib
	outgoingChan chan Message      // channel for outgoing messages via p2plib
	onHandshake  HandshakeCallBack // function called on handshake complete
	OnOffer      OfferCallBack     // function called on offer received
	OnAnswer     AnswerCallBack    // function called on answer received
	sessionID    string            // webrtc session id
}

func NewStreamHandler(sessionID string, onHandShake HandshakeCallBack) *StreamHandler {
	return &StreamHandler{
		incomingChan: make(chan Message, 10),
		outgoingChan: make(chan Message, 10),
		sessionID:    sessionID,
		onHandshake:  onHandShake,
	}
}

// HandleStream handles a new p2p stream
func (sh *StreamHandler) HandleStream(stream network.Stream) (peer.ID, peer.ID) {
	log.Info().Str("peer", stream.Conn().RemotePeer().String()).Msg("New stream opened")

	peerID := stream.Conn().RemotePeer()
	hostID := stream.Conn().LocalPeer()

	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go sh.handleRead(rw)
	go sh.handleWrite(rw)

	// Send handshake
	handshakeMsg := Message{Type: Handshake, SessionID: sh.sessionID}
	sh.outgoingChan <- handshakeMsg
	log.Debug().Msg("Handshake sent")

	return hostID, peerID
}

// handleRead reads messages from the stream p2plib
func (sh *StreamHandler) handleRead(rw *bufio.ReadWriter) {
	defer log.Debug().Msg("HandleRead exited")

	for {
		var length uint32
		if err := binary.Read(rw, binary.BigEndian, &length); err != nil {
			log.Error().Err(err).Msg("Error reading message length")
			return
		}

		payload := make([]byte, length)
		if _, err := io.ReadFull(rw, payload); err != nil {
			log.Error().Err(err).Msg("Error reading payload")
			return
		}

		log.Debug().Str("data", string(payload)).Msg("Received message")

		var message Message
		if err := json.Unmarshal(bytes.TrimSpace(payload), &message); err != nil {
			log.Error().Err(err).Msg("Error unmarshaling message")
			continue
		}

		sh.routeMessage(message)
	}
}

// handleWrite writes messages to the stream p2plib
func (sh *StreamHandler) handleWrite(rw *bufio.ReadWriter) {
	defer log.Debug().Msg("HandleWrite exited")

	for msg := range sh.outgoingChan {
		data := msg.ToBytes()
		if data == nil {
			continue
		}

		length := uint32(len(data))
		if err := binary.Write(rw, binary.BigEndian, length); err != nil {
			log.Error().Err(err).Msg("Error writing length")
			return
		}

		if _, err := rw.Write(data); err != nil {
			log.Error().Err(err).Msg("Error writing data")
			return
		}

		if err := rw.Flush(); err != nil {
			log.Error().Err(err).Msg("Error flushing")
			return
		}
	}
}

// routeMessage routes incoming messages to appropriate handlers
func (sh *StreamHandler) routeMessage(msg Message) {
	switch msg.Type {
	case Handshake:
		log.Info().Msg("Received handshake")
		ack := Message{Type: Ack, SessionID: sh.sessionID}
		sh.outgoingChan <- ack

	case Ack:
		log.Info().Msg("Received ACK")
		if sh.onHandshake != nil {
			sh.onHandshake()
		}

	case Offer:
		log.Info().Msg("Received offer")
		if sh.OnOffer != nil {
			sh.OnOffer(msg)
		}

	case Answer:
		log.Info().Msg("Received answer")
		if sh.OnAnswer != nil {
			sh.OnAnswer(msg)
		}

	default:
		sh.incomingChan <- msg
	}
}

func (sh *StreamHandler) SendMessage(msg Message) {
	sh.outgoingChan <- msg
}
