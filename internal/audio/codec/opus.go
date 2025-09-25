package codec

import (
	"errors"
	"p2p-call/internal/audio/convert"

	"gopkg.in/hraban/opus.v2"
)

// Допустимые frame sizes для 48kHz (мс): 2.5ms=120, 5ms=240, 10ms=480, 20ms=960, 40ms=1920, 60ms=2880
var ErrInvalidFrameSize = errors.New("invalid opus frame size for given sampleRate")

type OpusEncoder struct {
	enc        *opus.Encoder
	sampleRate int
	channels   int
	frameSize  int // samples per channel per пакет (например 960 для 20ms @48k)
}

// NewOpusEncoder creates an opus encoder.
// app: opus.AppVoIP / opus.AppAudio / opus.AppRestrictedLowDelay
func NewOpusEncoder(sampleRate, channels, frameSize int, app opus.Application) (*OpusEncoder, error) {
	if !convert.IsFrameSizeValid(sampleRate, frameSize) {
		return nil, ErrInvalidFrameSize
	}
	enc, err := opus.NewEncoder(sampleRate, channels, app)
	if err != nil {
		return nil, err
	}
	return &OpusEncoder{
		enc:        enc,
		sampleRate: sampleRate,
		channels:   channels,
		frameSize:  frameSize,
	}, nil
}

// EncodeFloat32 splits samples into opus packets.
// samples length must be multiple of frameSize*channels (остаток игнорируется).
func (e *OpusEncoder) EncodeFloat32(samples []float32) ([][]byte, error) {
	frameStride := e.frameSize * e.channels
	nFrames := len(samples) / frameStride
	result := make([][][]byte, 0, nFrames) // temp slice of slices (will flatten)
	var packets [][]byte

	for f := 0; f < nFrames; f++ {
		start := f * frameStride
		end := start + frameStride
		frame := samples[start:end]

		// float32 -> int16
		intFrame := convert.Float32ToInt16(frame)

		out := make([]byte, 1500) // MTU-safe upper bound for typical opus packet
		n, err := e.enc.Encode(intFrame, out)
		if err != nil {
			return nil, err
		}
		pkt := make([]byte, n)
		copy(pkt, out[:n])
		packets = append(packets, pkt)
		_ = result
	}
	return packets, nil
}

type OpusDecoder struct {
	dec        *opus.Decoder
	sampleRate int
	channels   int
	frameSize  int
}

func NewOpusDecoder(sampleRate, channels, frameSize int) (*OpusDecoder, error) {
	if !convert.IsFrameSizeValid(sampleRate, frameSize) {
		return nil, ErrInvalidFrameSize
	}
	dec, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, err
	}
	return &OpusDecoder{
		dec:        dec,
		sampleRate: sampleRate,
		channels:   channels,
		frameSize:  frameSize,
	}, nil
}

// DecodePacket decodes one opus packet -> float32 samples (interleaved).
func (d *OpusDecoder) DecodePacket(packet []byte) ([]float32, error) {
	intBuf := make([]int16, d.frameSize*d.channels)
	n, err := d.dec.Decode(packet, intBuf)
	if err != nil {
		return nil, err
	}
	intBuf = intBuf[:n*d.channels]
	return convert.Int16ToFloat32(intBuf), nil
}

// DecodePackets: decode sequence of packets.
func (d *OpusDecoder) DecodePackets(packets [][]byte) ([]float32, error) {
	var out []float32
	for _, p := range packets {
		frame, err := d.DecodePacket(p)
		if err != nil {
			return nil, err
		}
		out = append(out, frame...)
	}
	return out, nil
}
