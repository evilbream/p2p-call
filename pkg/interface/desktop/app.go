package desktop

import (
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
		if err != nil {
			a.ShowError(err, views.ErrorWarning)
		}
	}
}

func (a *App) ShowConnectionView() {
	a.currentView = a.connectionView.Build()
	a.window.SetContent(a.currentView)
}

func (a *App) ShowError(err error, severity views.ErrorSeverity) {
	a.errorView.ShowError(err, severity)
	a.window.SetContent(a.errorView.Build())
}

func (app *App) Run() {
	app.window.SetContent(app.connectionView.Build())
	app.window.ShowAndRun()
}
