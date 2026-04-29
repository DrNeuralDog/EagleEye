package storage

import (
	"bytes"
	"eagleeye/internal/ui/preferences"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSaveLoadRunOnStartup verifies both true and false autostart values survive YAML roundtrip
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

// TestSaveLoadBreakTimerStarted verifies persisted timer state is restored on load
func TestSaveLoadBreakTimerStarted(t *testing.T) {
	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)

	settings := preferences.DefaultSettings()
	settings.BreakTimerStarted = true

	if err := SaveSettings("EagleEyeBreakTimerStarted", settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	loaded, err := LoadSettings("EagleEyeBreakTimerStarted")
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if !loaded.BreakTimerStarted {
		t.Fatalf("loaded BreakTimerStarted = false, want true")
	}
}

// TestLoadSettingsWithoutBreakTimerStartedKeepsFalse covers configs created before timer state existed
func TestLoadSettingsWithoutBreakTimerStartedKeepsFalse(t *testing.T) {
	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)

	appName := "EagleEyeMissingBreakTimerStarted"
	configPath, err := resolveConfigPath(appName)

	if err != nil {
		t.Fatalf("resolveConfigPath() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	raw := []byte(strings.Join([]string{
		"short_interval_minutes: 15",
		"short_duration_seconds: 15",
		"long_interval_minutes: 50",
		"long_duration_minutes: 5",
		"strict_mode: false",
		"idle_enabled: true",
		"overlay_opacity: 0.85",
		"fullscreen: false",
		"run_on_startup: true",
		"language: en",
		"",
	}, "\n"))

	if err := os.WriteFile(configPath, raw, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loaded, err := LoadSettings(appName)
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if loaded.BreakTimerStarted {
		t.Fatalf("loaded BreakTimerStarted = true, want false")
	}
}

// TestSaveSettingsUsesPrivateFileMode verifies saved settings are not world-readable
func TestSaveSettingsUsesPrivateFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs are not represented reliably through os.FileMode permission bits")
	}

	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)

	appName := "EagleEyePrivateMode"
	if err := SaveSettings(appName, preferences.DefaultSettings()); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	configPath, err := resolveConfigPath(appName)
	if err != nil {
		t.Fatalf("resolveConfigPath() error = %v", err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if info.Mode().Perm() != 0o600 {
		t.Fatalf("settings mode = %o, want 0600", info.Mode().Perm())
	}
}

// TestLoadSettingsRejectsOversizedFile guards against loading unexpectedly large configs
func TestLoadSettingsRejectsOversizedFile(t *testing.T) {
	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)

	appName := "EagleEyeOversized"
	configPath, err := resolveConfigPath(appName)
	if err != nil {
		t.Fatalf("resolveConfigPath() error = %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	oversized := bytes.Repeat([]byte("a"), maxSettingsFileSize+1)
	if err := os.WriteFile(configPath, oversized, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := LoadSettings(appName); err == nil {
		t.Fatalf("LoadSettings() error = nil, want oversized file error")
	} else if !strings.Contains(err.Error(), "settings file exceeds") {
		t.Fatalf("LoadSettings() error = %v, want oversized file error", err)
	}
}

// TestResolveLogPathUsesAppConfigDir verifies logs stay under the app config directory
func TestResolveLogPathUsesAppConfigDir(t *testing.T) {
	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)

	logPath, err := ResolveLogPath("EagleEyeLogPath")
	if err != nil {
		t.Fatalf("ResolveLogPath() error = %v", err)
	}

	wantSuffix := filepath.Join("EagleEyeLogPath", logFileName)
	if !strings.HasSuffix(logPath, wantSuffix) {
		t.Fatalf("ResolveLogPath() = %q, want suffix %q", logPath, wantSuffix)
	}
}

// TestConfigPathEnvOverridesSettingsPathOnly verifies custom settings path does not move logs
func TestConfigPathEnvOverridesSettingsPathOnly(t *testing.T) {
	configRoot := t.TempDir()
	setUserConfigEnv(t, configRoot)
	overridePath := filepath.Join(t.TempDir(), "custom-settings.yaml")
	t.Setenv(configPathEnv, overridePath)

	settings := preferences.DefaultSettings()
	settings.BreakTimerStarted = true

	if err := SaveSettings("EagleEyeEnvOverride", settings); err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}

	loaded, err := LoadSettings("EagleEyeEnvOverride")
	if err != nil {
		t.Fatalf("LoadSettings() error = %v", err)
	}

	if !loaded.BreakTimerStarted {
		t.Fatalf("loaded BreakTimerStarted = false, want true")
	}

	if _, err := os.Stat(overridePath); err != nil {
		t.Fatalf("Stat(overridePath) error = %v", err)
	}

	logPath, err := ResolveLogPath("EagleEyeEnvOverride")
	if err != nil {
		t.Fatalf("ResolveLogPath() error = %v", err)
	}

	if strings.Contains(logPath, filepath.Dir(overridePath)) {
		t.Fatalf("ResolveLogPath() = %q, want app config dir independent of EAGLEEYE_CONFIG_PATH", logPath)
	}
}

// TestLoadSettingsLegacyResetWithoutRunOnStartup verifies old configs are replaced with defaults
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

	if err := os.WriteFile(configPath, legacy, 0o600); err != nil {
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

// setUserConfigEnv isolates user config paths inside each test temp directory
func setUserConfigEnv(t *testing.T, path string) {
	t.Helper()

	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", path)
	default:
		t.Setenv("XDG_CONFIG_HOME", path)
	}
}

// boolString keeps subtest and app names stable for boolean cases
func boolString(value bool) string {
	if value {
		return "True"
	}

	return "False"
}
