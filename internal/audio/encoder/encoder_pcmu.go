package encoder

import (
	"math"
	"p2p-call/internal/audio/config"
)

const muBias = 0x84
const muClip = 32635

type PCMUEncoder struct{}

// Encode encodes PCM int16 samples to mu-law bytes.
// error for compatibility with Encoder interface
func (c *PCMUEncoder) Encode(data []int16) ([]byte, error) {
	if isSilence(data) {
		// return empty slice for silence frames
		return nil, nil
	}
	return EncodePCM16ToMuLaw(data), nil
}

func Linear16ToMuLaw(sample int16) byte {
	sign := (sample >> 8) & 0x80
	if sign != 0 {
		sample = -sample
	}
	if sample > muClip {
		sample = muClip
	}
	sample += muBias
	exponent := uint8(7)
	mask := int16(0x4000)
	for (sample&mask) == 0 && exponent > 0 {
		mask >>= 1
		exponent--
	}
	mantissa := (sample >> (exponent + 3)) & 0x0F
	return ^(uint8(sign) | (exponent << 4) | uint8(mantissa))
}

func EncodePCM16ToMuLaw(pcm []int16) []byte {
	out := make([]byte, len(pcm))
	for i, s := range pcm {
		out[i] = Linear16ToMuLaw(s)
	}
	return out
}

// Simple silence detection based on RMS energy and zero-crossing rate
func isSilence(frame []int16) bool {
	// 1. RMS энергия
	var sumSquares float64
	for _, sample := range frame {
		sumSquares += float64(sample) * float64(sample)
	}
	rms := math.Sqrt(sumSquares / float64(len(frame)))

	if rms < config.EnergyThreshold {
		return true
	}

	// Zero-Crossing Rate
	var zeroCrossings int
	for i := 1; i < len(frame); i++ {
		if (frame[i-1] >= 0 && frame[i] < 0) || (frame[i-1] < 0 && frame[i] >= 0) {
			zeroCrossings++
		}
	}
	zcr := float64(zeroCrossings) / float64(len(frame))

	return zcr < 0.1
}
