package preferences

import (
	"eagleeye/internal/ui/i18n"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const (
	scheduleLabelWidthEN = float32(190)
	scheduleLabelExtraRU = prefsWindowWidth * 0.2 // ~20% of fixed preferences width
	valueEntryWidth      = float32(60)
)

func makeScheduleRow(label *widget.Label, labelWidth float32, entry *widget.Entry, entryWidth float32, unit *widget.Label) fyne.CanvasObject {
	labelObject := container.NewGridWrap(fyne.NewSize(labelWidth, entry.MinSize().Height), label)
	entryObject := container.NewGridWrap(fyne.NewSize(entryWidth, entry.MinSize().Height), entry)
	return container.NewHBox(labelObject, entryObject, unit)
}

func scheduleLabelWidthForLang(lang string) float32 {
	if i18n.NormalizeLanguage(lang) == i18n.LanguageRU {
		return scheduleLabelWidthEN + scheduleLabelExtraRU
	}
	return scheduleLabelWidthEN
}

func buildScheduleRowGroup(
	scheduleLabels map[string]*widget.Label,
	labels map[string]*widget.Label,
	labelWidth float32,
	shortInt, shortDur, longInt, longDur *widget.Entry,
) []fyne.CanvasObject {
	return []fyne.CanvasObject{
		makeScheduleRow(scheduleLabels["shortInterval"], labelWidth, shortInt, valueEntryWidth, labels["shortInterval"]),
		makeScheduleRow(scheduleLabels["shortDuration"], labelWidth, shortDur, valueEntryWidth, labels["shortDuration"]),
		makeScheduleRow(scheduleLabels["longInterval"], labelWidth, longInt, valueEntryWidth, labels["longInterval"]),
		makeScheduleRow(scheduleLabels["longDuration"], labelWidth, longDur, valueEntryWidth, labels["longDuration"]),
	}
}

func (prefs *Window) refreshScheduleLayoutIfNeeded() {
	lang := i18n.NormalizeLanguage(prefs.uiLocalizer.Language())
	if lang == prefs.scheduleLayoutLang {
		return
	}
	labelWidth := scheduleLabelWidthForLang(lang)
	prefs.scheduleSection.Objects = buildScheduleRowGroup(
		prefs.scheduleLabels,
		prefs.labels,
		labelWidth,
		prefs.shortInt,
		prefs.shortDur,
		prefs.longInt,
		prefs.longDur,
	)
	prefs.scheduleSection.Refresh()
	prefs.scheduleLayoutLang = lang
}
