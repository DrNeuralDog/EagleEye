package preferences

import (
	"eagleeye/internal/ui/i18n"
	"fmt"
	"strconv"
	"time"
)

// UpdateSettings replaces window values.
func (prefs *Window) UpdateSettings(settings Settings) {
	prefs.settings = settings
	prefs.uiLocalizer.SetLanguage(settings.Language)
	prefs.shortInt.SetText(fmt.Sprintf("%d", int(settings.ShortInterval.Minutes())))
	prefs.shortDur.SetText(fmt.Sprintf("%d", int(settings.ShortDuration.Seconds())))
	prefs.longInt.SetText(fmt.Sprintf("%d", int(settings.LongInterval.Minutes())))
	prefs.longDur.SetText(fmt.Sprintf("%d", int(settings.LongDuration.Minutes())))
	prefs.strict.SetChecked(settings.StrictMode)
	prefs.idleCheck.SetChecked(settings.IdleEnabled)
	prefs.opacity.Value = settings.OverlayOpacity
	prefs.opacity.Refresh()
	prefs.fullscreen.SetChecked(settings.Fullscreen)
	prefs.runOnStartup.SetChecked(settings.RunOnStartup)
	prefs.languageSelect.SetSelected(i18n.LanguageDisplayName(settings.Language))
	prefs.RefreshLocalization()
}

func (prefs *Window) handleSave() {
	settings := prefs.settings

	if minutes, ok := parsePositiveInt(prefs.shortInt.Text); ok {
		settings.ShortInterval = time.Duration(minutes) * time.Minute
	}
	if seconds, ok := parsePositiveInt(prefs.shortDur.Text); ok {
		settings.ShortDuration = time.Duration(seconds) * time.Second
	}
	if minutes, ok := parsePositiveInt(prefs.longInt.Text); ok {
		settings.LongInterval = time.Duration(minutes) * time.Minute
	}
	if minutes, ok := parsePositiveInt(prefs.longDur.Text); ok {
		settings.LongDuration = time.Duration(minutes) * time.Minute
	}

	settings.StrictMode = prefs.strict.Checked
	settings.IdleEnabled = prefs.idleCheck.Checked
	settings.OverlayOpacity = prefs.opacity.Value
	settings.Fullscreen = prefs.fullscreen.Checked
	settings.RunOnStartup = prefs.runOnStartup.Checked
	settings.Language = i18n.LanguageFromDisplayName(prefs.languageSelect.Selected)

	prefs.settings = settings
	if prefs.callbacks.OnSave != nil {
		prefs.callbacks.OnSave(settings)
	}
}

func parsePositiveInt(value string) (int, bool) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, false
	}
	return parsed, true
}

func (prefs *Window) dismiss(saved bool) {
	prefs.window.Hide()
	if !saved {
		prefs.uiLocalizer.SetLanguage(prefs.settings.Language)
		prefs.languageSelect.SetSelected(i18n.LanguageDisplayName(prefs.settings.Language))
		prefs.RefreshLocalization()
	}
	if !saved && prefs.callbacks.OnCancel != nil {
		prefs.callbacks.OnCancel()
	}
	if prefs.callbacks.OnDismiss != nil {
		prefs.callbacks.OnDismiss()
	}
}
