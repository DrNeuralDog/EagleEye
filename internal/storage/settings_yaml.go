package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"eagleeye/internal/ui/i18n"
	"eagleeye/internal/ui/preferences"
	"gopkg.in/yaml.v3"
)

const settingsFileName = "settings.yaml"
const logFileName = "EagleEye.log.jsonl"
const maxSettingsFileSize = 256 * 1024

type yamlSettings struct {
	ShortIntervalMinutes int     `yaml:"short_interval_minutes"`
	ShortDurationSeconds int     `yaml:"short_duration_seconds"`
	LongIntervalMinutes  int     `yaml:"long_interval_minutes"`
	LongDurationMinutes  int     `yaml:"long_duration_minutes"`
	StrictMode           bool    `yaml:"strict_mode"`
	IdleEnabled          bool    `yaml:"idle_enabled"`
	OverlayOpacity       float64 `yaml:"overlay_opacity"`
	Fullscreen           bool    `yaml:"fullscreen"`
	RunOnStartup         *bool   `yaml:"run_on_startup"`
	Language             string  `yaml:"language"`
}

// LoadSettings reads user preferences from YAML.
// If the config file does not exist, default settings are returned.
func LoadSettings(appName string) (preferences.Settings, error) {
	settings := preferences.DefaultSettings()
	configPath, err := resolveConfigPath(appName)
	if err != nil {
		return settings, err
	}

	stat, err := os.Stat(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return settings, nil
		}
		return settings, fmt.Errorf("stat settings file: %w", err)
	}
	if stat.Size() > maxSettingsFileSize {
		return settings, fmt.Errorf("settings file exceeds %d bytes", maxSettingsFileSize)
	}

	rawData, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return settings, nil
		}
		return settings, fmt.Errorf("read settings file: %w", err)
	}

	var fileData yamlSettings
	if err := yaml.Unmarshal(rawData, &fileData); err != nil {
		return settings, fmt.Errorf("parse settings yaml: %w", err)
	}
	if fileData.RunOnStartup == nil {
		defaultSettings := preferences.DefaultSettings()
		if err := os.Remove(configPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return defaultSettings, fmt.Errorf("reset legacy settings: remove old file: %w", err)
		}
		if err := SaveSettings(appName, defaultSettings); err != nil {
			return defaultSettings, fmt.Errorf("reset legacy settings: save defaults: %w", err)
		}
		return defaultSettings, nil
	}

	applyYamlSettings(&settings, fileData)
	return settings, nil
}

// SaveSettings writes user preferences to YAML.
func SaveSettings(appName string, settings preferences.Settings) error {
	configPath, err := resolveConfigPath(appName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	fileData := yamlSettings{
		ShortIntervalMinutes: int(settings.ShortInterval / time.Minute),
		ShortDurationSeconds: int(settings.ShortDuration / time.Second),
		LongIntervalMinutes:  int(settings.LongInterval / time.Minute),
		LongDurationMinutes:  int(settings.LongDuration / time.Minute),
		StrictMode:           settings.StrictMode,
		IdleEnabled:          settings.IdleEnabled,
		OverlayOpacity:       settings.OverlayOpacity,
		Fullscreen:           settings.Fullscreen,
		RunOnStartup:         boolPointer(settings.RunOnStartup),
		Language:             i18n.NormalizeLanguage(settings.Language),
	}

	serialized, err := yaml.Marshal(fileData)
	if err != nil {
		return fmt.Errorf("marshal settings yaml: %w", err)
	}

	if err := os.WriteFile(configPath, serialized, 0o600); err != nil {
		return fmt.Errorf("write settings file: %w", err)
	}
	if err := os.Chmod(configPath, 0o600); err != nil {
		return fmt.Errorf("secure settings file permissions: %w", err)
	}

	return nil
}

// ResolveLogPath returns the per-user JSONL log path for the application.
func ResolveLogPath(appName string) (string, error) {
	configDir, err := resolveAppConfigDir(appName)
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, logFileName), nil
}

func resolveConfigPath(appName string) (string, error) {
	configDir, err := resolveAppConfigDir(appName)
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, settingsFileName), nil
}

func resolveAppConfigDir(appName string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(configDir, appName), nil
}

func applyYamlSettings(settings *preferences.Settings, fileData yamlSettings) {
	if fileData.ShortIntervalMinutes > 0 {
		settings.ShortInterval = time.Duration(fileData.ShortIntervalMinutes) * time.Minute
	}
	if fileData.ShortDurationSeconds > 0 {
		settings.ShortDuration = time.Duration(fileData.ShortDurationSeconds) * time.Second
	}
	if fileData.LongIntervalMinutes > 0 {
		settings.LongInterval = time.Duration(fileData.LongIntervalMinutes) * time.Minute
	}
	if fileData.LongDurationMinutes > 0 {
		settings.LongDuration = time.Duration(fileData.LongDurationMinutes) * time.Minute
	}

	if fileData.OverlayOpacity >= 0.7 && fileData.OverlayOpacity <= 0.95 {
		settings.OverlayOpacity = fileData.OverlayOpacity
	}

	settings.StrictMode = fileData.StrictMode
	settings.IdleEnabled = fileData.IdleEnabled
	settings.Fullscreen = fileData.Fullscreen
	if fileData.RunOnStartup != nil {
		settings.RunOnStartup = *fileData.RunOnStartup
	}
	settings.Language = i18n.NormalizeLanguage(fileData.Language)
}

func boolPointer(value bool) *bool {
	pointer := value
	return &pointer
}
