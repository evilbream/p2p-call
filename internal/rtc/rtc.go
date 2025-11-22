package rtc

import (
	"context"
	"fmt"
	audiocfg "p2p-call/internal/audio/config"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/internal/rtc/datachannel"
	"p2p-call/pkg/config"
	"p2p-call/pkg/system"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

type Connection struct {
	Pipeline       *pipeline.AudioPipeline
	StateManager   *StateManager
	DcManager      *datachannel.DataChannelManager
	PeerConnection *webrtc.PeerConnection
}

func NewConnection(pipeline *pipeline.AudioPipeline) *Connection {
	return &Connection{
		Pipeline:     pipeline,
		StateManager: NewStateManager(),
		DcManager:    datachannel.NewDataChannelManager(),
	}
}

func createConfig() webrtc.Configuration {
	stunServers := config.GetStunServers()
	//turnServers := config.GetTurnServers()

	config := webrtc.Configuration{
		BundlePolicy:  webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
	}

	// use stun and turn servers from config
	config.ICEServers = stunServers

	config.ICECandidatePoolSize = 15 // reduce ice candidates and use trickle candidate send

	return config
}

// returns connection result error, nil if success
func (con Connection) Connect(ctx context.Context, audioCfg *audiocfg.AudioConfig) error {
	// create nat config
	config := createConfig()

	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetICETimeouts(
		time.Second*90, // Disconnected timeout upped for double NAT
		time.Second*45, // Failed timeout
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
	con.PeerConnection = peerConnection

	// setup text channel
	if err := con.DcManager.CreateDataChannel(con.PeerConnection, "chat"); err != nil {
		return fmt.Errorf("failed to create data channel: %v", err)
	}

	con.PeerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		log.Info().Msgf("New DataChannel %s %d", dc.Label(), dc.ID())
		con.DcManager.SetDataChannel(dc)
	})

	audioTrack, err := setupAudioTrack(con.PeerConnection, audioCfg)
	if err != nil {
		return fmt.Errorf("failed to setup audio track: %v", err)
	}

	sessionID := system.GenerateSessionID()
	fmt.Printf("Session ID: %s\n", sessionID)

	// create event handler
	eventHandler := EventHandlers{
		sm:       con.StateManager,
		pipeline: con.Pipeline,
	}
	eventHandler.setupEventHandlers(con.PeerConnection)
	go con.Pipeline.StartSending(audioTrack)
	signal := NewSignal(sessionID, con.PeerConnection)
	if err := signal.StartWebrtcCon(ctx); err != nil {
		return err
	}
	con.StateManager.UpdateState(StateConnected, "WebRTC connection established", nil)
	return nil
}

func (con *Connection) Close() error {
	if con.PeerConnection != nil {
		return con.PeerConnection.Close()
	}
	return nil
}
