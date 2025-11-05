package main

import (
	"context"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/internal/rtc"
	"p2p-call/pkg/interface/desktop"
	"p2p-call/pkg/logger"
	"p2p-call/pkg/system"

	"github.com/rs/zerolog/log"
)

func main() {
	if err := system.EnshureEnvLoaded(); err != nil {
		log.Error().Msgf("Failed to load .env file: %v", err)
		system.WaitForUserResponse(true)
	}
	logger.InitLogger()

	ctx := context.Background()
	audioCfg := config.NewOpusConfig()

	// connect to audio pipeline
	pipeline, err := pipeline.NewAudioPipeline(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create audio pipeline: %v", err)
		system.WaitForUserResponse(true)
	}
	defer pipeline.Close()

	webRtcCon := rtc.NewConnection(pipeline)
	// init peer connection
	go webRtcCon.Connect(ctx)

	// wait for connection to be established or failed
	if err := <-webRtcCon.ConStatusChannel; err != nil {
		log.Error().Msgf("failed to start webrtc connection: %v", err)
		system.WaitForUserResponse(true)
	}

	go webRtcCon.ReadConnectionLog(webRtcCon.ConStatusChannel)

	desktopIface, err := desktop.NewDesktopInterface(pipeline.Capture, pipeline.Playback)
	if err != nil {
		log.Printf("Failed to create desktop interface %v", err)
	}

	desktopIface.StartDesktopInterface()

}
