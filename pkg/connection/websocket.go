package connection

import (
	"log"
	"net/http"
	"p2p-call/internal/audio"
	"p2p-call/internal/audio/codec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3/pkg/media"
)

// function to process audiochunk data from websocket
type WebsocketHandler struct {
	Audio       audio.AudioHandler
	currentConn *websocket.Conn // current active websocket connection
	writeMutex  sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  4096 * 4, // 16KB for audio data
	WriteBufferSize: 4096 * 4, // 16KB for audio data
}

func (wh *WebsocketHandler) SendWebsocketMessage() error {
	for data := range wh.Audio.PlayAudioChan {
		wh.writeMutex.Lock()
		err := wh.currentConn.WriteMessage(websocket.BinaryMessage, data)
		wh.writeMutex.Unlock()
		if err != nil {
			log.Println("Write error:", err)
			return err
		}
	}
	return nil
}

func (wh *WebsocketHandler) HandleWebsocketMessage(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}
	wh.currentConn = conn
	defer conn.Close()
	go wh.SendWebsocketMessage()
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}
		switch messageType {
		case websocket.TextMessage:
			log.Printf("Received text message: %s", message)
		case websocket.BinaryMessage:
			log.Printf("Received binary message: %d bytes", len(message))

			pcmuData := codec.LinearToPCMUandResampling(message)
			// Duration для 16kHz
			duration := time.Duration(len(pcmuData)*1000/16000) * time.Millisecond

			wh.Audio.AudioCaptureChan <- media.Sample{
				Data:     pcmuData,
				Duration: duration,
			}

		default:
			log.Printf("Received unknown message type: %d", messageType)
		}
	}
}
