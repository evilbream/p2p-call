package playback

import (
	"fmt"
	"log"
	"p2p-call/internal/audio/config"
	"sync"

	"github.com/gen2brain/malgo"
	"github.com/hraban/opus"
)

type MalgoPlayback struct {
	Paused bool
	inChan chan []byte
	device *malgo.Device
	ctx    *malgo.AllocatedContext
	mutex  sync.Mutex
}

func (mp *MalgoPlayback) Close() {
	if mp.device != nil {
		mp.device.Uninit()

	}
}

func NewMalgoPlayback(inChan chan []byte) (*MalgoPlayback, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		log.Println("Malgo context message", msg)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to init malgo context: %w", err)
	}

	mp := &MalgoPlayback{
		inChan: inChan,
		Paused: true,
		ctx:    ctx,
	}
	// Opus decoder
	dec, err := opus.NewDecoder(config.SampleRate, config.Channels)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus decoder: %w", err)
	}
	// Playback config
	playCfg := malgo.DefaultDeviceConfig(malgo.Playback)
	playCfg.Playback.Format = malgo.FormatS16
	playCfg.Playback.Channels = config.Channels
	playCfg.SampleRate = config.SampleRate

	onPlay := func(pOutputSamples, pInputSamples []byte, frameCount uint32) {
		paused := mp.Paused

		if paused {
			for i := range pOutputSamples {
				pOutputSamples[i] = 0
			}
			return
		}

		select {
		case data := <-mp.inChan:
			log.Println("packet receoved in playback")
			pcmFrame := make([]int16, config.FramesPerBuffer*config.Channels)
			n, err := dec.Decode(data, pcmFrame)
			if err != nil {
				log.Printf("Opus decode error: %v", err)
				// Output silence on error
				for i := range pOutputSamples {
					pOutputSamples[i] = 0
				}
				return
			}
			decodedSamples := pcmFrame[:n*config.Channels]

			// 2 bytes per int16
			bytesToWrite := min(len(decodedSamples)*2, len(pOutputSamples))

			for i := 0; i < bytesToWrite/2; i++ {
				sample := decodedSamples[i]
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
