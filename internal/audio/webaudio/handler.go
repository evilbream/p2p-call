package webaudio

import (
	"fmt"
	"log"
	"p2p-call/internal/audio/codec"
	"p2p-call/internal/audio/convert"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"gopkg.in/hraban/opus.v2"
)

// TODO: there some error with web frontend compatibility (sample rate is 44100 not 48000)
const (
	sampleRate      = 48000 // Opus standard
	framesPerBuffer = 960   // 20ms @ 48kHz
	channels        = 1
)

type AudioHandler struct {
	AudioCaptureChan chan media.Sample
	PlayAudioChan    chan []byte
	Encoder          *codec.OpusEncoder
	Decoder          *codec.OpusDecoder
}

func NewAudioHandler() (*AudioHandler, error) {
	opusEncoder, err := codec.NewOpusEncoder(sampleRate, channels, framesPerBuffer, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %w", err)
	}
	opusDecoder, err := codec.NewOpusDecoder(sampleRate, channels, framesPerBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus decoder: %w", err)
	}
	return &AudioHandler{
		AudioCaptureChan: make(chan media.Sample, 2000), // buffered channel for audio samples
		PlayAudioChan:    make(chan []byte, 2000),       // buffered channel for incoming audio
		Encoder:          opusEncoder,
		Decoder:          opusDecoder,
	}, nil
}

func (ah AudioHandler) StartAudioCapture(audioTrack *webrtc.TrackLocalStaticSample) {
	log.Println("Starting audio capture...")

	for sample := range ah.AudioCaptureChan {

		//log.Printf("Sending sample: %d bytes, duration: %v",
		//	len(sample.Data), sample.Duration)

		done := make(chan error, 1)
		go func() {
			done <- audioTrack.WriteSample(sample)
		}()

		select {
		case err := <-done:
			if err != nil {
				log.Printf("Error writing audio sample: %v", err)
			}
		case <-time.After(100 * time.Millisecond):
			log.Println("WriteSample timeout, dropping packet")
		}
	}
}

func (ah AudioHandler) HandleIncomingAudio(track *webrtc.TrackRemote) {
	log.Println("Processing incoming audio stream...")

	trackKind := track.Kind().String()
	trackID := track.ID()
	streamID := track.StreamID()

	log.Printf("Track info: Kind=%s, ID=%s, StreamID=%s", trackKind, trackID, streamID)

	const ( // todo implement jitter buffer
		initialBufferSize = 5  // initial buffer 100ms (5 * 20ms)
		minBufferSize     = 3  // min 60ms
		maxBufferSize     = 15 // max 300ms
	)

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			log.Printf("Error reading RTP: %v", err)
			return
		}
		//log.Println("Received RTP packet, decoding...")
		float32BytesData, err := ah.Decoder.DecodePacket(rtpPacket.Payload)
		if err != nil {
			log.Printf("Error decoding RTP payload: %v", err)
			continue
		}

		select {
		case ah.PlayAudioChan <- convert.Float32ToBytes(float32BytesData):
		default:
			log.Println("PlayAudioChan is full, dropping audio frame")
		}
	}

}

func (da AudioHandler) GetAudioCaptureChan() chan media.Sample {
	return da.AudioCaptureChan
}

func (da AudioHandler) GetPlayAudioChan() chan []byte {
	return da.PlayAudioChan
}

func (da AudioHandler) GetEncoder() *codec.OpusEncoder {
	return da.Encoder
}

func (da AudioHandler) GetDecoder() *codec.OpusDecoder {
	return da.Decoder
}
