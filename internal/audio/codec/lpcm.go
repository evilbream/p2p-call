package codec

import (
	"encoding/binary"
	"fmt"
	"log"
	"unsafe"

	"github.com/pion/rtp"
)

// all in this package only for test lpcm will not be used in prod

// calculate duration in ms for given LPCM data
// duration = (sample_num * 1000) / sample_rate - calcualte duration for LPCM data
func CalcualteDurationForLPCM(message []byte) int {
	// hardcoded for Float32Array, 44100 Hz, mono, (TODO: get that data from frontend)
	const sampleRate = 44100
	const bytesPerSample = 4 // Float32 = 4 bytes
	sampleCount := len(message) / bytesPerSample
	durationMs := (sampleCount * 1000) / sampleRate
	return durationMs

}

// G.711 μ-law decode table
var muLawDecodeTable = [256]int16{
	-32124, -31100, -30076, -29052, -28028, -27004, -25980, -24956,
	-23932, -22908, -21884, -20860, -19836, -18812, -17788, -16764,
	-15996, -15484, -14972, -14460, -13948, -13436, -12924, -12412,
	-11900, -11388, -10876, -10364, -9852, -9340, -8828, -8316,
	-7932, -7676, -7420, -7164, -6908, -6652, -6396, -6140,
	-5884, -5628, -5372, -5116, -4860, -4604, -4348, -4092,
	-3900, -3772, -3644, -3516, -3388, -3260, -3132, -3004,
	-2876, -2748, -2620, -2492, -2364, -2236, -2108, -1980,
	-1884, -1820, -1756, -1692, -1628, -1564, -1500, -1436,
	-1372, -1308, -1244, -1180, -1116, -1052, -988, -924,
	-876, -844, -812, -780, -748, -716, -684, -652,
	-620, -588, -556, -524, -492, -460, -428, -396,
	-372, -356, -340, -324, -308, -292, -276, -260,
	-244, -228, -212, -196, -180, -164, -148, -132,
	-120, -112, -104, -96, -88, -80, -72, -64,
	-56, -48, -40, -32, -24, -16, -8, 0,
	32124, 31100, 30076, 29052, 28028, 27004, 25980, 24956,
	23932, 22908, 21884, 20860, 19836, 18812, 17788, 16764,
	15996, 15484, 14972, 14460, 13948, 13436, 12924, 12412,
	11900, 11388, 10876, 10364, 9852, 9340, 8828, 8316,
	7932, 7676, 7420, 7164, 6908, 6652, 6396, 6140,
	5884, 5628, 5372, 5116, 4860, 4604, 4348, 4092,
	3900, 3772, 3644, 3516, 3388, 3260, 3132, 3004,
	2876, 2748, 2620, 2492, 2364, 2236, 2108, 1980,
	1884, 1820, 1756, 1692, 1628, 1564, 1500, 1436,
	1372, 1308, 1244, 1180, 1116, 1052, 988, 924,
	876, 844, 812, 780, 748, 716, 684, 652,
	620, 588, 556, 524, 492, 460, 428, 396,
	372, 356, 340, 324, 308, 292, 276, 260,
	244, 228, 212, 196, 180, 164, 148, 132,
	120, 112, 104, 96, 88, 80, 72, 64,
	56, 48, 40, 32, 24, 16, 8, 0,
}

