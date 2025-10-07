package main

import (
	"log"
	"os"
	"p2p-call/internal/audio"
	webrtccon "p2p-call/internal/webrtc_con"
	"p2p-call/pkg/connection"
	"p2p-call/pkg/system"
	"p2p-call/pkg/web"

	"github.com/pion/webrtc/v3/pkg/media"
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

func peerConnection(webrtccon webrtccon.WebRtcConnector) {
	if err := webrtccon.Connect(1); err != nil {
		log.Printf("First attempt failed: %v", err)
		log.Println("Retrying with TURN servers...")
		if err := webrtccon.Connect(2); err != nil {
			log.Fatalf("Second attempt failed: %v", err)
		}
	}
}

func main() {
	loadEnv()

	audioHandler := audio.AudioHandler{
		AudioCaptureChan: make(chan media.Sample, 100), // buffered channel for audio samples
		PlayAudioChan:    make(chan []byte, 100),       // buffered channel for incoming audio
	}

	webrtccon := webrtccon.WebRtcConnector{Audio: &audioHandler}
	wh := &connection.WebsocketHandler{Audio: audioHandler}

	go web.StartWebInterface(wh)
	go peerConnection(webrtccon)
	select {} // block forever
}
