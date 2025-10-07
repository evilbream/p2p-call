package audio

import (
	"log"
	"p2p-call/internal/audio/codec"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

type AudioHandler struct {
	AudioCaptureChan chan media.Sample
	PlayAudioChan    chan []byte
}

func (ah AudioHandler) StartAudioCapture(audioTrack *webrtc.TrackLocalStaticSample) {
	log.Println("Starting audio capture...")

	for sample := range ah.AudioCaptureChan {
		log.Println("data sended to user")
		err := audioTrack.WriteSample(sample)
		if err != nil {
			log.Printf("Error writing audio sample: %v", err)
			return
		}
	}
}

func (ah AudioHandler) HandleIncomingAudio(track *webrtc.TrackRemote) {
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
		// RTP payload to Float32 for frontend
		float32Data, err := codec.ConvertRTPToFloat32(rtpPacket)
		if err != nil {
			log.Printf("Error converting RTP to Float32: %v", err)
			continue
		}

		// Send to front via WebSocket
		if len(float32Data) > 0 {
			ah.PlayAudioChan <- float32Data
		}

	}
}
