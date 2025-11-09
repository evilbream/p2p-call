package encoder

const muBias = 0x84
const muClip = 32635

type PCMUEncoder struct{}

// Encode encodes PCM int16 samples to mu-law bytes.
// error for compatibility with Encoder interface
func (c *PCMUEncoder) Encode(data []int16) ([]byte, error) {
	// Убираем детекцию тишины - кодируем все фреймы
	// Иначе теряются данные и звук прерывается
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
