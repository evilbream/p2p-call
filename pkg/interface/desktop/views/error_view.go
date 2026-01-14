package views

import (
	"image/color"
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
	titleLabel   *canvas.Text
	background   *canvas.Rectangle
}

func NewErrView() *ErrorView {
	// Initialize background first
	background := canvas.NewRectangle(theme.ErrorColorInfo)

	// Title using canvas.Text for color control
	titleLabel := canvas.NewText("Error", color.White)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter
	titleLabel.TextSize = 18

	messageLabel := widget.NewLabel("No error...")
	messageLabel.Alignment = fyne.TextAlignCenter
	messageLabel.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		titleLabel,
		widget.NewSeparator(),
		messageLabel,
	)

	stack := container.NewStack(
		background,
		container.NewPadded(content),
	)

	return &ErrorView{
		titleLabel:   titleLabel,
		messageLabel: messageLabel,
		background:   background,
		content:      stack,
		severity:     ErrorInfo,
	}
}

// ShowError displays error with specified severity
func (v *ErrorView) ShowError(err error, severity ErrorSeverity) {
	v.severity = severity

	fyne.Do(func() {
		// Update theme first
		v.updateTheme()

		// Update title based on severity
		switch severity {
		case ErrorInfo:
			v.titleLabel.Text = "Information"
		case ErrorWarning:
			v.titleLabel.Text = "Warning"
		case ErrorFatal:
			v.titleLabel.Text = "Critical Error"
		}
		v.titleLabel.Refresh()

		// Update message
		v.messageLabel.SetText(err.Error())
		v.messageLabel.Show()
	})
}

// updateTheme applies color based on severity
func (v *ErrorView) updateTheme() {
	if v.background == nil {
		return
	}

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

func (v *ErrorView) Hide() {
	fyne.Do(func() {
		v.messageLabel.Hide()
	})
}
