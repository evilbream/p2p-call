package views

import (
	"p2p-call/pkg/interface/desktop/theme"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ErrorSeverity int

const (
	ErrorInfo ErrorSeverity = iota
	ErrorWarning
	ErrorFatal
)

type ErrorView struct {
	messageLabel *widget.Label
	content      *fyne.Container
	severity     ErrorSeverity
	titleLabel   *widget.Label
	background   *canvas.Rectangle
}

func NewErrView() *ErrorView {
	v := &ErrorView{}
	// Title
	v.titleLabel = widget.NewLabel("Error")
	v.titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	v.titleLabel.Alignment = fyne.TextAlignCenter

	v.messageLabel = widget.NewLabel("No error...")
	v.messageLabel.Alignment = fyne.TextAlignCenter

	content := container.NewVBox(
		v.titleLabel,
		widget.NewSeparator(),
		v.messageLabel,
	)

	v.content = container.NewStack(
		v.background,
		container.NewPadded(content),
	)

	return v
}

// ShowError displays error with specified severity
func (v *ErrorView) ShowError(err error, severity ErrorSeverity) {
	v.severity = severity
	v.updateTheme()

	switch severity {
	case ErrorInfo:
		fyne.Do(func() {
			v.titleLabel.SetText("Information")
		})
	case ErrorWarning:
		fyne.Do(func() {
			v.titleLabel.SetText("Warning")
		})
	case ErrorFatal:
		fyne.Do(func() {
			v.titleLabel.SetText("Critical Error")
		})
	}

	fyne.Do(func() {
		v.messageLabel.SetText(err.Error())
		v.messageLabel.Show()
	})
}

// updateTheme applies color based on severity
func (v *ErrorView) updateTheme() {
	switch v.severity {
	case ErrorInfo:
		v.background.FillColor = theme.ErrorColorInfo
	case ErrorWarning:
		v.background.FillColor = theme.ErrorColorWarning
	case ErrorFatal:
		v.background.FillColor = theme.ErrorColorCritical
	}
	v.background.Refresh()
}

func (v *ErrorView) Build() fyne.CanvasObject {
	return v.content
}
