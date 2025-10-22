package main

import (
	"log"
	"os"
	"p2p-call/internal/audio/config"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/internal/connection/rtc"
	"p2p-call/pkg/interface/desktop"
	"p2p-call/pkg/system"
)

// loadEnv loads environment variables from a .env file if not already set
func loadEnv() {
	if os.Getenv("LOG_LEVEL") == "" { // means .env not loaded
		err := system.LoadEnv(".env")
		if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
		}
	}
}

func readConnectionLog(connErrors chan error) {
	for { // todo  remove it and make better
		err := <-connErrors
		if err != nil {
			log.Println("Connection error:", err)
		}
	}
}

func main() {
	loadEnv()
	audioCfg := config.NewOpusConfig() //  can choose specific config here

	// connect to audio pipeline
	pipeline, err := pipeline.NewAudioPipeline(audioCfg)
	if err != nil {
		//log.Fatal("Failed to create audio pipeline:", err)
		log.Printf("Failed to create audio pipeline %v", err)
		select {} // to check error in console
	}
	defer pipeline.Close()

	webRtcCon := rtc.NewConnection(pipeline, "opus")
	// init peer connection
	go webRtcCon.Connect(1)
	// wait for connection to be established
	if err := <-webRtcCon.Connectionchannel; err != nil {
		log.Fatal("Failed to establish WebRTC connection:", err)
	}
	go readConnectionLog(webRtcCon.Connectionchannel)

	desktopIface, err := desktop.NewDesktopInterface(pipeline.Capture, pipeline.Playback)
	if err != nil {
		log.Printf("Failed to create desktop interface %v", err)
	}

	desktopIface.StartDesktopInterface()

}
