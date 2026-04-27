package preferences

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type serviceState int

const (
	serviceStateNotStarted serviceState = iota
	serviceStateRunning
	serviceStatePaused
)

const (
	statusBarMainTextSize  = float32(11)
	statusTimerTextSize    = float32(12) // slightly larger than main status line
	statusIndicatorSize    = float32(18)
	statusBarFromFooterGap = float32(2) // space between timer button and divider line
)

var (
	statusHintColor     = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
	statusTimerColor    = color.NRGBA{R: 252, G: 252, B: 252, A: 255}
	statusBarDividerClr = color.NRGBA{R: 72, G: 72, B: 72, A: 255}
)

func newStatusBar() (fyne.CanvasObject, *canvas.Circle, *canvas.Text, *canvas.Text) {
	statusDot := canvas.NewCircle(color.NRGBA{R: 128, G: 128, B: 128, A: 255})
	statusDot.StrokeWidth = 0
	statusIndicatorWrap := container.NewGridWrap(fyne.NewSize(statusIndicatorSize, statusIndicatorSize), statusDot)
	statusBarMain := canvas.NewText("", statusHintColor)
	statusBarMain.TextSize = statusBarMainTextSize
	statusBarMain.Alignment = fyne.TextAlignLeading
	statusBarTimer := canvas.NewText("", statusTimerColor)
	statusBarTimer.TextSize = statusTimerTextSize
	statusBarTimer.Alignment = fyne.TextAlignLeading
	statusTextLine := container.NewHBox(statusBarMain, statusBarTimer)
	statusTextArea := container.NewStack(statusTextLine)
	indicatorCell := container.NewVBox(layout.NewSpacer(), statusIndicatorWrap, layout.NewSpacer())
	// Center = full-width text (leading); right = indicator pinned to window/content right edge.
	statusRow := container.NewBorder(nil, nil, nil, indicatorCell, statusTextArea)
	statusDivider := canvas.NewRectangle(statusBarDividerClr)
	statusDivider.SetMinSize(fyne.NewSize(1, 1))
	statusBar := container.NewBorder(statusDivider, nil, nil, nil, container.NewPadded(statusRow))
	return statusBar, statusDot, statusBarMain, statusBarTimer
}

// SetServiceNotStarted shows non-running service status.
func (prefs *Window) SetServiceNotStarted() {
	fyne.Do(func() {
		prefs.currentServiceState = serviceStateNotStarted
		prefs.runningTimerText = ""
		prefs.timerToggleButton.Enable()
		prefs.timerToggleButton.Importance = widget.SuccessImportance
		prefs.timerToggleButton.Refresh()
		prefs.renderServiceStatus()
	})
}

// SetServiceRunning shows running status with countdown.
func (prefs *Window) SetServiceRunning(remaining time.Duration) {
	fyne.Do(func() {
		prefs.currentServiceState = serviceStateRunning
		prefs.runningTimerText = formatDuration(remaining)
		prefs.timerToggleButton.Enable()
		prefs.timerToggleButton.Importance = widget.MediumImportance
		prefs.timerToggleButton.Refresh()
		prefs.renderServiceStatus()
	})
}

// SetServicePaused shows paused service status.
func (prefs *Window) SetServicePaused() {
	fyne.Do(func() {
		prefs.currentServiceState = serviceStatePaused
		prefs.runningTimerText = ""
		prefs.timerToggleButton.Enable()
		prefs.timerToggleButton.Importance = widget.MediumImportance
		prefs.timerToggleButton.Refresh()
		prefs.renderServiceStatus()
	})
}

// SetTimerControlState updates the bottom button label.
func (prefs *Window) SetTimerControlState(isRunning bool) {
	fyne.Do(func() {
		if prefs.currentServiceState == serviceStateNotStarted {
			prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.start"))
			return
		}
		if isRunning {
			prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.pauseBreakTimer"))
		} else {
			prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.resumeBreakTimer"))
		}
	})
}

func (prefs *Window) renderServiceStatus() {
	switch prefs.currentServiceState {
	case serviceStateRunning:
		prefs.statusIndicatorDot.FillColor = color.NRGBA{R: 57, G: 176, B: 99, A: 255}
		prefs.statusBarMain.Text = fmt.Sprintf("%s вЂ” %s ",
			prefs.uiLocalizer.T("prefs.serviceRunningLine"),
			prefs.uiLocalizer.T("prefs.nextBreakLine"),
		)
		prefs.statusBarTimer.Text = prefs.runningTimerText
		prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.pauseBreakTimer"))
	case serviceStatePaused:
		prefs.statusIndicatorDot.FillColor = color.NRGBA{R: 232, G: 190, B: 66, A: 255}
		prefs.statusBarMain.Text = fmt.Sprintf("%s вЂ” %s",
			prefs.uiLocalizer.T("prefs.servicePausedLine"),
			prefs.uiLocalizer.T("prefs.pressResumeLine"),
		)
		prefs.statusBarTimer.Text = ""
		prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.resumeBreakTimer"))
	default:
		prefs.statusIndicatorDot.FillColor = color.NRGBA{R: 128, G: 128, B: 128, A: 255}
		prefs.statusBarMain.Text = fmt.Sprintf("%s вЂ” %s",
			prefs.uiLocalizer.T("prefs.serviceNotStartedLine"),
			prefs.uiLocalizer.T("prefs.pressStartLine"),
		)
		prefs.statusBarTimer.Text = ""
		prefs.timerToggleButton.SetText(prefs.uiLocalizer.T("prefs.start"))
	}

	prefs.statusIndicatorDot.Refresh()
	prefs.statusBarMain.Refresh()
	prefs.statusBarTimer.Refresh()
}

func formatDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	seconds := int(value.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
