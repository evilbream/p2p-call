package playback

import (
	"fmt"
	"log"
	"p2p-call/internal/audio/config"
	"sync"

	"github.com/gen2brain/malgo"
)

type MalgoPlayback struct {
	Paused     bool
	InChan     chan []int16
	device     *malgo.Device
	ctx        *malgo.AllocatedContext
	PauseMutex sync.RWMutex
}

func (mp *MalgoPlayback) Close() {
	if mp.device != nil {
		mp.device.Uninit()

	}
}

func NewMalgoPlayback(audiocfg config.AudioConfig) (*MalgoPlayback, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		log.Println("Malgo context message", msg)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to init malgo context: %w", err)
	}

	mp := &MalgoPlayback{
		InChan: make(chan []int16, audiocfg.BufferSize),
		Paused: true,
		ctx:    ctx,
	}

	playCfg := malgo.DefaultDeviceConfig(malgo.Playback)
	playCfg.Playback.Format = malgo.FormatS16
	playCfg.Playback.Channels = audiocfg.Channels
	playCfg.SampleRate = audiocfg.SampleRate

	onPlay := func(pOutputSamples, pInputSamples []byte, frameCount uint32) {
		mp.PauseMutex.RLock()
		paused := mp.Paused
		mp.PauseMutex.RUnlock()

		if paused {
			for i := range pOutputSamples {
				pOutputSamples[i] = 0
			}
			return
		}

		select {
		case pcmFrame := <-mp.InChan:
			//log.Println("packet received in playback")
			// convert int t bytes
			bytesToWrite := min(len(pcmFrame)*2, len(pOutputSamples))

			for i := 0; i < bytesToWrite/2; i++ {
				sample := pcmFrame[i]
				pOutputSamples[i*2] = byte(sample)        // low byte
				pOutputSamples[i*2+1] = byte(sample >> 8) // high byte
			}

			// Fill remaining buffer with silence if needed
			for i := bytesToWrite; i < len(pOutputSamples); i++ {
				pOutputSamples[i] = 0
			}

		default:
			// If no data available, output silence
			for i := range pOutputSamples {
				pOutputSamples[i] = 0
			}
		}
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
