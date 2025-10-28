package main

import (
	"context"
	"os"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/internal/rtc"
	"p2p-call/pkg/interface/desktop"
	"p2p-call/pkg/system"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// loadEnv loads environment variables from a .env file if not already set
func loadEnv() {
	if os.Getenv("LOG_LEVEL") == "" { // means .env not loaded
		err := system.LoadEnv(".env")
		if err != nil {
			log.Fatal().Msgf("Failed to load .env file: %v", err)
		}
	}
}

func readConnectionLog(connErrors chan error) {
	for { // todo  remove it and make better
		err := <-connErrors
		if err != nil {
			log.Fatal().Msgf("WebRTC connection error: %v", err)
		}
	}
}

func main() {
	loadEnv()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	ctx := context.Background()
	audioCfg := config.NewOpusConfig() //  can choose specific config here

	// connect to audio pipeline
	pipeline, err := pipeline.NewAudioPipeline(audioCfg)
	if err != nil {
		log.Fatal().Msgf("Failed to create audio pipeline: %v", err)
		//log.Fatal("Failed to create audio pipeline:", err)
		//log.Printf("Failed to create audio pipeline %v", err)
		select {} // to check error in console
	}
	defer pipeline.Close()

	webRtcCon := rtc.NewConnection(pipeline, "opus")
	// init peer connection
	go webRtcCon.Connect(ctx, 1)
	// wait for connection to be established
	if err := <-webRtcCon.Connectionchannel; err != nil {
		log.Fatal().Msgf("failed to start webrtc connection: %v", err)
	}
	go readConnectionLog(webRtcCon.Connectionchannel)

	desktopIface, err := desktop.NewDesktopInterface(pipeline.Capture, pipeline.Playback)
	if err != nil {
		log.Printf("Failed to create desktop interface %v", err)
	}

	desktopIface.StartDesktopInterface()

}
