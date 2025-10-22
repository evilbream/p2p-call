package pipeline

import (
	"errors"
	"fmt"
	"log"
	"p2p-call/internal/audio/capture"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/decoder"
	"p2p-call/internal/audio/encoder"
	"p2p-call/internal/audio/playback"
	"sync"
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
	encoder  encoder.Encoder
	decoder  decoder.Decoder

	QuitSend chan struct{}
	QuitRecv chan struct{}

	//jitterBuffer
	jitterBuffer      [][]int16
	jitterBufferMutex sync.Mutex
	minBufferSize     int
	maxBufferSize     int
}

// создает новый аудио пайплайн с заданными параметрами, еще не стартовавший
func NewAudioPipeline(audiocfg config.AudioConfig) (*AudioPipeline, error) {
	encoder, err := encoder.New(audiocfg.SampleRate, audiocfg.Channels)
	if err != nil {
		return nil, err
	}
	decoder, err := decoder.New(audiocfg.SampleRate, audiocfg.Channels)
	if err != nil {
		return nil, err
	}

	capture, err := capture.NewMalgoCapture(audiocfg)
	if err != nil {
		return nil, err
	}
	playback, err := playback.NewMalgoPlayback(audiocfg)
	if err != nil {
		return nil, err
	}

	ap := &AudioPipeline{
		Capture:       capture,
		Playback:      playback,
		encoder:       encoder,
		decoder:       decoder,
		QuitSend:      make(chan struct{}),
		QuitRecv:      make(chan struct{}),
		jitterBuffer:  make([][]int16, config.JitterBufferSize),
		minBufferSize: config.JitterBufferSize,
		maxBufferSize: config.JitterBufferSize * 3,
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
		case pcm, ok := <-p.Capture.PcmChan:
			if !ok {
				return
			}
			encoded, err := p.Encode(pcm)
			if err != nil {
				log.Println(err)
				if errors.Is(err, ErrEncoderNil) {
					return
				}
				continue
			}
			if err := track.WriteSample(media.Sample{Data: encoded, Duration: duration}); err != nil {
				log.Printf("Error writing audio sample: %v", err)
				return
			}
			//log.Println("Packet sent")
		}
	}

}

func (p *AudioPipeline) Encode(pcm []int16) ([]byte, error) {
	if p.encoder == nil {
		log.Println("Encoder cant be nil")
		return nil, ErrEncoderNil
	}
	encoded, err := p.encoder.Encode(pcm)
	if err != nil {
		return nil, fmt.Errorf("failed to encode pcm: %v", err)
	}
	return encoded, nil
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
	// run jitter buffer manager
	go p.manageJitterBuffer()
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
			decoded, err := p.Decode(rtp.Payload)
			if err != nil {
				log.Println(err)
				if errors.Is(err, ErrDecoderNil) {
					return
				}
				continue
			}

			p.addToJitterBuffer(decoded)
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

// addToJitterBuffer adds a frame to the jitter buffer with overflow protection
func (p *AudioPipeline) addToJitterBuffer(frame []int16) {
	p.jitterBufferMutex.Lock()
	defer p.jitterBufferMutex.Unlock()

	// add one frame to buffer
	p.jitterBuffer = append(p.jitterBuffer, frame)

	if len(p.jitterBuffer) > p.maxBufferSize {
		log.Printf("Jitter buffer overflow: %d frames, dropping old frames", len(p.jitterBuffer))
		// remove old frames
		excess := len(p.jitterBuffer) - p.maxBufferSize
		p.jitterBuffer = p.jitterBuffer[excess:]
	}
}

// sends frames from jitter buffer to playback at regular intervals
func (p *AudioPipeline) manageJitterBuffer() {
	ticker := time.NewTicker(20 * time.Millisecond) // 20ms - standard frame duration
	defer ticker.Stop()

	for {
		select {
		case <-p.QuitRecv:
			return
		case <-ticker.C:
			p.jitterBufferMutex.Lock()

			bufferLen := len(p.jitterBuffer)

			// wait minimal buffer size
			if bufferLen < p.minBufferSize {
				p.jitterBufferMutex.Unlock()
				continue
			}

			// get one frame from buffer
			frame := p.jitterBuffer[0]
			p.jitterBuffer = p.jitterBuffer[1:]

			p.jitterBufferMutex.Unlock()

			// send to playback
			select {
			case p.Playback.InChan <- frame:
			default:
				log.Println("Playback channel full, dropping frame")
			}

		}
	}
}

func (p *AudioPipeline) Close() {
	close(p.QuitSend)
	close(p.QuitRecv)
	p.Capture.Close()
	p.Playback.Close()
}
