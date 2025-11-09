package capture

import (
	"fmt"
	"log"
	"p2p-call/internal/audio/codec/iface"
	"p2p-call/internal/audio/config"
	"runtime"

	"github.com/gen2brain/malgo"
)

type MalgoCapture struct {
	PcmChan      chan []byte // пока временно конвертация в этом же пакете
	ctx          *malgo.AllocatedContext
	device       *malgo.Device
	Paused       bool
	capCfg       malgo.DeviceConfig
	frameSamples int
	enc          iface.Encoder
}

func NewMalgoCapture(audiocfg config.AudioConfig) (*MalgoCapture, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		log.Println("Malgo context message", msg)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init malgo context: %w", err)
	}

	mc := &MalgoCapture{
		PcmChan: make(chan []byte, audiocfg.BufferSize),
		Paused:  true,
		ctx:     ctx,
	}

	capCfg := malgo.DefaultDeviceConfig(malgo.Capture)
	capCfg.Capture.Format = malgo.FormatS16
	capCfg.Capture.Channels = uint32(audiocfg.Channels)
	capCfg.SampleRate = audiocfg.SampleRate

	// alsa specific settings for linux
	if runtime.GOOS == "linux" {
		capCfg.Alsa.NoMMap = 1
	}

	mc.capCfg = capCfg
	mc.frameSamples = audiocfg.FrameSamples
	mc.enc = audiocfg.Encoder

	return mc, nil
}

func (mc *MalgoCapture) StartMalgoCapture() error {

	var capturedPCM []int16

	onCapture := func(_, input []byte, frameCount uint32) {
		if mc.Paused {
			return
		}
		samples := make([]int16, int(frameCount*mc.capCfg.Capture.Channels))
		for i := 0; i < len(samples); i++ {
			off := i * 2
			samples[i] = int16(input[off]) | int16(input[off+1])<<8
		}
		capturedPCM = append(capturedPCM, samples...)

		for len(capturedPCM) >= mc.frameSamples {
			int16Sample := capturedPCM[:mc.frameSamples]
			capturedPCM = capturedPCM[mc.frameSamples:]

			pkt, err := mc.enc.Encode(int16Sample)
			if err != nil {
				log.Printf("encode err: %v", err)
				continue
			}
			select {
			case mc.PcmChan <- pkt:
			default:

			}
		}
	}

	device, err := malgo.InitDevice(mc.ctx.Context, mc.capCfg, malgo.DeviceCallbacks{Data: onCapture})
	if err != nil {
		return fmt.Errorf("failed to open capture device: %w", err)
	}
	mc.device = device

	err = mc.device.Start()
	if err != nil {
		return fmt.Errorf("failed to start capture device: %w", err)
	}

	log.Println("Capture device started")

	return nil

}

func (mc *MalgoCapture) Close() {
	if mc.device != nil {
		mc.device.Uninit()
	}
	if mc.ctx != nil {
		_ = mc.ctx.Uninit()
		mc.ctx.Free()
	}
}
