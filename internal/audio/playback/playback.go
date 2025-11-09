package playback

import (
	"fmt"
	"log"
	"p2p-call/internal/audio/codec/iface"
	"p2p-call/internal/audio/config"
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
	dec       iface.Decoder
	playCfg   malgo.DeviceConfig
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
	mp := &MalgoPlayback{
		InChan:    make(chan []byte, audiocfg.BufferSize),
		Paused:    true,
		ctx:       ctx,
		pcmBuffer: make([]int16, 0, audiocfg.SampleRate), // one second buffer
		dec:       audiocfg.Decoder,
	}

	playCfg := malgo.DefaultDeviceConfig(malgo.Playback)
	playCfg.Playback.Format = malgo.FormatS16
	playCfg.Playback.Channels = uint32(audiocfg.Channels)
	playCfg.SampleRate = audiocfg.SampleRate
	mp.playCfg = playCfg
	mp.dec = audiocfg.Decoder

	return mp, nil
}

// StartMalgoPlayback starts the playback device
func (mp *MalgoPlayback) StartMalgoPlayback() error {
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

		samplesNeeded := int(frameCount) * int(mp.playCfg.Playback.Channels)

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

	playDev, err := malgo.InitDevice(mp.ctx.Context, mp.playCfg, malgo.DeviceCallbacks{Data: onPlay})
	if err != nil {
		return fmt.Errorf("failed to open playback device: %w", err)
	}
	mp.device = playDev

	if err := mp.device.Start(); err != nil {
		return fmt.Errorf("failed to start playback device: %w", err)
	}

	log.Println("Playback device started")
	return nil
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
