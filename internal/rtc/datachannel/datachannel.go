package datachannel

type MessageType string

const (
	MessageTypeText   MessageType = "text"
	MessageTypeImage  MessageType = "image"
	MessageTypeFile   MessageType = "file"
	MessageTypeSystem MessageType = "system"
	MessageTypeTyping MessageType = "typing"
)

type DataChannelMessage struct {
	Type    MessageType `json:"type"`
	Payload string      `json:"payload"`
}
