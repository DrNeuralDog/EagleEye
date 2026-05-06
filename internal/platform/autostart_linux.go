//go:build linux

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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
	if err := os.MkdirAll(autostartDir, 0o700); err != nil {
		return fmt.Errorf("enable autostart: create autostart dir: %w", err)
	}

	if err := os.Chmod(autostartDir, 0o700); err != nil {
		return fmt.Errorf("enable autostart: secure autostart dir permissions: %w", err)
	}

	desktopFilePath := filepath.Join(autostartDir, desktopFileName(appName))
	content, err := buildDesktopEntry(appName, execPath)

	if err != nil {
		return fmt.Errorf("enable autostart: build desktop entry: %w", err)
	}

	if err := os.WriteFile(desktopFilePath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("enable autostart: write desktop entry: %w", err)
	}

	if err := os.Chmod(desktopFilePath, 0o600); err != nil {
		return fmt.Errorf("enable autostart: secure desktop entry permissions: %w", err)
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

func buildDesktopEntry(appName, execPath string) (string, error) {
	if err := validateDesktopEntryValue("app name", appName); err != nil {
		return "", err
	}

	if err := validateDesktopEntryValue("exec path", execPath); err != nil {
		return "", err
	}

	return fmt.Sprintf(
		`[Desktop Entry]
		  Type=Application
		  Name=%s
		  Exec=%s
		  X-GNOME-Autostart-enabled=true
		  Terminal=false
`,
		escapeDesktopString(appName),
		buildDesktopExec(execPath),
	), nil
}

func buildDesktopExec(execPath string) string {
	return quoteDesktopExecArg(execPath) + " " + AutostartArg
}

func validateDesktopEntryValue(field string, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is empty", field)
	}

	for _, char := range value {
		if unicode.IsControl(char) {
			return fmt.Errorf("%s contains control character", field)
		}
	}

	return nil
}

func escapeDesktopString(value string) string {
	return strings.ReplaceAll(value, `\`, `\\`)
}

func quoteDesktopExecArg(value string) string {
	if !strings.ContainsAny(value, " \t\"\\$`%") {
		return value
	}

	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
		`%`, `%%`,
	)

	return `"` + replacer.Replace(value) + `"`
}
