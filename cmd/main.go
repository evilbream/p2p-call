package main

import (
	"log"
	"os"
	"p2p-call/internal/audio/capture"
	"p2p-call/internal/audio/handler"
	"p2p-call/internal/audio/playback"
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

func main() {
	loadEnv()
	audioHandler, err := handler.NewDesktopAudio()
	if err != nil {
		panic(err)
	}

	// init peer connection
	errorChannel := make(chan error, 10)
	webRtcCon := rtc.Connection{Audio: audioHandler, Connectionchannel: errorChannel}
	go webRtcCon.Connect(1)
	// wait for connection to be established
	if err := <-webRtcCon.Connectionchannel; err != nil {
		log.Fatal("Failed to establish WebRTC connection:", err)
	}
	// init play system
	capture, err := capture.NewMalgoCapture(audioHandler.AudioCaptureChan)
	if err != nil {
		panic(err)
	}
	playback, err := playback.NewMalgoPlayback(audioHandler.PlayAudioChan)
	if err != nil {
		panic(err)
	}
	desktopIface, err := desktop.NewDesktopInterface(capture, playback)
	if err != nil {
		panic(err)
	}

	desktopIface.StartDesktopInterface()

}
