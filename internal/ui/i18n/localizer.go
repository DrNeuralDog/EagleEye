package i18n

import (
	"fmt"
	"strings"
	"sync"
)

const (
	LanguageEN = "en"
	LanguageRU = "ru"
)

var translations = map[string]map[string]string{
	LanguageEN: {
		"main.trayWindowMessage":         "EagleEye is running in the system tray.",
		"prefs.windowTitle":              "EagleEye Settings",
		"prefs.headingGeneral":           "General",
		"prefs.shortBreakEvery":          "Short break every",
		"prefs.shortBreakDuration":       "Short break duration",
		"prefs.longBreakEvery":           "Long break every",
		"prefs.longBreakDuration":        "Long break duration",
		"prefs.strictMode":               "Strict mode (disable skip)",
		"prefs.idleTracking":             "Enable idle tracking",
		"prefs.fullscreenOverlay":        "Fullscreen overlay",
		"prefs.runOnStartup":             "Run on startup",
		"prefs.language":                 "Language",
		"prefs.overlayOpacity":           "Overlay opacity:",
		"prefs.autostartApplyErrorTitle": "Autostart Update Failed",
		"prefs.autostartApplyErrorBody":  "Could not apply run on startup setting: %v",
		"prefs.save":                     "Save",
		"prefs.cancel":                   "Cancel",
		"prefs.start":                    "Start",
		"prefs.pauseBreakTimer":          "Pause break timer",
		"prefs.resumeBreakTimer":         "Resume break timer",
		"prefs.serviceNotStartedLine":    "Service not started",
		"prefs.pressStartLine":           "Press Start to run",
		"prefs.serviceRunningLine":       "Service is running",
		"prefs.nextBreakLine":            "Next eye break in",
		"prefs.servicePausedLine":        "Service is paused",
		"prefs.pressResumeLine":          "Press Resume break timer",
		"unit.min":                       "min",
		"unit.sec":                       "sec",
		"tray.menuTitle":                 "EagleEye",
		"tray.statusStarting":            "starting...",
		"tray.statusFormat":              "Status: %s",
		"tray.preferences":               "Preferences",
		"tray.disableBreaksFor":          "Disable breaks for...",
		"tray.pauseForMinutes":           "%d minutes",
		"tray.takeLongBreakNow":          "Take a long break now",
		"tray.pause":                     "Pause",
		"tray.resume":                    "Resume",
		"tray.skipBreak":                 "Skip break",
		"tray.quit":                      "Quit",
		"tray.pausedSuffix":              "(paused)",
		"tray.nextBreakIn":               "next break in %s",
		"overlay.title":                  "Eagle Eye",
		"overlay.subtitle":               "Time to rest your eyes!",
		"overlay.skip":                   "Skip",
		"overlay.exercise.leftRight":     "Move your eyes left and right",
		"overlay.exercise.upDown":        "Move your eyes up and down",
		"overlay.exercise.blink":         "Squint and open your eyes again",
		"overlay.exercise.lookOut":       "Look into the distance and relax",
	},
	LanguageRU: {
		"main.trayWindowMessage":         "EagleEye запущен в системном трее.",
		"prefs.windowTitle":              "Настройки EagleEye",
		"prefs.headingGeneral":           "General",
		"prefs.shortBreakEvery":          "Короткий перерыв каждые",
		"prefs.shortBreakDuration":       "Длительность короткого перерыва",
		"prefs.longBreakEvery":           "Длинный перерыв каждые",
		"prefs.longBreakDuration":        "Длительность длинного перерыва",
		"prefs.strictMode":               "Строгий режим (без пропуска)",
		"prefs.idleTracking":             "Включить отслеживание бездействия",
		"prefs.fullscreenOverlay":        "Полноэкранный оверлей",
		"prefs.runOnStartup":             "Запускать при входе в систему",
		"prefs.language":                 "Язык",
		"prefs.overlayOpacity":           "Непрозрачность оверлея:",
		"prefs.autostartApplyErrorTitle": "Не удалось обновить автозапуск",
		"prefs.autostartApplyErrorBody":  "Не удалось применить настройку автозапуска: %v",
		"prefs.save":                     "Сохранить",
		"prefs.cancel":                   "Отмена",
		"prefs.start":                    "Старт",
		"prefs.pauseBreakTimer":          "Пауза таймера перерывов",
		"prefs.resumeBreakTimer":         "Возобновить таймер перерывов",
		"prefs.serviceNotStartedLine":    "Сервис не запущен",
		"prefs.pressStartLine":           "Нажмите Старт для запуска",
		"prefs.serviceRunningLine":       "Сервис запущен",
		"prefs.nextBreakLine":            "Следующий перерыв через",
		"prefs.servicePausedLine":        "Сервис на паузе",
		"prefs.pressResumeLine":          "Нажмите Возобновить таймер",
		"unit.min":                       "мин",
		"unit.sec":                       "сек",
		"tray.menuTitle":                 "EagleEye",
		"tray.statusStarting":            "запуск...",
		"tray.statusFormat":              "Статус: %s",
		"tray.preferences":               "Настройки",
		"tray.disableBreaksFor":          "Отключить перерывы на...",
		"tray.pauseForMinutes":           "%d минут",
		"tray.takeLongBreakNow":          "Начать длинный перерыв сейчас",
		"tray.pause":                     "Пауза",
		"tray.resume":                    "Продолжить",
		"tray.skipBreak":                 "Пропустить перерыв",
		"tray.quit":                      "Выход",
		"tray.pausedSuffix":              "(пауза)",
		"tray.nextBreakIn":               "следующий перерыв через %s",
		"overlay.title":                  "Eagle Eye",
		"overlay.subtitle":               "Пора отдыхать",
		"overlay.skip":                   "Пропустить",
		"overlay.exercise.leftRight":     "Двигайте глазами влево и вправо",
		"overlay.exercise.upDown":        "Двигайте глазами вверх и вниз",
		"overlay.exercise.blink":         "Зажмурьтесь и откройте глаза вновь",
		"overlay.exercise.lookOut":       "Посмотрите вдаль и расслабьте глаза",
	},
}

type Localizer struct {
	mu       sync.RWMutex
	language string
}

func New(language string) *Localizer {
	return &Localizer{language: NormalizeLanguage(language)}
}

func NormalizeLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case LanguageRU:
		return LanguageRU
	default:
		return LanguageEN
	}
}

func (localizer *Localizer) SetLanguage(language string) {
	localizer.mu.Lock()
	defer localizer.mu.Unlock()
	localizer.language = NormalizeLanguage(language)
}

func (localizer *Localizer) Language() string {
	localizer.mu.RLock()
	defer localizer.mu.RUnlock()
	return localizer.language
}

func (localizer *Localizer) T(key string, args ...any) string {
	localizer.mu.RLock()
	language := localizer.language
	localizer.mu.RUnlock()

	bundle, ok := translations[language]
	if !ok {
		bundle = translations[LanguageEN]
	}
	value, ok := bundle[key]
	if !ok {
		value = translations[LanguageEN][key]
	}
	if len(args) == 0 {
		return value
	}
	return fmt.Sprintf(value, args...)
}

func LanguageOptions() []string {
	return []string{"English", "Русский"}
}

func LanguageDisplayName(language string) string {
	if NormalizeLanguage(language) == LanguageRU {
		return "Русский"
	}
	return "English"
}

func LanguageFromDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "russian", "русский":
		return LanguageRU
	default:
		return LanguageEN
	}
}
