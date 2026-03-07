//go:build darwin

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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("enable autostart: get home dir: %w", err)
	}

	launchAgentsDir := filepath.Join(homeDir, "Library", "LaunchAgents")
	if err := os.MkdirAll(launchAgentsDir, 0o700); err != nil {
		return fmt.Errorf("enable autostart: create LaunchAgents dir: %w", err)
	}
	if err := os.Chmod(launchAgentsDir, 0o700); err != nil {
		return fmt.Errorf("enable autostart: secure LaunchAgents dir permissions: %w", err)
	}

	label := launchAgentLabel(appName)
	plistPath := filepath.Join(launchAgentsDir, label+".plist")
	content, err := buildLaunchAgentPlist(label, execPath)
	if err != nil {
		return fmt.Errorf("enable autostart: build plist: %w", err)
	}
	if err := os.WriteFile(plistPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("enable autostart: write plist: %w", err)
	}
	if err := os.Chmod(plistPath, 0o600); err != nil {
		return fmt.Errorf("enable autostart: secure plist permissions: %w", err)
	}

	return nil
}

func (service *platformService) DisableAutostart(appName string) error {
	if appName == "" {
		return fmt.Errorf("disable autostart: app name is empty")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("disable autostart: get home dir: %w", err)
	}

	label := launchAgentLabel(appName)
	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", label+".plist")
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("disable autostart: remove plist: %w", err)
	}

	return nil
}

func fallbackConfigDir(homeDir string) string {
	return filepath.Join(homeDir, "Library", "Application Support")
}

func launchAgentLabel(appName string) string {
	name := strings.TrimSpace(appName)
	if name == "" {
		name = "eagleeye"
	}
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "-")
	return "com.eagleeye." + name
}

func buildLaunchAgentPlist(label, execPath string) (string, error) {
	if err := validateLaunchAgentValue("label", label); err != nil {
		return "", err
	}
	if err := validateLaunchAgentValue("exec path", execPath); err != nil {
		return "", err
	}
	escapedPath := xmlEscape(execPath)
	escapedLabel := xmlEscape(label)
	escapedAutostartArg := xmlEscape(AutostartArg)

	return fmt.Sprintf(
		`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>%s</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`,
		escapedLabel,
		escapedPath,
		escapedAutostartArg,
	), nil
}

func validateLaunchAgentValue(field string, value string) error {
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

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}
