package observer

import (
	"p2p-call/internal/rtc"
	"p2p-call/pkg/interface/desktop/views"
)

type UIObserver struct {
	connectionView views.ConnectionView
	errorView      views.ErrorView
	onConnect      func()
	onError        func()
}

func NewUIObserver(connView views.ConnectionView, errView views.ErrorView) *UIObserver {
	return &UIObserver{
		connectionView: connView,
		errorView:      errView,
	}
}

// OnStateChange handles connection state changes and updates the UI accordingly
func (o *UIObserver) OnStateChange(event rtc.ConnectionEvent) {
	switch event.State {
	case rtc.StateConnected:
		o.connectionView.UpdateStatus("Connected")
		if o.onConnect != nil {
			o.onConnect()
		}
	case rtc.StateConnecting:
		o.connectionView.UpdateStatus("Connecting...")
	case rtc.StateDisconnected:
		o.connectionView.UpdateStatus("Disconnected")
	case rtc.StateChecking:
		o.connectionView.UpdateStatus("Checking Connection...")
	case rtc.StateFailed:
		o.connectionView.UpdateStatus("Connection Failed")
		o.errorView.ShowError(event.Error, views.ErrorWarning)
		if o.onError != nil {
			o.onError()
		}
	case rtc.StateClosed:
		o.connectionView.UpdateStatus("Connection Closed")
	}
}

func (o *UIObserver) SetOnConnect(f func()) {
	o.onConnect = f
}

func (o *UIObserver) SetOnError(f func()) {
	o.onError = f
}
