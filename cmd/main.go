package main

import (
	"context"
	"p2p-call/internal/audio/codec"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/internal/rtc"
	"p2p-call/pkg/interface/tui"
	"p2p-call/pkg/logger"
	"p2p-call/pkg/system"

	"github.com/rs/zerolog/log"
)

func closePeerConnection(peerConnection *rtc.Connection) {
	if peerConnection != nil {
		if err := peerConnection.Close(); err != nil {
			log.Error().Msgf("Failed to close peer connection: %v", err)
		}
	}
}

func main() {
	if err := system.EnshureEnvLoaded(); err != nil {
		log.Error().Msgf("Failed to load .env file: %v", err)
		tui.WaitForUserResponse(true)
	}
	logger.InitLogger()

	ctx := context.Background()

	// create audio codec also can be used opus
	audioCfg := config.NewOpusConfig() // or config.NewOpusConfig()

	// fabric create encoder and decoder based on build tags
	enc, err := codec.CreateEncoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create encoder: %v", err)
		tui.WaitForUserResponse(true)
	}
	audioCfg.Encoder = enc

	dec, err := codec.CreateDecoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create decoder: %v", err)
		tui.WaitForUserResponse(true)
	}
	audioCfg.Decoder = dec

	//audioCfg := config.NewOpusConfig() // can be selected any codec here

	// connect to audio pipeline
	pipeline, err := pipeline.NewAudioPipeline(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create audio pipeline: %v", err)
		tui.WaitForUserResponse(true)
	}
	defer pipeline.Close()

	webRtcCon := rtc.NewConnection(pipeline, "sss")
	defer closePeerConnection(webRtcCon)
	webRtcCon.StateManager.Subscribe(logger.NewLoggerObserver())

	// init peer connection
	if err := webRtcCon.Connect(ctx, &audioCfg); err != nil {
		log.Error().Msgf("Failed to start webrtc connection: %v", err)
		tui.WaitForUserResponse(true)
	}

	desktopIface, err := tui.NewDesktopInterface(pipeline.Capture, pipeline.Playback)
	if err != nil {
		log.Printf("Failed to create desktop interface %v", err)
	}

	desktopIface.StartDesktopInterface()

}
