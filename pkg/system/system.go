package system

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"strings"
)

func ReadMultilineJSON() string {
	reader := bufio.NewReader(os.Stdin)
	var lines []string

	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		if line == "" {
			break
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "")
}

func GenerateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "localhost"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func GetWebPort() string {
	port := os.Getenv("HTTPS_PORT")
	if port == "" {
		port = "8443"
	}
	return port
}
