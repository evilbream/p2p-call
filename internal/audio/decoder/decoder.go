package decoder

import (
	"log"
	"p2p-call/internal/audio/config"
)

type Decoder interface {
	Decode(encoded []byte) ([]int16, error)
}

func New(sampleRate, channels uint32) (Decoder, error) {
	if sampleRate == config.SampleRatePCM && channels == config.ChannelsPCM {
		return &PCMUDecoder{}, nil
	}
	if sampleRate == config.SampleRateOpus && channels == config.ChannelsOpus {
		return NewOpusDecoder(int(sampleRate), int(channels))
	}
	log.Println("Unrecognised sample rate and channel")
	return nil, nil
}
