package capture

import (
	"fmt"
	"log"
	"os"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/encoder"
	"runtime"

	"github.com/gen2brain/malgo"
)

type MalgoCapture struct {
	PcmChan chan []byte // пока временно конвертация в этом же пакете
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
	var capturedPCM []int16

	enc, err := encoder.New(audiocfg.SampleRate, uint32(audiocfg.Channels))
	if err != nil {
		log.Fatal(err)
	}

	//bufferSize := int(audiocfg.FramesPerBuffer * audiocfg.Channels)
	//buffer := make([]int16, 0, bufferSize)
	//encodPCMU := encoder.PCMUEncoder{}
	//sizeInBytes := uint32(malgo.SampleSizeInBytes(capCfg.Capture.Format))
	frameSamples := audiocfg.FrameSamples
	onCapture := func(_, input []byte, frameCount uint32) {
		if mc.Paused {
			return
		}
		samples := make([]int16, int(frameCount*capCfg.Capture.Channels))
		for i := 0; i < len(samples); i++ {
			off := i * 2
			samples[i] = int16(input[off]) | int16(input[off+1])<<8
		}
		capturedPCM = append(capturedPCM, samples...)

		for len(capturedPCM) >= frameSamples {
			int16Sample := capturedPCM[:frameSamples]
			capturedPCM = capturedPCM[frameSamples:]

			pkt, err := enc.Encode(int16Sample)
			if err != nil {
				log.Printf("encode err: %v", err)
				continue
			}
			select {
			case mc.PcmChan <- pkt:
			default:
				// drop packets if channel is full
				//log.Println("Capture channel full, dropping packet")
			}
		}

		// copy to buffer
		//buffer = append(buffer, int16Samples...)

		//for len(buffer) >= bufferSize {
		//	frame := buffer[:bufferSize]
		//	buffer = buffer[bufferSize:]

		//select {
		//case mc.PcmChan <- int16Samples:
		//default:
		// drop packets if channel is full
		//log.Println("Capture channel full, dropping packet")
		//}
		//}
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
