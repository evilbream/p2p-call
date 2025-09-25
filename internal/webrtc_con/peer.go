package webrtccon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"p2p-call/internal/audio"
	"p2p-call/pkg/config"
	"p2p-call/pkg/system"
	"strings"
	"time"

	"github.com/pion/webrtc/v4"
)

type WebRtcConnector struct {
	Audio audio.AudioHandlerInterface
}

type SignalData struct {
	Type      string                     `json:"type"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidate       `json:"candidate,omitempty"`
	SessionID string                     `json:"session_id"`
}

func (wc WebRtcConnector) Connect(attempt int) error {
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

	audioTrack, err := setupAudioTrack(peerConnection)
	if err != nil {
		log.Fatalf("Failed to setup audio track: %v", err)
	}

	// DataChannel options
	isOrdered := true
	maxRetransmits := uint16(15)

	dcOptions := &webrtc.DataChannelInit{
		Ordered:        &isOrdered, // keep message order
		MaxRetransmits: &maxRetransmits,
	}

	dataChannel, err := peerConnection.CreateDataChannel("messages", dcOptions)
	if err != nil {
		log.Fatalf("failed to create DataChannel: %v", err)
	}

	sessionID := system.GenerateSessionID()
	fmt.Printf("Session ID: %s\n", sessionID)

	connectionResult := make(chan error) // channel to signal connection result and retry on error
	wc.setupEventHandlers(peerConnection, dataChannel, sessionID, connectionResult)

	fmt.Println("What to do?:")
	fmt.Println("1. send an offer")
	fmt.Println("2. accept an offer")
	fmt.Print("Enter 1 or 2: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	go wc.Audio.StartAudioCapture(audioTrack)

	switch choice {
	case "1":
		createAndSendOffer(peerConnection, sessionID)
	case "2":
		receiveAndProcessOffer(peerConnection, sessionID)
	default:
		log.Fatal("Invalid choice")
	}

	err = <-connectionResult
	if err != nil {
		return err
	}

	select {}
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

func (wc WebRtcConnector) setupEventHandlers(pc *webrtc.PeerConnection, dc *webrtc.DataChannel, sessionID string, connectionResult chan error) error {
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
			connectionResult <- nil // signal successful connection
		case webrtc.ICEConnectionStateFailed:
			log.Println("Ice connection failed")
			connectionResult <- fmt.Errorf("ice connection failed")
		case webrtc.ICEConnectionStateDisconnected:
			log.Println("ICE disconnected...")
		case webrtc.ICEConnectionStateClosed:
			log.Println("ICE connection closed ")

		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {

		switch state {
		case webrtc.PeerConnectionStateConnected:
			log.Println("Peer connection established")
			log.Println("You can start typing messages now!")

		case webrtc.PeerConnectionStateFailed:
			log.Println("Peer connection failed")
			connectionResult <- fmt.Errorf("peer connection failed")

		case webrtc.PeerConnectionStateClosed:
			log.Println("Peer connection closed")
		case webrtc.PeerConnectionStateConnecting:
			log.Println("Peer connection connecting...")
		}
	})

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		setupDataChannelHandlers(dc, sessionID)
	})

	// its stream not
	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("Received track: %s, type: %s", track.ID(), track.Kind().String())

		if track.Kind() == webrtc.RTPCodecTypeAudio {
			log.Println("Audio track received from peer")
			go wc.Audio.HandleIncomingAudio(track)
		}
	})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
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

	setupDataChannelHandlers(dc, sessionID)
	return nil
}

func setupAudioTrack(pc *webrtc.PeerConnection) (*webrtc.TrackLocalStaticSample, error) {
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeOpus,
			Channels:  1,
			ClockRate: 48000},
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

func setupDataChannelHandlers(dc *webrtc.DataChannel, sessionID string) {
	dc.OnOpen(func() {
		log.Println("Data channel opened")
		go handleUserInput(dc, sessionID)
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Printf("%s\n", string(msg.Data))
	})

	dc.OnClose(func() {
		log.Println("Message channel closed")
	})

	dc.OnError(func(err error) {
		log.Println("Data channel error:", err)
	})
}

func handleUserInput(dc *webrtc.DataChannel, sessionID string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		if dc.ReadyState() == webrtc.DataChannelStateOpen {
			fmt.Print("> ")
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)

			if text == "quit" || text == "exit" {
				return
			}

			if text != "" {
				message := fmt.Sprintf("[%s]: %s", sessionID, text)
				err := dc.SendText(message)
				if err != nil {
					log.Println("Error sending message:", err)
				}
			}
		} else {
			log.Println("Data channel not open, waiting...")
			time.Sleep(1 * time.Second)
		}
	}
}

func createAndSendOffer(pc *webrtc.PeerConnection, sessionID string) {
	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Fatalf("Failed to create offer: %v", err)
	}

	err = pc.SetLocalDescription(offer)
	if err != nil {
		log.Printf("Failed to set local description: %v", err)
	}

	waitForICEGathering(pc)

	finalOffer := pc.LocalDescription()
	signalData := SignalData{
		Type:      "offer",
		SDP:       finalOffer,
		SessionID: sessionID,
	}

	offerJSON, err := json.Marshal(signalData)
	if err != nil {
		log.Fatalf("Failed to marshal offer: %v", err)
	}
	fmt.Println("\nCopy offer and send it:")
	fmt.Println(string(offerJSON))
	fmt.Println("\nEnter accept:")

	receiveAnswer(pc)
}

func receiveAndProcessOffer(pc *webrtc.PeerConnection, sessionID string) {
	fmt.Println("Enter offer:")

	offer := system.ReadMultilineJSON()

	var signalData SignalData
	err := json.Unmarshal([]byte(offer), &signalData)
	if err != nil {
		log.Fatalf("Failed to parse offer: %v", err)
	}

	err = pc.SetRemoteDescription(*signalData.SDP)
	if err != nil {
		log.Fatalf("Failed to set remote description: %v", err)
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Fatalf("Failed to create answer: %v", err)
	}

	err = pc.SetLocalDescription(answer)
	if err != nil {
		log.Fatalf("Failed to set local description: %v", err)
	}

	fmt.Println("Fetching ICE candidates...")
	waitForICEGathering(pc)

	finalAnswer := pc.LocalDescription()
	answerData := SignalData{
		Type:      "answer",
		SDP:       finalAnswer,
		SessionID: sessionID,
	}

	answerJSON, err := json.Marshal(answerData)
	if err != nil {
		log.Fatalf("Failed to marshal answer: %v", err)
	}
	fmt.Println("\nAnswer to send:")
	fmt.Println(string(answerJSON))
}

func receiveAnswer(pc *webrtc.PeerConnection) {
	answer := system.ReadMultilineJSON()

	var signalData SignalData
	err := json.Unmarshal([]byte(answer), &signalData)
	if err != nil {
		log.Fatalf("Failed to parse answer: %v", err)
	}

	err = pc.SetRemoteDescription(*signalData.SDP)
	if err != nil {
		log.Fatalf("Failed to set remote description: %v", err)
	}
}

func waitForICEGathering(pc *webrtc.PeerConnection) {
	done := make(chan struct{})

	pc.OnICEGatheringStateChange(func(state webrtc.ICEGatheringState) {
		fmt.Printf("ICE Gathering: %s\n", state.String())
		if state == webrtc.ICEGatheringStateComplete {
			close(done)
		}
	})

	select {
	case <-done:
		fmt.Println("ICE candidates gathered")
	case <-time.After(45 * time.Second):
		fmt.Println("ICE gathering timeout, proceeding with gathered candidates")
	}
}
