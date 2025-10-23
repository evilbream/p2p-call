package system

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadMultilineJSON reads JSON from stdin or file
func ReadMultilineJSON() string {
	fmt.Println("Options:")
	fmt.Println("1. Paste JSON (press Enter twice when done)")
	fmt.Println("2. Read from file (type 'file:path/to/offer.json')")
	fmt.Print("Choose or paste JSON: ")

	reader := bufio.NewReaderSize(os.Stdin, 1024*1024)
	firstLine, _ := reader.ReadString('\n')
	firstLine = strings.TrimSpace(firstLine)

	// read data from file
	if after, ok := strings.CutPrefix(firstLine, "file:"); ok {
		filePath := after
		data, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			return ReadMultilineJSON() // retry
		}
		return string(data)
	}

	// if its json read
	var lines []string
	lines = append(lines, firstLine)

	emptyLines := 0
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)

		if line == "" {
			emptyLines++
			if emptyLines >= 2 {
				break
			}
			continue
		}

		emptyLines = 0
		lines = append(lines, line)

		if err == io.EOF {
			break
		}
	}

	return strings.Join(lines, "")
}

func GenerateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
