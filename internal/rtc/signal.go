package rtc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"p2p-call/internal/p2p/discovery"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type Signal struct {
	SessionID    string
	incomingChan chan string
	outgoingChan chan string
	established  chan struct{}
	pc           *webrtc.PeerConnection
}

func NewSignal(ctx context.Context, sessionID string, pc *webrtc.PeerConnection) (*Signal, error) {
	// run discovery
	s := Signal{
		SessionID:    sessionID,
		pc:           pc,
		incomingChan: make(chan string),
		outgoingChan: make(chan string),
		established:  make(chan struct{}, 2),
	}

	dscvr, err := discovery.NewDiscover(s.handleOutgoingStream, s.handleIncomingStream)

	if err != nil {
		return nil, fmt.Errorf("failed to create discovery: %v", err)
	}

	// по хорошему тут надо изменить логику чтобы могло подлючиться несколько пиров,  но мы ждем любого первого и выходим
	if err := dscvr.StartDiscovery(ctx); err != nil {
		return nil, fmt.Errorf("failed to start discovery: %v", err)
	}
	i := 0
	for range s.established {
		i++
		if i >= 2 {
			log.Info().Msg("Signaling channels established")
			return &s, nil
		}
	}
	return nil, fmt.Errorf("failed to establish signaling channels")
}

func (s *Signal) HandleRead(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			log.Error().Err(err).Msg("Error reading from stream")
			return
		}
		if str == "" {
			continue
		}

		log.Info().Str("data", str).Msg("Received data")
		if strings.Contains(str, "established") { // first data
			s.established <- struct{}{}
			continue
		}
		s.incomingChan <- str
	}
}

func (s *Signal) HandleWrite(rw *bufio.ReadWriter) {
	for sendData := range s.outgoingChan {

		_, err := fmt.Fprintf(rw, "%s\n", sendData)
		if err != nil {
			log.Error().Err(err).Msg("Error writing to stream")
			panic(err)
		}

		err = rw.Flush()
		if err != nil {
			fmt.Println("Error flushing buffer")
			panic(err)
		}

		if sendData == "established" {
			s.established <- struct{}{}
		}
	}
}

//return &s, nil

func (s *Signal) handleIncomingStream(stream network.Stream) {
	log.Info().Str("peer", stream.Conn().RemotePeer().String()).Msg("New incoming stream opened")
	// Create a buffer stream for non-blocking read and write.
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go s.HandleRead(rw)
	go s.HandleWrite(rw)
}

func (s *Signal) handleOutgoingStream(stream network.Stream) {
	log.Info().Str("peer", stream.Conn().RemotePeer().String()).Msg("New outgoing stream opened")
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	go s.HandleRead(rw)
	go s.HandleWrite(rw)
	s.outgoingChan <- "established"

}

func (s *Signal) createAndSendOffer() {
	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create offer")
		return
	}

	err = s.pc.SetLocalDescription(offer)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to set local description, retrying...")
	}

	s.waitForICEGathering()

	finalOffer := s.pc.LocalDescription()
	signalData := SignalData{
		Type:      "offer",
		SDP:       finalOffer,
		SessionID: s.SessionID,
	}

	offerJSON, err := json.Marshal(signalData)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to marshal offer")
		return
	}

	//send offer via stream
	s.outgoingChan <- string(offerJSON)

	s.receiveAnswer()
}

func (s *Signal) receiveAndProcessOffer() {

	for offer := range s.incomingChan {

		var signalData SignalData
		err := json.Unmarshal([]byte(offer), &signalData)
		if err != nil {
			log.Fatal().Msgf("Failed to parse offer: %v", err)
		}

		err = s.pc.SetRemoteDescription(*signalData.SDP)
		if err != nil {
			log.Fatal().Msgf("Failed to set remote description: %v", err)
		}

		answer, err := s.pc.CreateAnswer(nil)
		if err != nil {
			log.Fatal().Msgf("Failed to create answer: %v", err)
		}

		err = s.pc.SetLocalDescription(answer)
		if err != nil {
			log.Fatal().Msgf("Failed to set local description: %v", err)
		}

		fmt.Println("Fetching ICE candidates...")
		s.waitForICEGathering()

		finalAnswer := s.pc.LocalDescription()
		answerData := SignalData{
			Type:      "answer",
			SDP:       finalAnswer,
			SessionID: s.SessionID,
		}

		answerJSON, err := json.Marshal(answerData)
		if err != nil {
			log.Fatal().Msgf("Failed to marshal answer: %v", err)
		}
		//send answer via stream
		s.outgoingChan <- string(answerJSON)
	}
}

func (s *Signal) receiveAnswer() {
	for answer := range s.incomingChan {

		var signalData SignalData
		err := json.Unmarshal([]byte(answer), &signalData)
		if err != nil {
			log.Fatal().Msgf("Failed to parse answer: %v", err)
		}

		err = s.pc.SetRemoteDescription(*signalData.SDP)
		if err != nil {
			log.Fatal().Msgf("Failed to set remote description: %v", err)
		}
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
