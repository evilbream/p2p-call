package main

import (
	"context"
	"p2p-call/internal/audio/codec"
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

	// create audio codec also can be used opus
	audioCfg := config.NewPCMUConfig() // or config.NewOpusConfig()

	// fabric create encoder and decoder based on build tags
	enc, err := codec.CreateEncoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create encoder: %v", err)
		system.WaitForUserResponse(true)
	}
	audioCfg.Encoder = enc

	dec, err := codec.CreateDecoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create decoder: %v", err)
		system.WaitForUserResponse(true)
	}
	audioCfg.Decoder = dec

	//audioCfg := config.NewOpusConfig() // can be selected any codec here

	// connect to audio pipeline
	pipeline, err := pipeline.NewAudioPipeline(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create audio pipeline: %v", err)
		system.WaitForUserResponse(true)
	}
	defer pipeline.Close()

	webRtcCon := rtc.NewConnection(pipeline)
	go webRtcCon.LogConnectionErrors(webRtcCon.ConStatusChannel)
	// init peer connection
	if err := webRtcCon.Connect(ctx, &audioCfg); err != nil {
		log.Error().Msgf("Failed to start webrtc connection: %v", err)
		system.WaitForUserResponse(true)
	}

	desktopIface, err := desktop.NewDesktopInterface(pipeline.Capture, pipeline.Playback)
	if err != nil {
		log.Printf("Failed to create desktop interface %v", err)
	}

	desktopIface.StartDesktopInterface()

}
