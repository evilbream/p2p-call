package rtc

import (
	"sync"
	"time"
)

type ConnectionState int

const (
	StateDisconnected ConnectionState = iota
	StateConnecting
	StateConnected
	StateFailed
	StateClosed
	StateChecking
)

func (cs ConnectionState) String() string {
	switch cs {
	case StateDisconnected:
		return "Disconnected"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateFailed:
		return "Failed"
	case StateClosed:
		return "Closed"
	case StateChecking:
		return "Checking"
	default:
		return "Unknown"
	}
}

type ConnectionEvent struct {
	State     ConnectionState
	Error     error
	Message   string
	Timestamp int64
}

type StateObserver interface {
	OnStateChange(event ConnectionEvent)
}

type StateManager struct {
	mu        sync.RWMutex
	state     ConnectionState
	observers []StateObserver
	events    chan ConnectionEvent
}

func NewStateManager() *StateManager {
	sm := &StateManager{
		state:     StateDisconnected,
		events:    make(chan ConnectionEvent, 5),
		observers: make([]StateObserver, 0),
	}
	go sm.eventLoop()
	return sm
}

func (sm *StateManager) GetState() ConnectionState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// eventLoop processes connection events and notifies observers
func (sm *StateManager) eventLoop() {
	for event := range sm.events {
		sm.mu.RLock()
		observers := make([]StateObserver, len(sm.observers))
		copy(observers, sm.observers)
		sm.mu.RUnlock()

		for _, observer := range observers {
			observer.OnStateChange(event)
		}
	}
}

func (sm *StateManager) Subscribe(observer StateObserver) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.observers = append(sm.observers, observer)
}

func (sm *StateManager) Unsubscribe(observer StateObserver) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	for i, obs := range sm.observers {
		if obs == observer {
			sm.observers = append(sm.observers[:i], sm.observers[i+1:]...)
			break
		}
	}
}

func (sm *StateManager) UpdateState(state ConnectionState, message string, err error) {
	sm.mu.Lock()
	oldState := sm.state
	sm.state = state
	sm.mu.Unlock()

	if oldState != state {
		event := ConnectionEvent{
			State:     state,
			Error:     err,
			Timestamp: time.Now().Unix(),
			Message:   message,
		}
		sm.events <- event
	}
}
