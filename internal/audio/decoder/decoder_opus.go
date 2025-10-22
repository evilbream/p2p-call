package decoder

import (
	"p2p-call/internal/audio/config"

	"gopkg.in/hraban/opus.v2"
)

type OpusDecoder struct {
	dec        *opus.Decoder
	sampleRate int
	channels   int
	frameSize  int
}

func NewOpusDecoder(sampleRate, channels int) (*OpusDecoder, error) {
	dec, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, err
	}
	return &OpusDecoder{
		dec:        dec,
		sampleRate: sampleRate,
		channels:   channels,
		frameSize:  config.FramesPerBufferOpus * channels,
	}, nil
}

// DecodePacket decodes one opus packet -> float32 samples (interleaved).
func (d *OpusDecoder) Decode(packet []byte) ([]int16, error) {
	intBuf := make([]int16, d.frameSize*d.channels)
	n, err := d.dec.Decode(packet, intBuf)
	if err != nil {
		return nil, err
	}
	intBuf = intBuf[:n*d.channels]
	return intBuf, nil
}
