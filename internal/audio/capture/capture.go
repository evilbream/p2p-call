package capture

import (
	"fmt"
	"log"
	"os"
	"p2p-call/internal/audio/config"
	"runtime"

	"github.com/gen2brain/malgo"
)

type MalgoCapture struct {
	PcmChan chan []int16 // пока временно конвертация в этом же пакете
	ctx     *malgo.AllocatedContext
	device  *malgo.Device
	Paused  bool
}

func NewMalgoCapture(audiocfg config.AudioConfig) (*MalgoCapture, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		log.Println("Malgo context message", msg)
	})
	if err != nil {
		fmt.Println("context error:", err)
		os.Exit(1)
	}

	mc := &MalgoCapture{
		PcmChan: make(chan []int16, audiocfg.BufferSize),
		Paused:  true,
		ctx:     ctx,
	}

	capCfg := malgo.DefaultDeviceConfig(malgo.Capture)
	capCfg.Capture.Format = malgo.FormatS16
	capCfg.Capture.Channels = audiocfg.Channels
	capCfg.SampleRate = audiocfg.SampleRate

	// alsa specific settings for linux
	if runtime.GOOS == "linux" {
		capCfg.Alsa.NoMMap = 1
	}

	bufferSize := int(audiocfg.FramesPerBuffer * audiocfg.Channels)
	buffer := make([]int16, 0, bufferSize)
	//encodPCMU := encoder.PCMUEncoder{}
	onCapture := func(_, pInputSamples []byte, frameCount uint32) {
		if mc.Paused {
			return
		}
		// convert []byte в []int16
		sampleCount := int(frameCount * audiocfg.Channels)
		int16Samples := make([]int16, sampleCount)
		for i := range sampleCount {
			int16Samples[i] = int16(pInputSamples[i*2]) | int16(pInputSamples[i*2+1])<<8
		}

		// copy to buffer
		buffer = append(buffer, int16Samples...)

		for len(buffer) >= bufferSize {
			frame := buffer[:bufferSize]
			buffer = buffer[bufferSize:]

			select {
			case mc.PcmChan <- frame:
			default:
				// drop packets if channel is full
				//log.Println("Capture channel full, dropping packet")
			}
		}
	}

	device, err := malgo.InitDevice(ctx.Context, capCfg, malgo.DeviceCallbacks{Data: onCapture})
	if err != nil {
		return nil, fmt.Errorf("failed to open capture device: %w", err)
	}
	mc.device = device

	err = mc.device.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start capture device: %w", err)
	}

	log.Println("Capture device started")

	return mc, nil

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
