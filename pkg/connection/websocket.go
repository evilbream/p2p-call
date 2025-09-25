package connection

import (
	"log"
	"net/http"
	"p2p-call/internal/audio"
	"p2p-call/internal/audio/convert"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4/pkg/media"
)

// function to process audiochunk data from websocket
type WebsocketHandler struct {
	Audio       audio.AudioHandlerInterface
	currentConn *websocket.Conn // current active websocket connection
	sampleBuf   []float32
	frameSize   int
	//writeMutex  sync.Mutex
}

func (wh *WebsocketHandler) initDefaults() {
	if wh.frameSize == 0 {
		wh.frameSize = 960
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin:       func(r *http.Request) bool { return true },
	ReadBufferSize:    1024 * 10, // 16KB for audio data
	WriteBufferSize:   1024 * 10, // 16KB for audio data
	EnableCompression: false,     // Disable compression for audio
}

func (wh *WebsocketHandler) SendWebsocketMessage() error {
	for data := range wh.Audio.GetPlayAudioChan() {
		err := wh.currentConn.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			log.Println("Write error:", err)
			return err
		}
		//log.Println("Sent audio data to client:", len(data), "bytes")
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
	wh.initDefaults()
	defer conn.Close()
	go wh.SendWebsocketMessage()
	encoder := wh.Audio.GetEncoder()
	audioCaptureChan := wh.Audio.GetAudioCaptureChan()
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		if messageType != websocket.BinaryMessage {
			log.Printf("Ignoring non-binary message of type: %d", messageType)
			continue
		}

		samples := convert.BytesToFloat32(message)
		if len(samples) == 0 {
			continue
		}

		wh.sampleBuf = append(wh.sampleBuf, samples...)

		for len(wh.sampleBuf) >= wh.frameSize {
			frame := wh.sampleBuf[:wh.frameSize] // 960 samples

			//  (expected 960 mono float32)
			pkts, err := encoder.EncodeFloat32(frame)
			if err != nil {
				log.Println("Encode error:", err)
				break
			}
			for _, pkt := range pkts {
				durationMs := float64(wh.frameSize) / float64(48000) * 1000.0
				duration := time.Duration(durationMs * float64(time.Millisecond))
				select {
				case audioCaptureChan <- media.Sample{
					Data:     pkt,
					Duration: duration,
				}:
				default:
					log.Println("AudioCaptureChan full, drop packet")
				}
			}

			wh.sampleBuf = wh.sampleBuf[wh.frameSize:]
		}

		/*
			decoded, err := wh.Audio.Decoder.DecodePackets(opusData)
			if err != nil {
				log.Println("Decoding error:", err)
				continue
			}
			log.Printf("Decoded back to PCM: %d bytes", len(decoded))
			conn.WriteMessage(websocket.BinaryMessage, convert.Float32ToBytes(decoded))
		*/

	}
}
