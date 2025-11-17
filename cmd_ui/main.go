package main

import (
	"context"
	"p2p-call/internal/audio/codec"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/internal/rtc"
	"p2p-call/pkg/interface/desktop"
	"p2p-call/pkg/interface/desktop/views"
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
	// init desktop UI
	app, err := desktop.NewApp()
	if err != nil {
		log.Error().Msgf("Failed to create desktop app: %v", err)
		app.ShowError(err, views.ErrorFatal)
	}
	// run desktop UI in a separate goroutine

	if err := system.EnshureEnvLoaded(); err != nil {
		log.Error().Msgf("Failed to load .env file: %v", err)
		app.ShowError(err, views.ErrorFatal)
	}
	logger.InitLogger()

	ctx := context.Background()

	// create audio codec also can be used opus
	audioCfg := config.NewOpusConfig() // or config.NewOpusConfig()

	// fabric create encoder and decoder based on build tags
	enc, err := codec.CreateEncoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create encoder: %v", err)
		app.ShowError(err, views.ErrorFatal)
	}
	audioCfg.Encoder = enc

	dec, err := codec.CreateDecoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create decoder: %v", err)
		app.ShowError(err, views.ErrorFatal)
	}
	audioCfg.Decoder = dec

	//audioCfg := config.NewOpusConfig() // can be selected any codec here

	// connect to audio pipeline
	pipeline, err := pipeline.NewAudioPipeline(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create audio pipeline: %v", err)
		app.ShowError(err, views.ErrorFatal)
	}
	defer pipeline.Close()
	// initialize main view with audio pipeline capture and playback

	webRtcCon := rtc.NewConnection(pipeline)
	defer closePeerConnection(webRtcCon)

	app.InitMainView(pipeline.Capture, pipeline.Playback, webRtcCon.DcManager.SendMessage)
	// subscribe logger observer on state manager
	webRtcCon.StateManager.Subscribe(logger.NewLoggerObserver())
	// subscribe UI observer on state manager
	webRtcCon.StateManager.Subscribe(app.UiObserver)
	app.MainView.SetConnection(webRtcCon)

	go webRtcCon.Connect(ctx, &audioCfg)

	app.Run()

}
