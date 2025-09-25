package main

import (
	"log"
	"os"
	"p2p-call/internal/audio/webaudio"
	webrtccon "p2p-call/internal/webrtc_con"
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

	audioHandler, err := webaudio.NewAudioHandler()
	if err != nil {
		log.Fatalf("Failed to create audio handler: %v", err)
	}

	webrtccon := webrtccon.WebRtcConnector{Audio: audioHandler}
	//wh := &connection.WebsocketHandler{Audio: audioHandler}

	//go web.StartWebInterface(wh)
	go peerConnection(webrtccon)
	select {} // block forever
}
