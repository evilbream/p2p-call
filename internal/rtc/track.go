package rtc

import (
	"fmt"
	"log"

	"github.com/pion/webrtc/v4"
)

// setupAudioTrack creates and adds an audio track to the peer connection
func setupAudioTrack(pc *webrtc.PeerConnection) (*webrtc.TrackLocalStaticSample, error) {

	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			Channels:  1,
			ClockRate: 48000,
		},
		"audio",
		"microphone",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %v", err)
	}

	rtpSender, err := pc.AddTrack(audioTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to add track: %v", err)
	}

	log.Printf("Audio track added: %s", rtpSender.Track().ID())
	return audioTrack, nil
}
