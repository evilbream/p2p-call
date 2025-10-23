package playback

import (
	"fmt"
	"log"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/decoder"
	"sync"

	"github.com/gen2brain/malgo"
)

type MalgoPlayback struct {
	Paused     bool
	InChan     chan []byte
	device     *malgo.Device
	ctx        *malgo.AllocatedContext
	PauseMutex sync.RWMutex

	pcmBuffer []int16
	bufferMu  sync.Mutex
	dec       decoder.Decoder
}

func (mp *MalgoPlayback) Close() {
	if mp.device != nil {
		mp.device.Uninit()
	}
	if mp.ctx != nil {
		_ = mp.ctx.Uninit()
		mp.ctx.Free()
	}
}

func NewMalgoPlayback(audiocfg config.AudioConfig) (*MalgoPlayback, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		log.Println("Malgo context message:", msg)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to init malgo context: %w", err)
	}

	dec, err := decoder.New(audiocfg.SampleRate, audiocfg.Channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	mp := &MalgoPlayback{
		InChan:    make(chan []byte, audiocfg.BufferSize),
		Paused:    true,
		ctx:       ctx,
		pcmBuffer: make([]int16, 0, audiocfg.SampleRate), // one second buffer at 48kHz
		dec:       dec,
	}

	playCfg := malgo.DefaultDeviceConfig(malgo.Playback)
	playCfg.Playback.Format = malgo.FormatS16
	playCfg.Playback.Channels = audiocfg.Channels
	playCfg.SampleRate = audiocfg.SampleRate

	// decode packets in a separate goroutine
	go mp.decodeWorker()

	onPlay := func(pOutputSamples, _ []byte, frameCount uint32) {
		mp.PauseMutex.RLock()
		paused := mp.Paused
		mp.PauseMutex.RUnlock()

		if paused {
			for i := range pOutputSamples {
				pOutputSamples[i] = 0
			}
			return
		}

		samplesNeeded := int(frameCount) * int(playCfg.Playback.Channels)

		mp.bufferMu.Lock()
		availableSamples := len(mp.pcmBuffer)

		if availableSamples >= samplesNeeded {
			for i := 0; i < samplesNeeded; i++ {
				sample := mp.pcmBuffer[i]
				pOutputSamples[i*2] = byte(sample)
				pOutputSamples[i*2+1] = byte(sample >> 8)
			}
			mp.pcmBuffer = mp.pcmBuffer[samplesNeeded:]
		} else if availableSamples > 0 {
			for i := 0; i < availableSamples; i++ {
				sample := mp.pcmBuffer[i]
				pOutputSamples[i*2] = byte(sample)
				pOutputSamples[i*2+1] = byte(sample >> 8)
			}
			for i := availableSamples * 2; i < len(pOutputSamples); i++ {
				pOutputSamples[i] = 0
			}
			mp.pcmBuffer = mp.pcmBuffer[:0]
		} else {
			for i := range pOutputSamples {
				pOutputSamples[i] = 0
			}
		}
		mp.bufferMu.Unlock()
	}

	playDev, err := malgo.InitDevice(ctx.Context, playCfg, malgo.DeviceCallbacks{Data: onPlay})
	if err != nil {
		return nil, fmt.Errorf("failed to open playback device: %w", err)
	}
	mp.device = playDev

	if err := mp.device.Start(); err != nil {
		return nil, fmt.Errorf("failed to start playback device: %w", err)
	}

	log.Println("Playback device started")
	return mp, nil
}

// decodeWorker decode incoming encoded packets
func (mp *MalgoPlayback) decodeWorker() {
	for encodedPacket := range mp.InChan {
		if encodedPacket == nil {
			continue
		}

		decoded, err := mp.dec.Decode(encodedPacket)
		if err != nil {
			log.Printf("decode err: %v", err)
			continue
		}

		mp.bufferMu.Lock()
		mp.pcmBuffer = append(mp.pcmBuffer, decoded...)
		mp.bufferMu.Unlock()
	}
}
