package logger

import (
	"p2p-call/internal/rtc"

	"github.com/rs/zerolog/log"
)

type LoggerObserver struct {
}

func NewLoggerObserver() *LoggerObserver {
	return &LoggerObserver{}
}

func (lo *LoggerObserver) OnStateChange(event rtc.ConnectionEvent) {
	log.Info().
		Str("state", event.State.String()).
		Str("message", event.Message).
		Int64("timestamp", event.Timestamp).
		Msg("Connection state changed")

	if event.Error != nil {
		log.Error().Err(event.Error).Msg("Connection error")
	}
}
