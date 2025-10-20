package handler

import (
	"log"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

const (
	sampleRate      = 48000 // for opus better to use 48000
	framesPerBuffer = 960   // samples 20 ms at 48kHz for opus
	channels        = 1
)

type DesktopAudio struct {
	AudioCaptureChan chan []byte
	PlayAudioChan    chan []byte
}

func NewDesktopAudio() (*DesktopAudio, error) {

	return &DesktopAudio{
		AudioCaptureChan: make(chan []byte, 1000), // buffered channel for audio samples
		PlayAudioChan:    make(chan []byte, 1000), // buffered channel for incoming audio
	}, nil
}

func (da *DesktopAudio) StartAudioCapture(audioTrack *webrtc.TrackLocalStaticSample) {
	log.Println("Starting audio capture...")

	for sample := range da.AudioCaptureChan {
		sample := media.Sample{
			Data:     sample,
			Duration: 20 * time.Millisecond,
		}

		err := audioTrack.WriteSample(sample)
		if err != nil {
			log.Printf("Error writing audio sample: %v", err)
			return
		}
		log.Println("Packet sended")
	}
}

func (da DesktopAudio) HandleIncomingAudio(track *webrtc.TrackRemote) {
	log.Println("Processing incoming audio stream...")

	trackKind := track.Kind().String()
	trackID := track.ID()
	streamID := track.StreamID()

	log.Printf("Track info: Kind=%s, ID=%s, StreamID=%s", trackKind, trackID, streamID)

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			log.Printf("Error reading RTP: %v", err)
			return
		}
		log.Println("packet received")
		da.PlayAudioChan <- rtpPacket.Payload
	}
}

// GetAudioCaptureChan returns the channel for captured audio packets. in opus format
func (da DesktopAudio) GetAudioCaptureChan() chan []byte {
	return da.AudioCaptureChan
}

// GetPlayAudioChan returns the channel for incoming audio packets. in opus format
func (da DesktopAudio) GetPlayAudioChan() chan []byte {
	return da.PlayAudioChan
}
