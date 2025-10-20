package config

const (
	SampleRate       = 48000 // for opus better to use 48000
	FramesPerBuffer  = 960   // samples 20 ms at 48kHz for opus
	Channels         = 1
	JitterBufferSize = 6 // frames to buffer
	BytesPerSample   = 2 // 16 bit audio
	FrameBytes       = FramesPerBuffer * BytesPerSample * Channels
)
