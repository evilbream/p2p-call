package capture

import (
	"fmt"
	"log"
	"os"
	"p2p-call/internal/audio/config"

	"github.com/gen2brain/malgo"
	"github.com/hraban/opus"
)

type MalgoCapture struct {
	RawPcmChan chan []byte // пока временно конвертация в этом же пакете
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	Paused     bool
}

func NewMalgoCapture(rawPcmChan chan []byte) (*MalgoCapture, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(msg string) {
		log.Println("Malgo context message", msg)
	})
	if err != nil {
		fmt.Println("context error:", err)
		os.Exit(1)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	// Opus encoder
	enc, err := opus.NewEncoder(config.SampleRate, config.Channels, opus.AppAudio)
	if err != nil {
		return nil, fmt.Errorf("failed to create Opus encoder: %w", err)
	}

	mc := &MalgoCapture{
		RawPcmChan: rawPcmChan,
		Paused:     true,
		ctx:        ctx,
	}

	capCfg := malgo.DefaultDeviceConfig(malgo.Capture)
	capCfg.Capture.Format = malgo.FormatS16
	capCfg.Capture.Channels = config.Channels
	capCfg.SampleRate = config.SampleRate
	capCfg.Alsa.NoMMap = 1

	bufferSize := config.FramesPerBuffer * config.Channels
	buffer := make([]int16, 0, bufferSize)

	onCapture := func(_, pInputSamples []byte, frameCount uint32) {
		if mc.Paused {
			return
		}
		// конвертировать []byte в []int16
		sampleCount := int(frameCount) * config.Channels
		int16Samples := make([]int16, sampleCount)
		for i := range sampleCount {
			int16Samples[i] = int16(pInputSamples[i*2]) | int16(pInputSamples[i*2+1])<<8
		}

		// копировать в буфер
		buffer = append(buffer, int16Samples...)

		for len(buffer) >= bufferSize {
			frame := buffer[:bufferSize]
			buffer = buffer[bufferSize:]

			// кодировать в opus (временное решение)
			opusData := make([]byte, 4000) // max opus packet size
			n, err := enc.Encode(frame, opusData)
			if err != nil {
				log.Println("Opus encode error:", err)
				continue
			}
			packet := make([]byte, n)
			copy(packet, opusData[:n])

			select {
			case mc.RawPcmChan <- packet:
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
