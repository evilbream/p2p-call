package rtc

import (
	"context"
	"fmt"
	audiocfg "p2p-call/internal/audio/config"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/pkg/config"
	"p2p-call/pkg/system"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type Connection struct {
	Pipeline         *pipeline.AudioPipeline
	ConStatusChannel chan error
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
func (con Connection) LogConnectionErrors(connErrors chan error) {
	for {
		err := <-connErrors
		if err != nil {
			log.Error().Err(err).Msg("WebRTC connection error")
			system.WaitForUserResponse(true)
		}
	}
}

// returns connection result error, nil if success
func (con Connection) Connect(ctx context.Context, audioCfg *audiocfg.AudioConfig) error {
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
			MimeType:    audioCfg.MimeType,
			ClockRate:   audioCfg.SampleRate,
			Channels:    audioCfg.Channels,
			SDPFmtpLine: audioCfg.SDPFmtpLine,
		},
		PayloadType: webrtc.PayloadType(audioCfg.PayloadType),
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
	//defer peerConnection.Close()

	audioTrack, err := setupAudioTrack(peerConnection, audioCfg)
	if err != nil {
		return fmt.Errorf("failed to setup audio track: %v", err)
	}

	sessionID := system.GenerateSessionID()
	fmt.Printf("Session ID: %s\n", sessionID)

	// create event handler
	eventHandler := EventHandlers{
		statusChannel: con.ConStatusChannel,
		pipeline:      con.Pipeline,
	}
	eventHandler.setupEventHandlers(peerConnection)
	go con.Pipeline.StartSending(audioTrack)
	signal := NewSignal(sessionID, peerConnection)
	if err := signal.StartWebrtcCon(ctx); err != nil {
		return err
	}
	return nil
}
