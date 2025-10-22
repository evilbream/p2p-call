package encoder

import (
	"log"
	"p2p-call/internal/audio/config"
)

type Encoder interface {
	Encode(pcm []int16) ([]byte, error)
}

// Automatically determine encoder type based on sample rate and channels
func New(sampleRate, channels uint32) (Encoder, error) {
	if sampleRate == config.SampleRatePCM && channels == config.ChannelsPCM {
		return &PCMUEncoder{}, nil
	}

	if sampleRate == config.SampleRateOpus && channels == config.ChannelsOpus {
		return NewOpusEncoder(int(sampleRate), int(channels))
	}
	log.Println("Unrecognised sample rate and channel")
	return nil, nil

}
