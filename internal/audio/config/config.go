package config

import (
	"log"
	"p2p-call/internal/audio/codec/iface"

	"github.com/pion/webrtc/v4"
)

type Encoder interface {
	Encode(pcm []int16) ([]byte, error)
}

type Decoder interface {
	Decode(encoded []byte) ([]int16, error)
}

type AudioConfigType string

func (ac AudioConfigType) String() string {
	return string(ac)
}

const (
	SampleRateOpus   = 48000 // for opus better to use 48000
	FrameSamplesOpus = 960   // samples 20 ms at 48kHz for opus
	ChannelsOpus     = 1

	SampleRatePCM   = 8000 // for opus better to use 48000
	FrameSamplesPCM = 160  // samples 20 ms at 48kHz for opus
	ChannelsPCM     = 1

	JitterBufferSize = 2   // frames to buffer
	EnergyThreshold  = 500 // RMS energy threshold for silence detection

	AudioCodecOpus AudioConfigType = "opus"
	AudioCodecPCMU AudioConfigType = "pcmu"
)

type AudioConfig struct {
	SampleRate   uint32
	FrameSamples int
	Channels     uint16
	BufferSize   int // channel buffer size in frames
	Type         AudioConfigType
	SDPFmtpLine  string
	PayloadType  uint8
	MimeType     string
	Encoder      iface.Encoder
	Decoder      iface.Decoder
}

// NewOpusConfig creates AudioConfig for Opus codec
func NewOpusConfig() AudioConfig {
	log.Println("Using Opus config (48kHz, high quality)")
	return AudioConfig{
		SampleRate:   SampleRateOpus,
		FrameSamples: FrameSamplesOpus,
		Channels:     ChannelsOpus,
		BufferSize:   300,
		Type:         AudioCodecOpus,
		SDPFmtpLine:  "minptime=10;useinbandfec=1;maxaveragebitrate=64000;stereo=0;sprop-stereo=0;cbr=0",
		PayloadType:  111,
		MimeType:     webrtc.MimeTypeOpus,
	}
}

// NewPCMUConfig creates AudioConfig for PCMU/G.711 codec
// probable wont be used in future
func NewPCMUConfig() AudioConfig {
	log.Println("Using PCMU/G.711 config (8kHz, telephone quality)")
	return AudioConfig{
		SampleRate:   SampleRatePCM,
		FrameSamples: FrameSamplesPCM,
		Channels:     ChannelsPCM,
		BufferSize:   300,
		Type:         AudioCodecPCMU,
		SDPFmtpLine:  "",
		PayloadType:  0,
		MimeType:     webrtc.MimeTypePCMU,
	}
}

// SetEncoderDecoder sets the encoder and decoder for the audio config
func (ac *AudioConfig) SetEncoderDecoder(enc iface.Encoder, dec iface.Decoder) {
	ac.Encoder = enc
	ac.Decoder = dec
}
