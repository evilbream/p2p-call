package desktop

import (
	"context"
	"errors"
	"p2p-call/internal/audio/capture"
	"p2p-call/internal/audio/playback"
	"p2p-call/pkg/interface/desktop/observer"
	"p2p-call/pkg/interface/desktop/theme"
	"p2p-call/pkg/interface/desktop/views"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

type App struct {
	window         fyne.Window
	app            fyne.App
	connectionView *views.ConnectionView
	rendezvousView *views.RendezvousView
	errorView      *views.ErrorView
	MainView       *views.MainView
	currentView    fyne.CanvasObject
	UiObserver     *observer.UIObserver
}

func NewApp() (*App, error) {
	a := app.New()
	a.Settings().SetTheme(theme.MonochromeTheme{})

	w := a.NewWindow("p2p call")
	w.Resize(fyne.NewSize(450, 650))

	connectionView := views.NewConnectionView()
	errorView := views.NewErrView()

	uiObserver := observer.NewUIObserver(*connectionView, *errorView)
	uiObserver.SetOnError(func() { w.SetContent(errorView.Build()) })

	return &App{
		app:            a,
		window:         w,
		errorView:      errorView,
		connectionView: connectionView,
		UiObserver:     uiObserver,
	}, nil
}

func (a *App) InitMainView(capture *capture.MalgoCapture, playback *playback.MalgoPlayback,
	sendMessage func(text string) error) {
	a.MainView = views.NewMainView(capture, playback, sendMessage)
	a.UiObserver.SetOnConnect(func() { fyne.Do(func() { a.window.SetContent(a.MainView.Build()) }) })
}

func (a *App) LogConnectionErrors(errChan chan error) {
	for err := range errChan {
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			a.ShowError(err, views.ErrorWarning)
		}
	}
}

func (a *App) ShowConnectionView() {
	a.currentView = a.connectionView.Build()
	a.window.SetContent(a.currentView)
}

func (a *App) SetRendezvousView(onConnect func(string)) {
	a.rendezvousView = views.NewRendezvousView(onConnect)
}

func (a *App) ShowRendezvousView() {
	if a.rendezvousView != nil {
		a.window.SetContent(a.rendezvousView.Build())
	}
}

func (a *App) ShowError(err error, severity views.ErrorSeverity) {
	fyne.Do(func() {
		a.errorView.ShowError(err, severity)
		a.window.SetContent(a.errorView.Build())
	})
}

func (app *App) Run() {
	// Show rendezvous view first if set, otherwise show connection view
	if app.rendezvousView != nil {
		app.window.SetContent(app.rendezvousView.Build())
	} else {
		app.window.SetContent(app.connectionView.Build())
	}
	app.window.ShowAndRun()
}

func (app *App) SetDisconnectCallback(callback func()) {
	if app.MainView != nil {
		app.MainView.SetDisconnectCallback(callback)
	}
	if app.connectionView != nil {
		app.connectionView.SetCancelCallback(callback)
	}
}
