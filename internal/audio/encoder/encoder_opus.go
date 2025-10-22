package encoder

import (
	"fmt"

	"gopkg.in/hraban/opus.v2"
)

// Допустимые frame sizes для 48kHz (мс): 2.5ms=120, 5ms=240, 10ms=480, 20ms=960, 40ms=1920, 60ms=2880

type OpusEncoder struct {
	enc        *opus.Encoder
	sampleRate int
	channels   int
}

// NewOpusEncoder creates an opus encoder.
// app: opus.AppVoIP / opus.AppAudio / opus.AppRestrictedLowDelay
func NewOpusEncoder(sampleRate, channels int) (*OpusEncoder, error) {

	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		return nil, err
	}

	if err := enc.SetDTX(true); err != nil {
		return nil, fmt.Errorf("failed to enavle DTX: %w", err)
	}

	return &OpusEncoder{
		enc:        enc,
		sampleRate: sampleRate,
		channels:   channels,
	}, nil
}

// EncodeFloat32 splits samples into opus packets.
// samples length must be multiple of frameSize*channels (остаток игнорируется).
func (e *OpusEncoder) Encode(samples []int16) ([]byte, error) {

	opusData := make([]byte, 4000) // max opus packet size
	n, err := e.enc.Encode(samples, opusData)
	if err != nil {
		return nil, err
	}

	if n < 3 {
		// very small packet, likely DTX/no voice
		return nil, nil
	}

	packet := make([]byte, n)
	copy(packet, opusData[:n])

	return packet, nil
}
