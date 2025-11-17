package datachannel

import (
	"encoding/json"

	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog/log"
)

func (dcm *DataChannelManager) setupHandlers() {
	dcm.dataChannel.OnOpen(dcm.onOpen)
	dcm.dataChannel.OnClose(dcm.onClose)
	dcm.dataChannel.OnMessage(dcm.onMessage)
	dcm.dataChannel.OnError(dcm.onError)
}

func (dcm *DataChannelManager) onOpen() {
	log.Info().Msg("Data channel opened")
	if dcm.OnStateChange != nil {
		dcm.OnStateChange(webrtc.DataChannelStateOpen)
	}
	go dcm.processSendQueue()
}

func (dcm *DataChannelManager) onClose() {
	log.Info().Msg("Data channel closed")
	if dcm.OnStateChange != nil {
		dcm.OnStateChange(webrtc.DataChannelStateClosed)
	}
}

func (dcm *DataChannelManager) onMessage(msg webrtc.DataChannelMessage) {
	log.Debug().Msgf("Received message over data channel: %d bytes", len(msg.Data))
	var dcMsg DataChannelMessage
	if err := json.Unmarshal(msg.Data, &dcMsg); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal data channel message")
		return
	}
	log.Debug().Msgf("Received data channel message of type: %s", dcMsg.Type)
	if dcm.OnMessage != nil {
		dcm.OnMessage(dcMsg)
	}
}

func (dcm *DataChannelManager) onError(err error) {
	log.Error().Err(err).Msg("Data channel error")
}
