package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"p2p-call/config"
	"p2p-call/system"
	"strings"
	"time"

	"github.com/pion/webrtc/v3"
)

type SignalData struct {
	Type      string                     `json:"type"`
	SDP       *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate *webrtc.ICECandidate       `json:"candidate,omitempty"`
	SessionID string                     `json:"session_id"`
}

func main() {
	for i := range 4 { // attempt to connect with fallback configs
		if i == 0 { // try direct connection first
			//log.Println("Trying direct manual connection")
			//log.Println("manual udp hole punching not implemented yet")
			continue
		}
		if err := tryConnect(i); err == nil {
			return
		}
	}
}

func tryConnect(attempt int) error {
	// create nat config
	log.Println("Trying connection with config attempt ", attempt)
	config := createConfigForNATType(attempt)

	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetICETimeouts(
		time.Second*45, // Disconnected timeout upped for double NAT
		time.Second*10, // Failed timeout
		time.Second*15, // Keepalive interval
	)

	settingEngine.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
		webrtc.NetworkTypeTCP4,
		webrtc.NetworkTypeTCP6,
	})

	mediaEngine := &webrtc.MediaEngine{}
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(settingEngine),
	)

	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		log.Fatalf("Failed to create PeerConnection: %v", err)
	}
	defer peerConnection.Close()

	dcOptions := &webrtc.DataChannelInit{
		Ordered:        &[]bool{true}[0],
		MaxRetransmits: &[]uint16{15}[0],
	}

	dataChannel, err := peerConnection.CreateDataChannel("messages", dcOptions)
	if err != nil {
		log.Fatalf("failed to create DataChannel: %v", err)
	}

	sessionID := generateSessionID()

	connectionResult := make(chan error, 1)
	setupEventHandlers(peerConnection, dataChannel, sessionID, connectionResult)

	fmt.Println("What to do?:")
	fmt.Println("1. send an offer")
	fmt.Println("2. accept an offer")
	fmt.Print("Enter 1 or 2: ")

	reader := bufio.NewReader(os.Stdin)
	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	switch choice {
	case "1":
		createAndSendOffer(peerConnection, sessionID)
	case "2":
		receiveAndProcessOffer(peerConnection, sessionID)
	default:
		log.Fatal("Invalid choice")
	}

	select {
	case err := <-connectionResult:
		if err != nil {
			return err
		}
	case <-time.After(900 * time.Second): // 15 minute timeout
		return fmt.Errorf("connection timeout")
	}

	select {}
}

func createConfigForNATType(attempt int) webrtc.Configuration {
	stunServers := config.GetStunServers()

	turnServers := config.TestTurnServers()

	config := webrtc.Configuration{
		BundlePolicy:  webrtc.BundlePolicyMaxBundle,
		RTCPMuxPolicy: webrtc.RTCPMuxPolicyRequire,
	}

	switch attempt {

	case 2:
		log.Printf("Config for defaule nat added turn sercers\n")
		config.ICEServers = append(stunServers, turnServers[:2]...)
		config.ICECandidatePoolSize = 25

	case 1:
		log.Printf("Config only stun servers\n")
		config.ICEServers = stunServers // only stun servers
		config.ICECandidatePoolSize = 20

	default:
		log.Printf("Universal config\n")
		config.ICEServers = append(stunServers, turnServers...)
		config.ICECandidatePoolSize = 30
	}

	log.Printf("Used %d ICE serveres\n", len(config.ICEServers))
	return config
}

func setupEventHandlers(pc *webrtc.PeerConnection, dc *webrtc.DataChannel, sessionID string, connectionResult chan error) error {
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			var connType string
			switch candidate.Typ.String() {
			case "host":
				connType = "Direct"
			case "srflx":
				connType = "STUN"
			case "relay":
				connType = "TURN"
			case "prflx":
				connType = "Peer"
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

		switch state {
		case webrtc.ICEConnectionStateConnected:
			log.Println("Ice connection is set!")
		case webrtc.ICEConnectionStateFailed:
			log.Println("Ice connection failed")
			connectionResult <- fmt.Errorf("ice connection failed")
		case webrtc.ICEConnectionStateDisconnected:
			log.Println("ICE disconnected...")
		case webrtc.ICEConnectionStateClosed:
			log.Println("ICE connection closed ")
		case webrtc.ICEConnectionStateChecking:
			log.Println("Checking candidates...")
		case webrtc.ICEConnectionStateCompleted:
			log.Println("found optimal path")
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

	setupDataChannelHandlers(dc, sessionID)
	return nil
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

	pc.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		fmt.Printf("ICE Gathering: %s\n", state.String())
		if state == webrtc.ICEGathererStateComplete {
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

func generateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
