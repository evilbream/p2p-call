package tui

import (
	"bufio"
	"fmt"
	"os"
	"p2p-call/internal/audio/capture"
	"p2p-call/internal/audio/playback"
	"strings"

	"github.com/rs/zerolog/log"
)

type DesktopInterface struct {
	capture  *capture.MalgoCapture
	playback *playback.MalgoPlayback
}

func NewDesktopInterface(capture *capture.MalgoCapture, playback *playback.MalgoPlayback) (*DesktopInterface, error) {
	if capture == nil || playback == nil {
		return nil, fmt.Errorf("pparams cant be nill")
	}
	return &DesktopInterface{
		capture:  capture,
		playback: playback,
	}, nil
}

func (di *DesktopInterface) StartDesktopInterface() {
	// Implementation for starting the desktop interface
	menu := "1. Unmute\n2. Mute\n3. Play sound\n 4. Stop sound\n5. Exit"
	println("Desktop Interface Started\nBy default u are muted and sound is on")
	println("Menu:")
	println(menu)
	reader := bufio.NewReader(os.Stdin)
	for {
		print("Enter choice: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			println("Unmuted")
			di.capture.Paused = false
		case "2":
			println("Muted")
			di.capture.Paused = true
		case "3":
			println("Playing sound")
			di.playback.Paused = false
		case "4":
			println("Stopping sound")
			di.playback.Paused = true
		case "5":
			println("Exiting...")
			return
		default:
			println("Invalid choice, please try again.")
		}
	}
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

// reads connection log and process errors
func LogConnectionErrors(connErrors chan error) {
	for {
		err := <-connErrors
		if err != nil {
			log.Error().Err(err).Msg("connection error")
			WaitForUserResponse(true)
		}
	}
}
