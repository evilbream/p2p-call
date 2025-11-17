package views

import (
	"image/color"
	"p2p-call/internal/audio/capture"
	"p2p-call/internal/audio/playback"
	"p2p-call/internal/rtc"
	"p2p-call/internal/rtc/datachannel"
	"p2p-call/pkg/interface/desktop/viewmodel"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	labelStartCall  = "Start Call"
	labelEndCall    = "End Call"
	labelMute       = "Mute"
	labelUnmute     = "Unmute"
	labelSpeakerOn  = "Speaker"
	labelSpeakerOff = "Speaker Off"
)

type MainView struct {
	content        *fyne.Container
	messages       []viewmodel.ChatMessage
	messagesBox    *fyne.Container
	messagesScroll *container.Scroll
	messageEntry   *widget.Entry
	startCallBtn   *widget.Button
	endCallBtn     *widget.Button
	muteBtn        *widget.Button
	speakerBtn     *widget.Button
	callStatusText *widget.Label
	isCallActive   bool
	isMuted        bool
	isSpeakerOn    bool
	SendMessage    func(text string) error
	// add fields as necessary
}

func NewMainView(capture *capture.MalgoCapture, playback *playback.MalgoPlayback, sendMessage func(text string) error) *MainView {
	if capture == nil || playback == nil {
		return nil
	}
	mv := MainView{messages: []viewmodel.ChatMessage{},
		isCallActive: false,
		isMuted:      capture.Paused,
		isSpeakerOn:  capture.Paused,
		SendMessage:  sendMessage,
	}

	// set message box and scroll
	mv.messagesBox = container.NewVBox()
	mv.messagesScroll = container.NewVScroll(mv.messagesBox)
	mv.messagesScroll.SetMinSize(fyne.NewSize(0, 320))

	// set call status
	mv.callStatusText = widget.NewLabel("")
	mv.callStatusText.TextStyle = fyne.TextStyle{Bold: true}
	mv.callStatusText.Alignment = fyne.TextAlignCenter
	mv.callStatusText.Hide()

	// set call button
	mv.startCallBtn = widget.NewButton(labelStartCall, func() {
		mv.startCall()
	})

	mv.endCallBtn = widget.NewButton(labelEndCall, func() {
		mv.endCall()
	})
	mv.endCallBtn.Hide()

	mv.muteBtn = widget.NewButton(labelMute, func() {
		mv.toggleMute()
	})
	mv.muteBtn.Hide()

	mv.speakerBtn = widget.NewButton(labelSpeakerOn, func() {
		mv.toggleSpeaker()
	})
	mv.speakerBtn.Hide()

	callControls := container.NewGridWithColumns(2,
		mv.startCallBtn,
		mv.endCallBtn,
	)

	audioControls := container.NewGridWithColumns(2,
		mv.muteBtn,
		mv.speakerBtn,
	)

	// Поле ввода сообщения
	mv.messageEntry = widget.NewEntry()
	mv.messageEntry.SetPlaceHolder("Введите сообщение...")
	mv.messageEntry.OnSubmitted = func(text string) {
		mv.sendMessage()
	}

	sendBtn := widget.NewButton("Отправить", func() {
		mv.sendMessage()
	})

	inputArea := container.NewBorder(nil, nil, nil, sendBtn, mv.messageEntry)

	// Добавляем приветственное сообщение

	// Заголовок
	header := canvas.NewText("P2P call", color.Black)
	header.TextSize = 20
	header.Alignment = fyne.TextAlignCenter
	header.TextStyle = fyne.TextStyle{Bold: true}

	headerContainer := container.NewVBox(
		container.NewCenter(header),
		widget.NewSeparator(),
	)

	// Макет чата (появится после подключения)
	chatContent := container.NewBorder(
		container.NewVBox(
			headerContainer,
			callControls,
			audioControls,
			mv.callStatusText,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			inputArea,
		),
		nil,
		nil,
		mv.messagesScroll,
	)
	mv.content = chatContent
	return &mv
}

func (v *MainView) Build() *fyne.Container {
	// build the main view UI components
	return v.content
}

