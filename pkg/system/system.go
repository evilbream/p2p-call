package system

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func GenerateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func WaitForUserResponse(exit bool, prompt ...string) {
	var strPrompt string
	if len(prompt) == 0 {
		strPrompt = "\nPress Enter to exit..."
	} else {
		strPrompt = strings.Join(prompt, " ")
	}
	fmt.Println(strPrompt)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	if exit {
		os.Exit(1)
	}

}
