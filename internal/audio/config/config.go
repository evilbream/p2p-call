package config

import "log"

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
	SampleRate  uint32
	FrameSamles int
	Channels    uint32
	BufferSize  int // channel buffer size in frames
	Type        AudioConfigType
}

// NewOpusConfig creates AudioConfig for Opus codec
func NewOpusConfig() AudioConfig {
	log.Println("Using Opus config (48kHz, high quality)")
	return AudioConfig{
		SampleRate:  SampleRateOpus,
		FrameSamles: FrameSamplesOpus,
		Channels:    ChannelsOpus,
		BufferSize:  300,
		Type:        AudioCodecOpus,
	}
}

// NewPCMUConfig creates AudioConfig for PCMU/G.711 codec
// probable wont be used in future
func NewPCMUConfig() AudioConfig {
	log.Println("Using PCMU/G.711 config (8kHz, telephone quality)")
	return AudioConfig{
		SampleRate:  SampleRatePCM,
		FrameSamles: FrameSamplesPCM,
		Channels:    ChannelsPCM,
		BufferSize:  300,
		Type:        AudioCodecPCMU,
	}
}
