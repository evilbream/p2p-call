package system

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
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