// G.711 A-law decode table
var aLawDecodeTable = [256]int16{
	-5504, -5248, -6016, -5760, -4480, -4224, -4992, -4736,
	-7552, -7296, -8064, -7808, -6528, -6272, -7040, -6784,
	-2752, -2624, -3008, -2880, -2240, -2112, -2496, -2368,
	-3776, -3648, -4032, -3904, -3264, -3136, -3520, -3392,
	-22016, -20992, -24064, -23040, -17920, -16896, -19968, -18944,
	-30208, -29184, -32256, -31232, -26112, -25088, -28160, -27136,
	-11008, -10496, -12032, -11520, -8960, -8448, -9984, -9472,
	-15104, -14592, -16128, -15616, -13056, -12544, -14080, -13568,
	-344, -328, -376, -360, -280, -264, -312, -296,
	-472, -456, -504, -488, -408, -392, -440, -424,
	-88, -72, -120, -104, -24, -8, -56, -40,
	-216, -200, -248, -232, -152, -136, -184, -168,
	-1376, -1312, -1504, -1440, -1120, -1056, -1248, -1184,
	-1888, -1824, -2016, -1952, -1632, -1568, -1760, -1696,
	-688, -656, -752, -720, -560, -528, -624, -592,
	-944, -912, -1008, -976, -816, -784, -880, -848,
	5504, 5248, 6016, 5760, 4480, 4224, 4992, 4736,
	7552, 7296, 8064, 7808, 6528, 6272, 7040, 6784,
	2752, 2624, 3008, 2880, 2240, 2112, 2496, 2368,
	3776, 3648, 4032, 3904, 3264, 3136, 3520, 3392,
	22016, 20992, 24064, 23040, 17920, 16896, 19968, 18944,
	30208, 29184, 32256, 31232, 26112, 25088, 28160, 27136,
	11008, 10496, 12032, 11520, 8960, 8448, 9984, 9472,
	15104, 14592, 16128, 15616, 13056, 12544, 14080, 13568,
	344, 328, 376, 360, 280, 264, 312, 296,
	472, 456, 504, 488, 408, 392, 440, 424,
	88, 72, 120, 104, 24, 8, 56, 40,
	216, 200, 248, 232, 152, 136, 184, 168,
	1376, 1312, 1504, 1440, 1120, 1056, 1248, 1184,
	1888, 1824, 2016, 1952, 1632, 1568, 1760, 1696,
	688, 656, 752, 720, 560, 528, 624, 592,
	944, 912, 1008, 976, 816, 784, 880, 848,
}

// decodePCMU декодирует μ-law в Linear PCM 16-bit
func decodePCMU(data []byte) []byte {
	pcm := make([]byte, len(data)*2) // 16-bit = 2 bytes per sample

	for i, sample := range data {
		decoded := muLawDecodeTable[sample]
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(decoded))
	}

	return pcm
}

// decodePCMA декодирует A-law в Linear PCM 16-bit
func decodePCMA(data []byte) []byte {
	pcm := make([]byte, len(data)*2) // 16-bit = 2 bytes per sample

	for i, sample := range data {
		decoded := aLawDecodeTable[sample]
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(decoded))
	}

	return pcm
}

// DecodeRTPToLPCM декодирует RTP payload в Linear PCM
func DecodeRTPToLPCM(rtpPayload []byte, payloadType uint8) ([]byte, error) {
	switch payloadType {
	case 0: // PCMU (G.711 μ-law)
		return decodePCMU(rtpPayload), nil
	case 8: // PCMA (G.711 A-law)
		return decodePCMA(rtpPayload), nil
	case 11: // L16 (Linear PCM 16-bit)
		return rtpPayload, nil // уже Linear PCM
	default:
		return nil, fmt.Errorf("unsupported payload type: %d", payloadType)
	}
}

func LinearToPCMU(pcm int16) uint8 {
	const BIAS = 0x84
	const CLIP = 32635

	sign := uint8(0)
	if pcm < 0 {
		sign = 0x80
		pcm = -pcm
	}

	if pcm > CLIP {
		pcm = CLIP
	}

	pcm += BIAS

	seg := uint8(0)
	if pcm >= 0x100 {
		seg = 1
		if pcm >= 0x200 {
			seg = 2
			if pcm >= 0x400 {
				seg = 3
				if pcm >= 0x800 {
					seg = 4
					if pcm >= 0x1000 {
						seg = 5
						if pcm >= 0x2000 {
							seg = 6
							if pcm >= 0x4000 {
								seg = 7
							}
						}
					}
				}
			}
		}
	}

	if seg >= 1 {
		pcm >>= (seg - 1) + 3
	} else {
		pcm >>= 3
	}

	return (sign | (seg << 4) | (uint8(pcm) & 0x0F)) ^ 0xFF
}

