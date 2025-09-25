package audio

import (
	"p2p-call/internal/audio/codec"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type AudioHandlerInterface interface {
	StartAudioCapture(audioTrack *webrtc.TrackLocalStaticSample)
	HandleIncomingAudio(track *webrtc.TrackRemote)

	// Getters for channels and codec
	GetAudioCaptureChan() chan media.Sample
	GetPlayAudioChan() chan []byte
	GetEncoder() *codec.OpusEncoder
	GetDecoder() *codec.OpusDecoder
}
