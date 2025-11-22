package viewmodel

import (
	"sync"
	"time"
)

type OnMessageReceivedFunc func(text string)
type SendMessageFunc func(text string) error

type ChatMessage struct {
	Text      string
	Timestamp time.Time
	IsOwn     bool
}

type ChatViewModel struct {
	messages []ChatMessage
	mu       sync.RWMutex
}

func NewChatViewModel() *ChatViewModel {
	return &ChatViewModel{
		messages: make([]ChatMessage, 0),
	}
}

func (cvm *ChatViewModel) OnMessageReceived(text string, isOwn bool) {

}
