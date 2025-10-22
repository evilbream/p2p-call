package config

import "log"

const (
	SampleRateOpus      = 48000 // for opus better to use 48000
	FramesPerBufferOpus = 960   // samples 20 ms at 48kHz for opus
	ChannelsOpus        = 1

	SampleRatePCM      = 8000 // for opus better to use 48000
	FramesPerBufferPCM = 160  // samples 20 ms at 48kHz for opus
	ChannelsPCM        = 1

	JitterBufferSize = 6   // frames to buffer
	EnergyThreshold  = 500 // RMS energy threshold for silence detection
	//FrameBytesOpus      = FramesPerBufferOpus * BytesPerSample * ChannelsOpus
	//FrameBytesPCM       = FramesPerBufferPCM * BytesPerSample * ChannelsPCM
)

type AudioConfig struct {
	SampleRate      uint32
	FramesPerBuffer uint32
	Channels        uint32
	BufferSize      int // channel buffer size in frames
}

func NewOpusConfig() AudioConfig {
	log.Println("Using Opus config (48kHz, high quality)")
	return AudioConfig{
		SampleRate:      SampleRateOpus,
		FramesPerBuffer: FramesPerBufferOpus,
		Channels:        ChannelsOpus,
		BufferSize:      300,
	}
}

func NewPCMUConfig() AudioConfig {
	log.Println("Using PCMU/G.711 config (8kHz, telephone quality)")
	return AudioConfig{
		SampleRate:      SampleRatePCM,
		FramesPerBuffer: FramesPerBufferPCM,
		Channels:        ChannelsPCM,
		BufferSize:      300,
	}
}
