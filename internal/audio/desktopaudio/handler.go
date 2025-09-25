package desktopaudio

import (
	"log"
	"p2p-call/internal/audio/codec"
	"sync"

	"github.com/gordonklaus/portaudio"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"gopkg.in/hraban/opus.v2"
)

const (
	sampleRate      = 48000 // for opus better to use 48000
	framesPerBuffer = 960   // samples 20 ms at 48kHz for opus
	channels        = 1
)

type DesktopAudio struct {
	AudioCaptureChan chan media.Sample
	PlayAudioChan    chan []byte
	Encoder          *codec.OpusEncoder
	Decoder          *codec.OpusDecoder
}

func NewDesktopAudio() (*DesktopAudio, error) {
	opusEncoder, err := codec.NewOpusEncoder(sampleRate, channels, framesPerBuffer, opus.AppVoIP)
	if err != nil {
		return nil, err
	}
	opusDecoder, err := codec.NewOpusDecoder(sampleRate, channels, framesPerBuffer)
	if err != nil {
		return nil, err
	}
	return &DesktopAudio{
		AudioCaptureChan: make(chan media.Sample, 1000), // buffered channel for audio samples
		PlayAudioChan:    make(chan []byte, 1000),       // buffered channel for incoming audio
		Encoder:          opusEncoder,
		Decoder:          opusDecoder,
	}, nil
}

func (ah DesktopAudio) StartAudioCapture(audioTrack *webrtc.TrackLocalStaticSample) {
	log.Println("Starting audio capture...")

	for sample := range ah.AudioCaptureChan {
		//log.Println("data sended to user")
		err := audioTrack.WriteSample(sample)
		if err != nil {
			log.Printf("Error writing audio sample: %v", err)
			return
		}
	}
}

func (ah DesktopAudio) HandleIncomingAudio(track *webrtc.TrackRemote) {
	log.Println("Processing incoming audio stream...")

	trackKind := track.Kind().String()
	trackID := track.ID()
	streamID := track.StreamID()

	log.Printf("Track info: Kind=%s, ID=%s, StreamID=%s", trackKind, trackID, streamID)

	if err := portaudio.Initialize(); err != nil {
		log.Printf("PortAudio init error: %v", err)
		return
	}
	defer portaudio.Terminate()

	defaultOutputDevice, err := portaudio.DefaultOutputDevice()
	if err != nil || defaultOutputDevice == nil {
		log.Println("Error: no default output device")
		return
	}

	outputParams := portaudio.LowLatencyParameters(nil, defaultOutputDevice)
	outputParams.Output.Channels = channels
	outputParams.SampleRate = sampleRate
	outputParams.FramesPerBuffer = framesPerBuffer

	//  (jitter buffer)
	const jitterBufferSize = 10
	playbackBuffer := make([][]float32, 0, jitterBufferSize*2)
	var bufferMutex sync.RWMutex

	currentIndex := 0
	var currentFrame []float32

	stream, err := portaudio.OpenStream(outputParams, func(out []float32) {
		bufferMutex.RLock()
		defer bufferMutex.RUnlock()

		for i := range out {
			if currentFrame == nil || currentIndex >= len(currentFrame) {
				if len(playbackBuffer) > 0 {
					currentFrame = playbackBuffer[0]
					playbackBuffer = playbackBuffer[1:]
					currentIndex = 0
				} else {
					out[i] = 0
					continue
				}
			}

			if currentIndex < len(currentFrame) {
				out[i] = currentFrame[currentIndex]
				currentIndex++
			} else {
				out[i] = 0
			}
		}
	})

	if err != nil {
		log.Printf("Error opening playback stream: %v", err)
		return
	}
	defer stream.Close()

	if err := stream.Start(); err != nil {
		log.Printf("Error starting playback: %v", err)
		return
	}
	defer stream.Stop()

	log.Println("Playback stream started")

	var packetsReceived uint64
	var packetsDecoded uint64

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			log.Printf("Error reading RTP: %v", err)
			return
		}

		packetsReceived++

		if len(rtpPacket.Payload) < 2 {
			log.Printf("Too small RTP payload: %d bytes", len(rtpPacket.Payload))
			continue
		}

		// Opus -> Float32
		float32Samples, err := ah.Decoder.DecodePacket(rtpPacket.Payload)
		if err != nil {
			log.Printf("Decode error: %v", err)
			continue
		}

		packetsDecoded++

		bufferMutex.Lock()
		playbackBuffer = append(playbackBuffer, float32Samples)

		if len(playbackBuffer) > jitterBufferSize*2 {
			log.Printf("Playback buffer overflow, dropping old frames")
			playbackBuffer = playbackBuffer[len(playbackBuffer)-jitterBufferSize:]
		}
		bufferMutex.Unlock()

		if packetsDecoded <= jitterBufferSize {
			log.Printf("Buffering... (%d/%d frames)", packetsDecoded, jitterBufferSize)
		}
	}
}

func (da DesktopAudio) GetAudioCaptureChan() chan media.Sample {
	return da.AudioCaptureChan
}

func (da DesktopAudio) GetPlayAudioChan() chan []byte {
	return da.PlayAudioChan
}

func (da DesktopAudio) GetEncoder() *codec.OpusEncoder {
	return da.Encoder
}

func (da DesktopAudio) GetDecoder() *codec.OpusDecoder {
	return da.Decoder
}
