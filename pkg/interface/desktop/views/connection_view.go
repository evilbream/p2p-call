package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ConnectionView struct {
	statuslabel *widget.Label
	content     *fyne.Container
	cancelBtn   *widget.Button
	onCancel    func()
}

func NewConnectionView() *ConnectionView {
	statusLabel := widget.NewLabel("Connecting...")
	statusLabel.Alignment = fyne.TextAlignCenter

	cancelBtn := widget.NewButton("Cancel", func() {
		// will be set latter
	})

	content := container.NewCenter(
		container.NewVBox(
			widget.NewProgressBarInfinite(),
			statusLabel,
			container.NewPadded(cancelBtn),
		),
	)

	return &ConnectionView{
		statuslabel: statusLabel,
		content:     content,
		cancelBtn:   cancelBtn,
	}
}

func (v *ConnectionView) SetCancelCallback(callback func()) {
	v.onCancel = callback
	if v.cancelBtn != nil {
		v.cancelBtn.OnTapped = v.onCancel
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
