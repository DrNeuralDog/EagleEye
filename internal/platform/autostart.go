package platform

import (
	"fmt"
	"os"
)

// Service defines OS-specific helpers needed by the application.
type Service interface {
	GetConfigDir() (string, error)
	EnableAutostart(appName, execPath string) error
	DisableAutostart(appName string) error
}

type platformService struct{}

// NewService returns a platform-specific implementation.
func NewService() Service {
	return &platformService{}
}

// GetConfigDir returns the OS-standard configuration directory.
func (service *platformService) GetConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return configDir, nil
	}

	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		if err != nil {
			return "", fmt.Errorf("get config dir: %w", err)
		}
		return "", fmt.Errorf("get config dir: %w", homeErr)
	}

	return fallbackConfigDir(homeDir), nil
}
