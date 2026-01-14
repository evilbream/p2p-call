package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type RendezvousView struct {
	entry     *widget.Entry
	statusLbl *widget.Label
	onConnect func(string)
}

func NewRendezvousView(onConnect func(string)) *RendezvousView {
	return &RendezvousView{
		onConnect: onConnect,
	}
}

func (v *RendezvousView) Build() fyne.CanvasObject {
	v.entry = widget.NewEntry()
	v.entry.SetPlaceHolder("Enter meeting code...")

	v.statusLbl = widget.NewLabel("")
	v.statusLbl.Hide()

	connectBtn := widget.NewButton("Connect", func() {
		code := v.entry.Text
		if code == "" {
			v.statusLbl.SetText("Please enter a meeting code")
			v.statusLbl.Show()
			return
		}
		if v.onConnect != nil {
			v.onConnect(code)
		}
	})

	v.entry.OnSubmitted = func(text string) {
		connectBtn.OnTapped()
	}

	return container.NewCenter(
		container.NewVBox(
			widget.NewLabel("Enter Rendezvous Point"),
			v.entry,
			container.NewGridWithColumns(1, connectBtn),
			v.statusLbl,
		),
	)
}
