package rtc

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/pkg/config"
	"p2p-call/pkg/system"
	"strings"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type Connection struct {
	Pipeline         *pipeline.AudioPipeline
	ConStatusChannel chan error
}

type SignalData struct {
	Type      string                     `json:"type"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidate       `json:"candidate,omitempty"`
	SessionID string                     `json:"session_id"`
}

func NewConnection(pipeline *pipeline.AudioPipeline) *Connection {
	return &Connection{
		Pipeline:         pipeline,
		ConStatusChannel: make(chan error, 1),
	}
}

func createConfig() webrtc.Configuration {
	stunServers := config.GetStunServers()
	turnServers := config.GetTurnServers()

	config := webrtc.Configuration{
		BundlePolicy:  webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
	}

	// use stun and turn servers from config
	config.ICEServers = append(stunServers, turnServers...)
	config.ICECandidatePoolSize = 15 // reduce ice candidates and use trickle candidate send

	return config
}

// reads connection log and process errors
func (con Connection) ReadConnectionLog(connErrors chan error) {
	for {
		err := <-connErrors
		if err != nil {
			log.Error().Err(err).Msg("WebRTC connection error")
			system.WaitForUserResponse(true)
		}
	}
}

// returns connection result error, nil if success
func (con Connection) Connect(ctx context.Context) error {
	// create nat config
	config := createConfig()

	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetICETimeouts(
		time.Second*60, // Disconnected timeout upped for double NAT
		time.Second*30, // Failed timeout
		time.Second*5,  // Keepalive interval
	)

	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeTCP4,
		webrtc.NetworkTypeTCP6,
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
	})

	mediaEngine := &webrtc.MediaEngine{}
	err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   48000,
			Channels:    1,
			SDPFmtpLine: "minptime=10;useinbandfec=1;maxaveragebitrate=64000;stereo=0;sprop-stereo=0;cbr=0",
		},
		PayloadType: 111,
	}, webrtc.RTPCodecTypeAudio)

	if err != nil {
		return fmt.Errorf("failed to register codec: %v", err)
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %v", err)
	}
	defer peerConnection.Close()

	audioTrack, err := setupAudioTrack(peerConnection)
	if err != nil {
		return fmt.Errorf("failed to setup audio track: %v", err)
	}

	sessionID := system.GenerateSessionID()
	fmt.Printf("Session ID: %s\n", sessionID)

	signal, err := NewSignal(ctx, sessionID, peerConnection)
	if err != nil {
		return fmt.Errorf("failed to create signal: %v", err)
	}

	// create event handler
	eventHandler := EventHandlers{
		statusChannel: con.ConStatusChannel,
		pipeline:      con.Pipeline,
	}
	eventHandler.setupEventHandlers(peerConnection)

	fmt.Println("─────────────────────────────")
	fmt.Println(" What would you like to do?")
	fmt.Println("─────────────────────────────")
	fmt.Println(" 1. Send an offer")
	fmt.Println(" 2. Accept an offer")
	fmt.Println(" 3. Exit")
	fmt.Print(" Please enter 1, 2, or 3: ")

	go con.Pipeline.StartSending(audioTrack)

	for {
		reader := bufio.NewReader(os.Stdin)
		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)
		switch choice {
		case "1":
			signal.createAndSendOffer()
			return nil
		case "2":
			signal.receiveAndProcessOffer()
			return nil
		case "3":
			fmt.Println("Exiting...")
			return nil
		default:
			fmt.Printf("Invalid choice: %s, choose 1 or 2\n", choice)
		}
	}

	select {} // keep connection alive
}
