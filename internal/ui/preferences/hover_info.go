package preferences

import (
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

const (
	hoverInfoDelay       = time.Second
	hoverInfoPopupWidth  = float32(390)
	hoverInfoPopupHeight = float32(118)
	hoverInfoPopupGap    = float32(4)
)

func newDelayedHoverInfoRow(content fyne.CanvasObject, canvas fyne.Canvas, message func() string, onTap func()) fyne.CanvasObject {
	return container.NewMax(content, newDelayedHoverInfoHotspot(canvas, message, onTap))
}

type delayedHoverInfoHotspot struct {
	widget.BaseWidget

	mu      sync.Mutex
	canvas  fyne.Canvas
	message func() string
	onTap   func()

	hovering bool
	timer    *time.Timer
	popup    *widget.PopUp
}

func newDelayedHoverInfoHotspot(canvas fyne.Canvas, message func() string, onTap func()) *delayedHoverInfoHotspot {
	hotspot := &delayedHoverInfoHotspot{
		canvas:  canvas,
		message: message,
		onTap:   onTap,
	}
	hotspot.ExtendBaseWidget(hotspot)
	return hotspot
}

func (hotspot *delayedHoverInfoHotspot) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(color.Transparent)
	background.SetMinSize(fyne.NewSize(1, 1))
	return widget.NewSimpleRenderer(background)
}

func (hotspot *delayedHoverInfoHotspot) MouseIn(*desktop.MouseEvent) {
	hotspot.mu.Lock()
	defer hotspot.mu.Unlock()

	hotspot.hovering = true
	if hotspot.timer != nil {
		hotspot.timer.Stop()
	}
	hotspot.timer = time.AfterFunc(hoverInfoDelay, func() {
		fyne.Do(hotspot.showIfStillHovering)
	})
}

func (hotspot *delayedHoverInfoHotspot) MouseMoved(*desktop.MouseEvent) {}

func (hotspot *delayedHoverInfoHotspot) MouseOut() {
	hotspot.hidePopup()
}

func (hotspot *delayedHoverInfoHotspot) Tapped(*fyne.PointEvent) {
	hotspot.hidePopup()
	if hotspot.onTap != nil {
		hotspot.onTap()
	}
}

func (hotspot *delayedHoverInfoHotspot) showIfStillHovering() {
	hotspot.mu.Lock()
	if !hotspot.hovering || hotspot.canvas == nil || hotspot.message == nil {
		hotspot.mu.Unlock()
		return
	}
	previous := hotspot.popup
	hotspot.popup = nil
	hotspot.mu.Unlock()

	if previous != nil {
		previous.Hide()
	}

	label := widget.NewLabel(hotspot.message())
	label.Wrapping = fyne.TextWrapWord
	content := container.NewGridWrap(fyne.NewSize(hoverInfoPopupWidth, hoverInfoPopupHeight), label)
	popup := widget.NewPopUp(content, hotspot.canvas)
	popup.ShowAtRelativePosition(fyne.NewPos(0, hotspot.Size().Height+hoverInfoPopupGap), hotspot)

	hotspot.mu.Lock()
	if hotspot.hovering {
		hotspot.popup = popup
		hotspot.mu.Unlock()
		return
	}
	hotspot.mu.Unlock()
	popup.Hide()
}

func (hotspot *delayedHoverInfoHotspot) hidePopup() {
	hotspot.mu.Lock()
	hotspot.hovering = false
	if hotspot.timer != nil {
		hotspot.timer.Stop()
		hotspot.timer = nil
	}
	popup := hotspot.popup
	hotspot.popup = nil
	hotspot.mu.Unlock()

	if popup != nil {
		fyne.Do(func() {
			popup.Hide()
		})
	}
}
