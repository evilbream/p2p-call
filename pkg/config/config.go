package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"p2p-call/pkg/system"
	"strings"
	"time"

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

func GenerateSelfSignedCert() (tls.Certificate, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Voice Recorder"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:    []string{"localhost"},
	}

	localIP := system.GetLocalIP()
	if localIP != "" {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(localIP))
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	return tls.X509KeyPair(certPEM, keyPEM)
}
