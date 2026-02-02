//go:build windows

package platform

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const registryRunKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

func (service *platformService) EnableAutostart(appName, execPath string) error {
	if appName == "" {
		return fmt.Errorf("enable autostart: app name is empty")
	}
	if execPath == "" {
		return fmt.Errorf("enable autostart: exec path is empty")
	}

	quotedPath := quoteWindowsPath(execPath)
	command := exec.Command(
		"reg",
		"add",
		registryRunKey,
		"/v",
		appName,
		"/t",
		"REG_SZ",
		"/d",
		quotedPath,
		"/f",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable autostart: reg add failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func (service *platformService) DisableAutostart(appName string) error {
	if appName == "" {
		return fmt.Errorf("disable autostart: app name is empty")
	}

	command := exec.Command(
		"reg",
		"delete",
		registryRunKey,
		"/v",
		appName,
		"/f",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("disable autostart: reg delete failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

func fallbackConfigDir(homeDir string) string {
	return filepath.Join(homeDir, "AppData", "Roaming")
}

func quoteWindowsPath(execPath string) string {
	trimmed := strings.Trim(execPath, `"`)
	return fmt.Sprintf(`"%s"`, trimmed)
}