func (v *MainView) sendMessage() {
	text := v.messageEntry.Text
	if text == "" {
		return
	}

	msg := viewmodel.ChatMessage{
		Text:      text,
		Timestamp: time.Now(),
		IsOwn:     true,
	}
	v.messages = append(v.messages, msg)
	v.addMessageToUI(msg)
	v.messageEntry.SetText("")
	if err := v.SendMessage(text); err != nil { // send message via data channel
		log.Error().Err(err).Msgf("Failed to send message")
	}
}

func (ca *MainView) ReceiveMesage(text string) {
	msg := viewmodel.ChatMessage{
		Text:      text,
		Timestamp: time.Now(),
		IsOwn:     false,
	}
	ca.messages = append(ca.messages, msg)
	ca.addMessageToUI(msg)
}

func (ca *MainView) addSystemMessage(text string) {
	msg := viewmodel.ChatMessage{
		Text:      "System: " + text,
		Timestamp: time.Now(),
		IsOwn:     false,
	}
	ca.messages = append(ca.messages, msg)
	ca.addMessageToUI(msg)
}

// addMessageToUI создает виджет для сообщения и добавляет в контейнер
func (ca *MainView) addMessageToUI(msg viewmodel.ChatMessage) {
	prefix := ""
	style := fyne.TextStyle{}
	if msg.IsOwn {
		prefix = "You: "
		style = fyne.TextStyle{Bold: true}
	} else if strings.HasPrefix(msg.Text, "System") {
		style = fyne.TextStyle{Italic: true}
	}
	ts := msg.Timestamp.Format("15:04:05")
	lbl := widget.NewLabel(" [" + ts + "] " + prefix + msg.Text)
	lbl.TextStyle = style
	lbl.Wrapping = fyne.TextWrapWord
	ca.messagesBox.Add(lbl)
	ca.messagesScroll.Refresh()
}

func (ca *MainView) startCall() {
	ca.isCallActive = true
	ca.startCallBtn.Hide()
	ca.endCallBtn.Show()
	ca.muteBtn.Show()
	ca.speakerBtn.Show()
	ca.callStatusText.SetText("Call Active")
	ca.callStatusText.Show()
	ca.addSystemMessage("Call started")
}

func (ca *MainView) endCall() {
	ca.isCallActive = false
	ca.isMuted = false
	ca.isSpeakerOn = true
	ca.startCallBtn.Show()
	ca.endCallBtn.Hide()
	ca.muteBtn.Hide()
	ca.muteBtn.SetText(labelMute)
	ca.speakerBtn.Hide()
	ca.speakerBtn.SetText(labelSpeakerOn)
	ca.callStatusText.Hide()
	ca.addSystemMessage("Call ended")
}

func (ca *MainView) toggleMute() {
	ca.isMuted = !ca.isMuted
	if ca.isMuted {
		ca.muteBtn.SetText(labelUnmute)
		ca.addSystemMessage("Microphone muted")
	} else {
		ca.muteBtn.SetText(labelMute)
		ca.addSystemMessage("Microphone unmuted")
	}
}

func (ca *MainView) toggleSpeaker() {
	ca.isSpeakerOn = !ca.isSpeakerOn
	if ca.isSpeakerOn {
		ca.speakerBtn.SetText(labelSpeakerOn)
		ca.addSystemMessage("Динамик включен")
	} else {
		ca.speakerBtn.SetText(labelSpeakerOff)
		ca.addSystemMessage("Dynamic off")
	}
}

func (v *MainView) SetConnection(conn *rtc.Connection) {

	conn.DcManager.OnMessage = func(msg datachannel.DataChannelMessage) {
		log.Info().
			Str("type", string(msg.Type)).
			Str("text", msg.Payload).
			Msg("ChatViewModel received message") // ← ДОБАВЬТЕ ЛОГ

		if msg.Type == datachannel.MessageTypeText {
			v.ReceiveMesage(msg.Payload)
		}
	}

	log.Info().Msg("ChatViewModel connected to DataChannel")
}
