package handler

import (
	"github.com/pion/webrtc/v4"
)

type AudioHandlerInterface interface {
	StartAudioCapture(audioTrack *webrtc.TrackLocalStaticSample)
	HandleIncomingAudio(track *webrtc.TrackRemote)

	// Getters for channels and codec
	GetAudioCaptureChan() chan []byte
	GetPlayAudioChan() chan []byte
}
