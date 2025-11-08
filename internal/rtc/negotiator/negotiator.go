package negotiator

import (
	"context"
	"fmt"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type Negotiator struct {
	pc         *webrtc.PeerConnection
	offerChan  chan Message
	answerChan chan Message
	stream     *StreamHandler
}

// NewNegotiator creates a new Negotiator instance
func NewNegotiator(pc *webrtc.PeerConnection, stream *StreamHandler) *Negotiator {
	return &Negotiator{
		pc:         pc,
		offerChan:  make(chan Message),
		answerChan: make(chan Message, 1),
		stream:     stream,
	}
}

// SetupCallbacks sets up the callbacks for handling offers and answers
func (n *Negotiator) SetupCallbacks() {
	n.stream.OnOffer = func(msg Message) {
		select {
		case n.offerChan <- msg:
		default:
			log.Warn().Msg("Offer channel full")
		}
	}

	n.stream.OnAnswer = func(msg Message) {
		select {
		case n.answerChan <- msg:
		default:
			log.Warn().Msg("Answer channel full")
		}
	}
}

func (n *Negotiator) CreateOffer(ctx context.Context) error {
	offer, err := n.pc.CreateOffer(nil)
	if err != nil {
		return fmt.Errorf("failed to create offer: %w", err)
	}

	if err = n.pc.SetLocalDescription(offer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	n.waitForICEGathering()

	finalOffer := n.pc.LocalDescription()
	offerMsg := Message{
		Type:      Offer,
		SDP:       finalOffer,
		SessionID: n.stream.sessionID,
	}

	n.stream.SendMessage(offerMsg)
	log.Info().Msg("Offer sent, waiting for answer...")

	// Wait for answer
	select {
	case answer := <-n.answerChan:
		return n.processAnswer(answer)
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for answer")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *Negotiator) AcceptOffer(ctx context.Context) error {
	log.Info().Msg("Waiting for offer...")

	select {
	case offer := <-n.offerChan:
		return n.processOffer(offer)
	case <-time.After(30 * time.Second):
		return fmt.Errorf("timeout waiting for offer")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (n *Negotiator) processOffer(offer Message) error {
	if err := n.pc.SetRemoteDescription(*offer.SDP); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	answer, err := n.pc.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	if err = n.pc.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	log.Info().Msg("Gathering ICE candidates...")
	n.waitForICEGathering()

	finalAnswer := n.pc.LocalDescription()
	answerMsg := Message{
		Type:      Answer,
		SDP:       finalAnswer,
		SessionID: n.stream.sessionID,
	}

	n.stream.SendMessage(answerMsg)
	log.Info().Msg("Answer sent")
	return nil
}

func (n *Negotiator) processAnswer(answer Message) error {
	if err := n.pc.SetRemoteDescription(*answer.SDP); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}
	log.Info().Msg("Answer processed successfully")
	return nil
}

func (n *Negotiator) waitForICEGathering() {
	done := make(chan struct{})

	n.pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		log.Info().Str("state", state.String()).Msg("ICE gathering state")
		if state == webrtc.ICEGatheringStateComplete {
			close(done)
		}
	})

	select {
	case <-done:
		log.Info().Msg("ICE candidates gathered")
	case <-time.After(45 * time.Second):
		log.Warn().Msg("ICE gathering timeout")
	}
}
