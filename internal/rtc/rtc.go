package rtc

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"p2p-call/internal/audio/pipeline"
	"p2p-call/pkg/config"
	"p2p-call/pkg/system"
	"strings"
	"time"

	"github.com/pion/webrtc/v4"
)

type Connection struct {
	Pipeline          *pipeline.AudioPipeline
	Connectionchannel chan error
	Codec             string
}

type SignalData struct {
	Type      string                     `json:"type"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidate       `json:"candidate,omitempty"`
	SessionID string                     `json:"session_id"`
}

func NewConnection(pipeline *pipeline.AudioPipeline, codec string) *Connection {
	if codec == "" {
		codec = "opus" // default codec
	}
	return &Connection{
		Pipeline:          pipeline,
		Connectionchannel: make(chan error, 1),
		Codec:             codec,
	}
}

// returns connection result error, nil if success
func (con Connection) Connect(ctx context.Context, attempt int) error {
	// create nat config
	log.Println("Trying connection with config attempt ", attempt)
	config := createConfigForNATType(attempt)

	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetICETimeouts(
		time.Second*60, // Disconnected timeout upped for double NAT
		time.Second*30, // Failed timeout
		time.Second*5,  // Keepalive interval
	)
	settingEngine.SetReceiveMTU(1500)

	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
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
		log.Fatalf("Failed to register Opus codec: %v", err)
	}

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create PeerConnection: %v", err)
	}
	defer peerConnection.Close()

	audioTrack, err := setupAudioTrack(peerConnection, con.Codec)
	if err != nil {
		log.Fatalf("Failed to setup audio track: %v", err)
	}

	sessionID := system.GenerateSessionID()
	fmt.Printf("Session ID: %s\n", sessionID)

	signal, err := NewSignal(ctx, sessionID, peerConnection)
	if err != nil {
		return fmt.Errorf("failed to create signal: %v", err)
	}

	con.setupEventHandlers(peerConnection)

	fmt.Println("What to do?:")
	fmt.Println("1. send an offer")
	fmt.Println("2. accept an offer")
	fmt.Print("Enter 1 or 2: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	go con.Pipeline.StartSending(audioTrack)

	switch choice {
	case "1":
		signal.createAndSendOffer()
	case "2":
		signal.receiveAndProcessOffer()
	default:
		log.Fatal("Invalid choice")
	}

	select {} // keep connection alive
}

func createConfigForNATType(attempt int) webrtc.Configuration {
	stunServers := config.GetStunServers()

	turnServers := config.GetTurnServers()

	config := webrtc.Configuration{
		BundlePolicy:  webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
	}

	switch attempt {
	case 1:
		log.Println("CONFIGURED ONLY WITH STUN servers")
		config.ICEServers = stunServers
		config.ICECandidatePoolSize = 15

	default:
		log.Println("CONFIGURED WITH STUN and TURN servers")
		config.ICEServers = append(stunServers, turnServers...)
		config.ICECandidatePoolSize = 25
	}

	return config
}

func (con Connection) setupEventHandlers(pc *webrtc.PeerConnection) error {
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			var connType string
			switch candidate.Typ.String() {
			case "host":
				connType = "Direct" // local network or public ip
			case "srflx":
				connType = "STUN" // via stun server
			case "relay":
				connType = "TURN" // via turn server (relay)
			case "prflx":
				connType = "Peer" // addition peer reflexive candidate
			default:
				connType = "Undefined"
			}

			fmt.Printf("%s ICE: %s %s:%d (priority: %d)\n",
				connType,
				candidate.Protocol.String(),
				candidate.Address,
				candidate.Port,
				candidate.Priority,
			)
		}
	})

	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Printf("ICE state: %s\n", state.String())
		// process other connection result
		switch state {
		case webrtc.ICEConnectionStateConnected:
			log.Println("Ice connection is set!")
		case webrtc.ICEConnectionStateFailed:
			log.Println("Ice connection failed")
			con.Connectionchannel <- fmt.Errorf("ice connection failed")
		case webrtc.ICEConnectionStateDisconnected:
			log.Println("ICE disconnected...")
			con.Connectionchannel <- fmt.Errorf("ice disconnected")
		case webrtc.ICEConnectionStateClosed:
			log.Println("ICE connection closed ")
			con.Connectionchannel <- fmt.Errorf("ice connection closed")

		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {

		switch state {
		case webrtc.PeerConnectionStateConnected:
			log.Println("Peer connection established")
			log.Println("You can start messaging!")
			con.Connectionchannel <- nil // signal successful connection
		case webrtc.PeerConnectionStateFailed:
			log.Println("Peer connection failed")
			con.Connectionchannel <- fmt.Errorf("peer connection failed")

		case webrtc.PeerConnectionStateClosed:
			log.Println("Peer connection closed")
		case webrtc.PeerConnectionStateConnecting:
			log.Println("Peer connection connecting...")
		}
	})

	//pc.OnDataChannel(func(dc *webrtc.DataChannel) {
	//	setupDataChannelHandlers(dc, sessionID)
	//})

	// its stream not
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Received track: %s, type: %s", track.ID(), track.Kind().String())

		if track.Kind() == webrtc.RTPCodecTypeAudio {
			log.Println("Audio track received from peer")
			go con.Pipeline.StartReceiving(track)
		}
	})

	go func() {
		ticker := time.NewTicker(500 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			stats := pc.GetStats()
			for _, stat := range stats {
				if inbound, ok := stat.(webrtc.InboundRTPStreamStats); ok {
					log.Printf("Inbound RTP: packets=%d, bytes=%d, lost=%d, jitter=%.3f",
						inbound.PacketsReceived,
						inbound.BytesReceived,
						inbound.PacketsLost,
						inbound.Jitter)
				}
				if outbound, ok := stat.(webrtc.OutboundRTPStreamStats); ok {
					log.Printf("Outbound RTP: packets=%d, bytes=%d",
						outbound.PacketsSent,
						outbound.BytesSent)
				}
			}
		}
	}()

	return nil
}

func setupAudioTrack(pc *webrtc.PeerConnection, codecType string) (*webrtc.TrackLocalStaticSample, error) {
	var capability webrtc.RTPCodecCapability
	switch codecType {
	case "opus":
		capability = webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			Channels:  1,
			ClockRate: 48000,
		}
	case "pcmu":
		capability = webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypePCMU,
			Channels:  1,
			ClockRate: 8000,
		}
	default:
		return nil, fmt.Errorf("unsupported codec: %s", codecType)
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		capability,
		"audio",
		"microphone",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create audio track: %v", err)
	}

	rtpSender, err := pc.AddTrack(audioTrack)
	if err != nil {
		return nil, fmt.Errorf("failed to add track: %v", err)
	}

	log.Printf("Audio track added: %s", rtpSender.Track().ID())
	return audioTrack, nil
}
