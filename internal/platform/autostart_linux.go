//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (service *platformService) EnableAutostart(appName, execPath string) error {
	if appName == "" {
		return fmt.Errorf("enable autostart: app name is empty")
	}
	if execPath == "" {
		return fmt.Errorf("enable autostart: exec path is empty")
	}

	configDir, err := service.GetConfigDir()
	if err != nil {
		return fmt.Errorf("enable autostart: %w", err)
	}

	autostartDir := filepath.Join(configDir, "autostart")
	if err := os.MkdirAll(autostartDir, 0o755); err != nil {
		return fmt.Errorf("enable autostart: create autostart dir: %w", err)
	}

	desktopFilePath := filepath.Join(autostartDir, desktopFileName(appName))
	if err := os.WriteFile(desktopFilePath, []byte(buildDesktopEntry(appName, execPath)), 0o644); err != nil {
		return fmt.Errorf("enable autostart: write desktop entry: %w", err)
	}

	return nil
}

func (service *platformService) DisableAutostart(appName string) error {
	if appName == "" {
		return fmt.Errorf("disable autostart: app name is empty")
	}

	configDir, err := service.GetConfigDir()
	if err != nil {
		return fmt.Errorf("disable autostart: %w", err)
	}

	desktopFilePath := filepath.Join(configDir, "autostart", desktopFileName(appName))
	if err := os.Remove(desktopFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("disable autostart: remove desktop entry: %w", err)
	}

	return nil
}

func fallbackConfigDir(homeDir string) string {
	return filepath.Join(homeDir, ".config")
}

func desktopFileName(appName string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "eagleeye"
	}
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	return name + ".desktop"
}

func buildDesktopEntry(appName, execPath string) string {
	execLine := execPath
	if strings.Contains(execLine, " ") && !strings.HasPrefix(execLine, `"`) {
		execLine = `"` + execLine + `"`
	}

	return fmt.Sprintf(
		`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
X-GNOME-Autostart-enabled=true
Terminal=false
`,
		appName,
		execLine,
	)
}
