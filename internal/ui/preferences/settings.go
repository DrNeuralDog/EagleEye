package preferences

import (
	"time"

	"eagleeye/internal/core/model"
)

// Settings defines editable user preferences.
type Settings struct {
	ShortInterval time.Duration
	ShortDuration time.Duration
	LongInterval  time.Duration
	LongDuration  time.Duration
	StrictMode    bool
	IdleEnabled   bool

	OverlayOpacity float64
	Fullscreen     bool
}

// DefaultSettings returns default settings for EagleEye.
func DefaultSettings() Settings {
	return Settings{
		ShortInterval: 15 * time.Minute,
		ShortDuration: 15 * time.Second,
		LongInterval:  50 * time.Minute,
		LongDuration:  5 * time.Minute,
		StrictMode:    false,
		IdleEnabled:   true,
		OverlayOpacity: 0.85,
		Fullscreen:     true,
	}
}

// TimeKeeperConfig converts settings to TimeKeeperConfig.
func (settings Settings) TimeKeeperConfig() model.TimeKeeperConfig {
	return model.TimeKeeperConfig{
		Short: model.BreakConfig{
			Interval: settings.ShortInterval,
			Duration: settings.ShortDuration,
			Enabled:  true,
		},
		Long: model.LongBreakConfig{
			BreakConfig: model.BreakConfig{
				Interval: settings.LongInterval,
				Duration: settings.LongDuration,
				Enabled:  true,
			},
			StrictMode: settings.StrictMode,
		},
		IdleResetEnabled:  settings.IdleEnabled,
		IdleResetAfter:    5 * time.Minute,
		IdleCheckInterval: 5 * time.Second,
	}
}
