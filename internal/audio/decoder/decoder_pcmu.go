package decoder

// Mu-law decode (G.711 PCMU)
const muBias = 0x84

type PCMUDecoder struct{}

// Decode преобразует μ-law байты → PCM int16
func (d *PCMUDecoder) Decode(mu []byte) ([]int16, error) {
	return DecodeMuLawToPCM16(mu), nil
}

func MuLawToLinear16(mu byte) int16 {
	mu = ^mu
	sign := mu & 0x80
	exponent := (mu >> 4) & 0x07
	mantissa := mu & 0x0F
	segmentEnd := int16(0x84) << exponent
	step := int16(1) << (exponent + 3)
	value := segmentEnd + (int16(mantissa) * step)
	value -= muBias
	if sign != 0 {
		return -value
	}
	return value
}

func DecodeMuLawToPCM16(mu []byte) []int16 {
	out := make([]int16, len(mu))
	for i, b := range mu {
		out[i] = MuLawToLinear16(b)
	}
	return out
}
