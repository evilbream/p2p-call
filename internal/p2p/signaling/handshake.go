package signaling

import "sync"

// set connection
type HandshakeManager struct {
	once  sync.Once
	Ready chan struct{}
}

func NewHandshake() *HandshakeManager {
	return &HandshakeManager{
		Ready: make(chan struct{}),
	}
}

func (h *HandshakeManager) MarkReady() {
	h.once.Do(func() {
		close(h.Ready)
	})
}

func (h *HandshakeManager) Wait() {
	<-h.Ready
}
