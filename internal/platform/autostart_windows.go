//go:build windows

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

const registryRunKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

func (service *platformService) EnableAutostart(appName, execPath string) error {
	if appName == "" {
		return fmt.Errorf("enable autostart: app name is empty")
	}
	if execPath == "" {
		return fmt.Errorf("enable autostart: exec path is empty")
	}
	if err := validateWindowsExecPath(execPath); err != nil {
		return fmt.Errorf("enable autostart: %w", err)
	}
	if err := validateRegistryValueName(appName); err != nil {
		return fmt.Errorf("enable autostart: %w", err)
	}
	regPath, err := regExePath()
	if err != nil {
		return fmt.Errorf("enable autostart: %w", err)
	}

	quotedPath := quoteWindowsPath(execPath)
	command := exec.Command(
		regPath,
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
	if err := validateRegistryValueName(appName); err != nil {
		return fmt.Errorf("disable autostart: %w", err)
	}
	regPath, err := regExePath()
	if err != nil {
		return fmt.Errorf("disable autostart: %w", err)
	}

	command := exec.Command(
		regPath,
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
	return fmt.Sprintf(`"%s"`, execPath)
}

func validateWindowsExecPath(execPath string) error {
	if !filepath.IsAbs(execPath) {
		return fmt.Errorf("executable path must be absolute")
	}
	if strings.Contains(execPath, `"`) {
		return fmt.Errorf("executable path contains quote")
	}
	if strings.Contains(execPath, `%`) {
		return fmt.Errorf("executable path contains environment variable marker")
	}
	if containsControlRune(execPath) {
		return fmt.Errorf("executable path contains control character")
	}
	return nil
}

func validateRegistryValueName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("registry value name is empty")
	}
	if strings.Contains(name, `\`) {
		return fmt.Errorf("registry value name contains backslash")
	}
	if containsControlRune(name) {
		return fmt.Errorf("registry value name contains control character")
	}
	return nil
}

func regExePath() (string, error) {
	systemRoot := os.Getenv("SystemRoot")
	if strings.TrimSpace(systemRoot) == "" {
		systemRoot = `C:\Windows`
	}
	path := filepath.Join(systemRoot, "System32", "reg.exe")
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("resolve reg.exe: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("resolve reg.exe: path is a directory")
	}
	return path, nil
}

func containsControlRune(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) {
			return true
		}
	}
	return false
}
