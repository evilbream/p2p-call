package pipeline

import (
	"errors"
	"fmt"
	"log"
	"p2p-call/internal/audio/capture"
	"p2p-call/internal/audio/codec/iface"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/playback"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

// AddOnPipe adds a processing function to the pipeline.
// q - quit channel to stop the processing
// f - processing function
// in - input channel
// chanBuffer - buffer size for the output channel
// returns output channel (можно добвалять доп обработку по кодированию напрмиер)
func AddOnPipe[X, Y any](q <-chan struct{}, f func(X) Y, in <-chan X, chanBuffer int) chan Y {
	out := make(chan Y, chanBuffer)
	go func() {
		defer close(out)
		for {
			select {
			case <-q:
				return
			case data, ok := <-in:
				if !ok {
					return
				}
				result := f(data)
				select {
				case out <- result:
				default: // if out channel is full, drop the data
					log.Println("Dropping data in pipeline stage")
				}
			}
		}

	}()
	return out
}

var (
	ErrEncoderNil = errors.New("encoder cannot be nil")
	ErrDecoderNil = errors.New("decoder cannot be nil")
)

type AudioPipeline struct {
	Capture  *capture.MalgoCapture
	Playback *playback.MalgoPlayback
	encoder  iface.Encoder
	decoder  iface.Decoder

	QuitSend chan struct{}
	QuitRecv chan struct{}
}

func NewAudioPipeline(audiocfg config.AudioConfig) (*AudioPipeline, error) {

	// create capture
	capture, err := capture.NewMalgoCapture(audiocfg)
	if err != nil {
		return nil, err
	}

	if err := capture.StartMalgoCapture(); err != nil {
		return nil, err
	}

	// create playback
	playback, err := playback.NewMalgoPlayback(audiocfg)
	if err != nil {
		return nil, err
	}
	if err := playback.StartMalgoPlayback(); err != nil {
		return nil, err
	}

	ap := &AudioPipeline{
		Capture:  capture,
		Playback: playback,
		encoder:  audiocfg.Encoder,
		decoder:  audiocfg.Decoder,
		QuitSend: make(chan struct{}),
		QuitRecv: make(chan struct{}),
	}
	return ap, nil
}

// StartSending starts the audio capture, encoding, and sending process.
// capture -> encode -> send
func (p *AudioPipeline) StartSending(track *webrtc.TrackLocalStaticSample) {
	defer log.Println("Sending pipeline stopped")
	const duration = time.Millisecond * 20

	for {
		select {
		case <-p.QuitSend:
			return
		case encoded, ok := <-p.Capture.PcmChan:
			if !ok {
				return
			}
			if err := track.WriteSample(media.Sample{Data: encoded, Duration: duration}); err != nil {
				log.Printf("Error writing audio sample: %v", err)
				return
			}
			//log.Println("Packet sent")
		}
	}

}

// StartReceiving starts the audio receiving, decoding, and playback process.
// receive -> decode -> playback
func (p *AudioPipeline) StartReceiving(track *webrtc.TrackRemote) {
	log.Println("Processing incoming audio stream...")
	defer log.Println("Receiving pipeline stoppped")
	trackKind := track.Kind().String()
	trackID := track.ID()
	streamID := track.StreamID()

	log.Printf("Track info: Kind=%s, ID=%s, StreamID=%s", trackKind, trackID, streamID)
	for {
		select {
		case <-p.QuitRecv:
			return
		default:
			rtp, _, err := track.ReadRTP()
			if err != nil {
				log.Printf("Error reading RTP: %v", err)
				return
			}
			select {
			case p.Playback.InChan <- rtp.Payload:
			default:
				log.Println("RTP channel full, dropping packet")
			}

		}
	}

}

func (p *AudioPipeline) Decode(data []byte) ([]int16, error) {
	if p.decoder == nil {
		return nil, ErrDecoderNil
	}
	decoded, err := p.decoder.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %v", err)
	}
	return decoded, nil
}

func (p *AudioPipeline) Close() {
	close(p.QuitSend)
	close(p.QuitRecv)
	p.Capture.Close()
	p.Playback.Close()
}
