package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ConnectionView struct {
	statuslabel *widget.Label
	content     *fyne.Container
}

func NewConnectionView() *ConnectionView {
	statusLabel := widget.NewLabel("Connecting...")
	statusLabel.Alignment = fyne.TextAlignCenter

	content := container.NewCenter(statusLabel)

	return &ConnectionView{
		statuslabel: statusLabel,
		content:     content,
	}
}

func (v *ConnectionView) UpdateStatus(status string) {
	fyne.Do(func() {
		v.statuslabel.SetText(status)
	})
}

func (v *ConnectionView) Build() fyne.CanvasObject {
	return v.content
}
