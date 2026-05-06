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
		"prefs.strictMode":               "Strict mode (disable skip, blocks all screens)",
		"prefs.idleTracking":             "Enable idle tracking",
		"prefs.idleTrackingHelp":         "If you are away for 5+ minutes,\nEagleEye treats that as eye rest\nand restarts the break countdown.\nChecked every 20 seconds.",
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
		"tray.takeNextBreakNow":          "Start next break now",
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
		"prefs.strictMode":               "Строгий режим (без пропуска, блокирует все экраны)",
		"prefs.idleTracking":             "Включить отслеживание бездействия",
		"prefs.idleTrackingHelp":         "Если ты отошел от компьютера\nна 5+ минут, EagleEye считает,\nчто глаза уже отдохнули,\nи запускает таймер заново.\nПроверка идет раз в 20 секунд.",
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
		"tray.takeNextBreakNow":          "\u041d\u0430\u0447\u0430\u0442\u044c \u0441\u043b\u0435\u0434\u0443\u044e\u0449\u0443\u044e \u0440\u0430\u0437\u043c\u0438\u043d\u043a\u0443 \u0441\u0435\u0439\u0447\u0430\u0441",
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

// New creates a localizer with a normalized starting language
func New(language string) *Localizer {
	return &Localizer{language: NormalizeLanguage(language)}
}

// NormalizeLanguage maps user input to a supported language code
func NormalizeLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case LanguageRU:
		return LanguageRU
	default:
		return LanguageEN
	}
}

// SetLanguage updates the active language safely
func (localizer *Localizer) SetLanguage(language string) {
	localizer.mu.Lock()
	defer localizer.mu.Unlock()

	localizer.language = NormalizeLanguage(language)
}

// Language returns the active language code
func (localizer *Localizer) Language() string {
	localizer.mu.RLock()
	defer localizer.mu.RUnlock()

	return localizer.language
}

// T resolves a translation key and formats optional arguments
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

// LanguageOptions returns display labels for the preferences selector
func LanguageOptions() []string {
	return []string{"English", "Русский"}
}

// LanguageDisplayName returns the selector label for a language code
func LanguageDisplayName(language string) string {
	if NormalizeLanguage(language) == LanguageRU {
		return "Русский"
	}

	return "English"
}

// LanguageFromDisplayName converts a selector label into a language code
func LanguageFromDisplayName(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "russian", "русский":
		return LanguageRU
	default:
		return LanguageEN
	}
}
