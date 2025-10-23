package config

import (
	"log"
	"os"
	"strings"

	"github.com/pion/webrtc/v4"
)

func getServersFromString(envServers string) []string {
	servers := strings.Split(envServers, ",")
	for i, server := range servers {
		servers[i] = strings.TrimSpace(server)
	}
	return servers
}

func GetStunServers() []webrtc.ICEServer {
	envServer := os.Getenv("STUN_SERVERS")
	if envServer == "" {
		log.Fatalf("STUN_SERVERS not set in environment, cant run app")
	}
	serverList := getServersFromString(envServer)

	stunServers := make([]webrtc.ICEServer, len(serverList))
	for i, server := range serverList {
		stunServers[i] = webrtc.ICEServer{
			URLs: []string{server},
		}
	}
	return stunServers
}
func GetTurnServers() []webrtc.ICEServer {
	username := os.Getenv("TURN_USERNAME")
	credential := os.Getenv("TURN_CREDENTIAL")
	envServer := os.Getenv("TURN_SERVERS")
	if envServer == "" {
		log.Println("Warning: TURN server configuration missing in environment, some connections may fail")
	}

	serverList := getServersFromString(envServer)

	turnServers := make([]webrtc.ICEServer, len(serverList))
	for i, server := range serverList {
		turnServers[i] = webrtc.ICEServer{
			URLs:       []string{server},
			Username:   username,
			Credential: credential,
		}
	}
	return turnServers
}
