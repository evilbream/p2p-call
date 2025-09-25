package config

import (
	"net"
	"testing"
	"time"

	"github.com/pion/stun"
)

func TestTurnServers(t *testing.T) {

	turnServers := GetTurnServers()
	if len(turnServers) == 0 {
		t.Fatalf("Expected at least one TURN server, got 0")
	}

	for _, server := range turnServers {
		if len(server.URLs) == 0 {
			t.Error("TURN server has no URLs")
		}
		if server.Username == "" {
			t.Error("TURN server missing username")
		}
		if server.Credential == "" {
			t.Error("TURN server missing credential")
		}
	}

}

func TestStunServers(t *testing.T) {
	stunServers := GetStunServers()
	if len(stunServers) == 0 {
		t.Error("No STUN servers configured")
	}

	for _, server := range stunServers {
		if len(server.URLs) == 0 {
			t.Error("STUN server has no URLs")
			continue
		}

		for _, url := range server.URLs {
			t.Run(url, func(t *testing.T) {
				available := testStunServerAvailability(url)
				if !available {
					t.Errorf("STUN server %s is not available", url)
				} else {
					t.Logf("STUN server %s is available", url)
				}
			})
		}
	}
}

func testStunServerAvailability(stunURL string) bool {
	address := stunURL[5:] // Remove "stun:"

	// Create UDP connection
	conn, err := net.Dial("udp", address)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Set timeout
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Create STUN binding request
	m := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	// Send request
	_, err = conn.Write(m.Raw)
	if err != nil {
		return false
	}

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return false
	}

	// Parse response
	var response stun.Message
	response.Raw = buf[:n]
	err = response.Decode()
	if err != nil {
		return false
	}

	// Check if it's a valid STUN response
	return response.Type == stun.BindingSuccess
}
