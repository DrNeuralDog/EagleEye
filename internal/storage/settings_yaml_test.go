package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"eagleeye/internal/ui/preferences"
)

func TestSaveLoadRunOnStartup(t *testing.T) {
	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)

	for _, expected := range []bool{true, false} {
		expected := expected
		t.Run(strings.ToLower("run_on_startup_"+boolString(expected)), func(t *testing.T) {
			appName := "EagleEyeSaveLoad" + boolString(expected)
			settings := preferences.DefaultSettings()
			settings.RunOnStartup = expected

			if err := SaveSettings(appName, settings); err != nil {
				t.Fatalf("SaveSettings() error = %v", err)
			}

			loaded, err := LoadSettings(appName)
			if err != nil {
				t.Fatalf("LoadSettings() error = %v", err)
			}

			if loaded.RunOnStartup != expected {
				t.Fatalf("loaded RunOnStartup = %t, want %t", loaded.RunOnStartup, expected)
			}
		})
	}
}

func TestLoadSettingsLegacyResetWithoutRunOnStartup(t *testing.T) {
	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)

	appName := "EagleEyeLegacyReset"
	configPath, err := resolveConfigPath(appName)
	if err != nil {
		t.Fatalf("resolveConfigPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	legacy := []byte(strings.Join([]string{
		"short_interval_minutes: 1",
		"short_duration_seconds: 2",
		"long_interval_minutes: 3",
		"long_duration_minutes: 4",
		"strict_mode: true",
		"idle_enabled: false",
		"overlay_opacity: 0.95",
		"fullscreen: true",
		"language: ru",
		"",
	}, "\n"))
	if err := os.WriteFile(configPath, legacy, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := LoadSettings(appName)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	defaults := preferences.DefaultSettings()
	if loaded != defaults {
		t.Fatalf("legacy load must reset to defaults; got %+v want %+v", loaded, defaults)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(raw), "run_on_startup: true") {
		t.Fatalf("reset config must contain run_on_startup: true, got:\n%s", string(raw))
	}
}

func setUserConfigEnv(t *testing.T, path string) {
	t.Helper()

	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", path)
	default:
		t.Setenv("XDG_CONFIG_HOME", path)
	}
}

func boolString(value bool) string {
	if value {
		return "True"
	}
	return "False"
}
