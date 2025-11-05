package logger

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger инициализирует zerolog с заданным уровнем логирования
func InitLogger() {
	var logLevel zerolog.Level
	logLevlStr := os.Getenv("LOG_LEVEL")
	switch logLevlStr {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.SetGlobalLevel(logLevel)
	log.Info().
		Str("level", logLevel.String()).
		Str("time_format", "unix ms").
		Msg("Logger initialized")
}
