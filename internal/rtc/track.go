package rtc

import (
	"fmt"
	audiocfg "p2p-call/internal/audio/config"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

// setupAudioTrack creates and adds an audio track to the peer connection
func setupAudioTrack(pc *webrtc.PeerConnection, audioConfig *audiocfg.AudioConfig) (*webrtc.TrackLocalStaticSample, error) {
	var codecCapability webrtc.RTPCodecCapability
	codecCapability = webrtc.RTPCodecCapability{
		MimeType:  audioConfig.MimeType,
		Channels:  uint16(audioConfig.Channels),
		ClockRate: audioConfig.SampleRate, // 8000 для PCMU
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		codecCapability,
		"audio",
		"microphone",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	rtpSender, err := pc.AddTrack(audioTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to add track: %w", err)
	}

	log.Info().
		Str("track_id", rtpSender.Track().ID()).
		Str("codec", string(audioConfig.Type)).
		Uint32("sample_rate", audioConfig.SampleRate).
		Msg("Audio track added")

	return audioTrack, nil
}