func ConvertToFloat32PCM(pcmData []byte, bitDepth int) []byte {
	var samples []float32

	switch bitDepth {
	case 16:
		for i := 0; i < len(pcmData); i += 2 {
			sample := int16(binary.LittleEndian.Uint16(pcmData[i : i+2]))
			normalized := float32(sample) / 32768.0
			samples = append(samples, normalized)
		}
	case 8:
		for _, b := range pcmData {
			normalized := (float32(b) - 128.0) / 128.0
			samples = append(samples, normalized)
		}
	}

	// convert to []byte
	result := make([]byte, len(samples)*4)
	for i, sample := range samples {
		binary.LittleEndian.PutUint32(result[i*4:], *(*uint32)(unsafe.Pointer(&sample)))
	}

	return result
}

func ConvertRTPToFloat32(rtpPacket *rtp.Packet) ([]byte, error) {
	log.Printf("Converting RTP packet: PayloadType=%d, Length=%d", rtpPacket.PayloadType, len(rtpPacket.Payload))

	// Decode RTP payload to Linear PCM 16-bit
	pcm16Data, err := DecodeRTPToLPCM(rtpPacket.Payload, rtpPacket.PayloadType)
	if err != nil {
		return nil, err
	}

	float32Data := ConvertToFloat32PCM(pcm16Data, 16)

	if rtpPacket.PayloadType == 0 { // PCMU
		float32Data = upsampleTo44100(float32Data)
	}

	log.Printf("Converted to Float32: %d bytes", len(float32Data))
	return float32Data, nil
}

func upsampleTo44100(inputData []byte) []byte {
	// upsampling 8kHz -> 44.1kHz
	// Коэффициент: 44100/8000 ≈ 5.5125
	const ratio = 5.5125

	inputFloat32 := make([]float32, len(inputData)/4)
	for i := 0; i < len(inputFloat32); i++ {
		bits := uint32(inputData[i*4]) | uint32(inputData[i*4+1])<<8 |
			uint32(inputData[i*4+2])<<16 | uint32(inputData[i*4+3])<<24
		inputFloat32[i] = *(*float32)(unsafe.Pointer(&bits))
	}

	outputLength := int(float64(len(inputFloat32)) * ratio)
	outputFloat32 := make([]float32, outputLength)

	for i := 0; i < outputLength; i++ {
		srcIndex := float64(i) / ratio
		srcIndexInt := int(srcIndex)

		if srcIndexInt >= len(inputFloat32)-1 {
			outputFloat32[i] = inputFloat32[len(inputFloat32)-1]
		} else {
			frac := float32(srcIndex - float64(srcIndexInt))
			outputFloat32[i] = inputFloat32[srcIndexInt]*(1-frac) + inputFloat32[srcIndexInt+1]*frac
		}
	}

	result := make([]byte, len(outputFloat32)*4)
	for i, sample := range outputFloat32 {
		bits := *(*uint32)(unsafe.Pointer(&sample))
		result[i*4] = byte(bits)
		result[i*4+1] = byte(bits >> 8)
		result[i*4+2] = byte(bits >> 16)
		result[i*4+3] = byte(bits >> 24)
	}

	return result
}

func resample(input []float32, fromRate, toRate int) []float32 {
	ratio := float64(fromRate) / float64(toRate)
	outputLength := int(float64(len(input)) / ratio)
	output := make([]float32, outputLength)

	for i := 0; i < outputLength; i++ {
		srcIndex := float64(i) * ratio
		srcIndexInt := int(srcIndex)

		if srcIndexInt >= len(input)-1 {
			output[i] = input[len(input)-1]
		} else {
			frac := float32(srcIndex - float64(srcIndexInt))
			output[i] = input[srcIndexInt]*(1-frac) + input[srcIndexInt+1]*frac
		}
	}

	return output
}

func LinearToPCMUandResampling(message []byte) []byte {
	// convert Float32Array
	float32Data := (*[4096]float32)(unsafe.Pointer(&message[0]))[:len(message)/4]

	// resample 44100 -> 16000 (less loss)
	resampledData := resample(float32Data, 44100, 16000)

	// Конвертируем в μ-law
	pcmuData := make([]byte, len(resampledData))
	for i, sample := range resampledData {
		pcm16 := int16(sample * 32767)
		pcmuData[i] = LinearToPCMU(pcm16)
	}
	return pcmuData
}
