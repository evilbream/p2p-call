package rtc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"p2p-call/internal/p2p/discovery"
	"p2p-call/internal/p2p/signaling"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type Signal struct {
	SessionId    string
	handshake    *signaling.HandshakeManager
	incomingChan chan Message // duplex channel
	outgoingChan chan Message // duplex channel
	pc           *webrtc.PeerConnection
	answerChan   chan Message
	offerChan    chan Message
	peerID       peer.ID // добавить
	hostID       peer.ID // добавить
}

func NewSignal(sessionID string, pc *webrtc.PeerConnection) *Signal {
	return &Signal{
		SessionId:    sessionID,
		pc:           pc,
		incomingChan: make(chan Message),
		outgoingChan: make(chan Message),
		answerChan:   make(chan Message),
		offerChan:    make(chan Message),
		handshake:    signaling.NewHandshake(),
	}
}

func (s *Signal) HandleRead(rw *bufio.ReadWriter) {
	for {

		// read length prefix
		var length uint32
		if err := binary.Read(rw, binary.BigEndian, &length); err != nil {
			log.Error().Err(err).Msg("Error reading message length from stream")
			return
		}

		payload := make([]byte, length)
		if _, err := io.ReadFull(rw, payload); err != nil {
			log.Error().Err(err).Msg("Error reading message payload from stream")
			return
		}

		log.Printf("<< Raw data: %s", string(payload))
		var message Message
		if err := json.Unmarshal(bytes.TrimSpace(payload), &message); err != nil {
			log.Error().Err(err).Msg("Error unmarshaling message")
			s.incomingChan <- Message{Type: ErrorMsg, SessionID: s.SessionId}
			continue
		}

		switch message.Type {
		case Handshake:
			log.Info().Msg("Received handshake message")
			ackMsg := Message{Type: Ack, SessionID: s.SessionId}
			s.outgoingChan <- ackMsg
		case Ack:
			log.Info().Msg("Received ack message")
			s.handshake.MarkReady()
		case Offer:
			log.Info().Msg("Received offer message")
			s.offerChan <- message
		case Answer:
			log.Info().Msg("Received answer message")
			s.answerChan <- message
		default:
			s.incomingChan <- message

		}
	}
}

func (s *Signal) HandleWrite(rw *bufio.ReadWriter) {
	for sendData := range s.outgoingChan {

		data := sendData.ToBytes()
		if data == nil {
			continue
		}

		length := uint32(len(data))
		if err := binary.Write(rw, binary.BigEndian, length); err != nil {
			log.Fatal().Err(err).Msg("Fatal error writing message length to stream")
		}

		_, err := rw.Write(data)
		if err != nil {
			log.Fatal().Err(err).Msg("Fatal error writing to stream")
		}

		err = rw.Flush()
		if err != nil {
			log.Fatal().Err(err).Msg("Error flushing buffer")
		}

	}
}

func (s *Signal) handleStream(stream network.Stream) {
	log.Info().Str("peer", stream.Conn().RemotePeer().String()).Msg("New stream opened")
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	s.peerID = stream.Conn().RemotePeer()
	s.hostID = stream.Conn().LocalPeer()

	go s.HandleRead(rw)
	go s.HandleWrite(rw)

	handshakeMsg := Message{Type: Handshake, SessionID: s.SessionId}
	s.outgoingChan <- handshakeMsg
	log.Debug().Msg("Send handshake message")

}

func (s *Signal) StartWebrtcCon(ctx context.Context) error {

	dscvr, err := discovery.NewDiscover(s.handleStream)

	if err != nil {
		return fmt.Errorf("failed to create discovery: %v", err)
	}

	// по хорошему тут надо изменить логику чтобы могло подлючиться несколько пиров,  но мы ждем любого первого и выходим
	if err := dscvr.StartDiscovery(ctx, s.handshake.Ready); err != nil {
		return fmt.Errorf("failed to start discovery: %v", err)
	}
	s.handshake.Wait()
	log.Info().Msg("Handshake completed")
	return s.negotiateWebRTC()
}

func (s *Signal) negotiateWebRTC() error {
	// Определяем роль по peer ID (меньший ID = offerer)
	if s.hostID < s.peerID {
		log.Info().Msg("Acting as offerer (my ID is smaller)")
		return s.createAndSendOffer()
	} else {
		log.Info().Msg("Acting as answerer (my ID is larger)")
		return s.receiveAndProcessOffer()
	}
}

func (s *Signal) createAndSendOffer() error {
	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	err = s.pc.SetLocalDescription(offer)
	if err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	s.waitForICEGathering()

	finalOffer := s.pc.LocalDescription()
	signalData := Message{
		Type:      "offer",
		SDP:       finalOffer,
		SessionID: s.SessionId,
	}

	//send offer via stream
	s.outgoingChan <- signalData
	// wait for an answer
	message := <-s.answerChan
	s.receiveAnswer(message)
	return nil
}

func (s *Signal) receiveAndProcessOffer() error {
	offer := <-s.offerChan
	if err := s.pc.SetRemoteDescription(*offer.SDP); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	err = s.pc.SetLocalDescription(answer)
	if err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	fmt.Println("Fetching ICE candidates...")
	s.waitForICEGathering()

	finalAnswer := s.pc.LocalDescription()
	answerData := Message{
		Type:      "answer",
		SDP:       finalAnswer,
		SessionID: s.SessionId,
	}
	//send answer via stream
	s.outgoingChan <- answerData
	return nil
}

func (s *Signal) receiveAnswer(answer Message) {
	err := s.pc.SetRemoteDescription(*answer.SDP)
	if err != nil {
		log.Fatal().Msgf("Failed to set remote description: %v", err)
	}
}

func (s *Signal) waitForICEGathering() {
	done := make(chan struct{})

	s.pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		fmt.Printf("ICE Gathering: %s\n", state.String())
		if state == webrtc.ICEGatheringStateComplete {
			close(done)
		}
	})

	select {
	case <-done:
		fmt.Println("ICE candidates gathered")
	case <-time.After(45 * time.Second):
		fmt.Println("ICE gathering timeout, proceeding with gathered candidates")
	}
}
