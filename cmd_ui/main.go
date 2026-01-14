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

type CallManager struct {
	app        *desktop.App
	connection *rtc.Connection
	pipeline   *pipeline.AudioPipeline
	cancel     context.CancelFunc
}

func (cm *CallManager) Close() {
	if cm.cancel != nil {
		cm.cancel()
		cm.cancel = nil
	}
	if cm.connection != nil {
		if err := cm.connection.Close(); err != nil {
			log.Error().Msgf("Failed to close peer connection: %v", err)
		}
		cm.connection = nil
	}
	// think how to close it correctly
	//if cm.pipeline != nil {
	//	cm.pipeline.Close()
	//	cm.pipeline = nil
	//}
}

func main() {
	// init desktop UI
	app, err := desktop.NewApp()
	if err != nil {
		log.Error().Msgf("Failed to create desktop app: %v", err)
		return
	}

	manager := &CallManager{app: app}
	defer manager.Close()

	// Set up rendezvous view with callback
	app.SetRendezvousView(func(rendezvousString string) {
		go manager.initializeConnection(rendezvousString)
	})

	app.Run()
}

func (cm *CallManager) initializeConnection(rendezvousString string) {
	if err := system.EnshureEnvLoaded(); err != nil {
		log.Error().Msgf("Failed to load .env file: %v", err)
		cm.app.ShowError(err, views.ErrorFatal)
		return
	}
	logger.InitLogger()

	ctx, cancel := context.WithCancel(context.Background())
	cm.cancel = cancel

	// Show connecting status
	cm.app.ShowConnectionView()

	// create audio codec also can be used opus
	audioCfg := config.NewOpusConfig() // or config.NewOpusConfig()

	// fabric create encoder and decoder based on build tags
	enc, err := codec.CreateEncoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create encoder: %v", err)
		cm.app.ShowError(err, views.ErrorFatal)
		return
	}
	audioCfg.Encoder = enc

	dec, err := codec.CreateDecoder(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create decoder: %v", err)
		cm.app.ShowError(err, views.ErrorFatal)
		return
	}
	audioCfg.Decoder = dec
	// connect to audio pipeline
	pipeline, err := pipeline.NewAudioPipeline(audioCfg)
	if err != nil {
		log.Error().Msgf("Failed to create audio pipeline: %v", err)
		cm.app.ShowError(err, views.ErrorFatal)
		return
	}
	cm.pipeline = pipeline
	webRtcCon := rtc.NewConnection(pipeline, rendezvousString)
	cm.connection = webRtcCon

	cm.app.InitMainView(pipeline.Capture, pipeline.Playback, webRtcCon.DcManager.SendMessage)
	cm.app.SetDisconnectCallback(func() {
		cm.Close()
		cm.app.ShowRendezvousView()
	})

	// subscribe logger observer on state manager
	webRtcCon.StateManager.Subscribe(logger.NewLoggerObserver())
	// subscribe UI observer on state manager
	webRtcCon.StateManager.Subscribe(cm.app.UiObserver)
	cm.app.MainView.SetConnection(webRtcCon)

	if err := webRtcCon.Connect(ctx, &audioCfg); err != nil {
		log.Error().Msgf("Failed to start webrtc connection: %v", err)
		//cm.app.ShowError(err, views.ErrorFatal) // TODO think how to handle errors on cancel
		return
	}
}
