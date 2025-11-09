//go:build cgo && opus

package codec

import (
	"errors"
	"p2p-call/internal/audio/codec/iface"
	"p2p-call/internal/audio/codec/opus"
	"p2p-call/internal/audio/codec/pcmu"
	"p2p-call/internal/audio/config"
)

// CreateEncoder создает encoder на основе конфигурации
// В версии с CGO доступны оба кодека
func CreateEncoder(cfg config.AudioConfig) (iface.Encoder, error) {
	switch cfg.Type {
	case config.AudioCodecPCMU:
		return pcmu.NewPCMUEncoder(), nil
	case config.AudioCodecOpus:
		return opus.NewOpusEncoderFromConfig(cfg)
	default:
		return nil, errors.New("unknown codec type")
	}
}

// CreateDecoder создает decoder на основе конфигурации
// В версии с CGO доступны оба кодека
func CreateDecoder(cfg config.AudioConfig) (iface.Decoder, error) {
	switch cfg.Type {
	case config.AudioCodecPCMU:
		return pcmu.NewPCMUDecoder(), nil
	case config.AudioCodecOpus:
		return opus.NewOpusDecoderFromConfig(cfg)
	default:
		return nil, errors.New("unknown codec type")
	}
}
