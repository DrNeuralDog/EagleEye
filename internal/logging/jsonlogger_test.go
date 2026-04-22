package logging

import (
	"log/slog"
	"testing"
)

func TestLevelFromEnv(t *testing.T) {
	tests := []struct {
		value string
		want  slog.Level
	}{
		{value: "", want: slog.LevelInfo},
		{value: "debug", want: slog.LevelDebug},
		{value: "warn", want: slog.LevelWarn},
		{value: "warning", want: slog.LevelWarn},
		{value: "error", want: slog.LevelError},
		{value: "unknown", want: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Setenv("EAGLEEYE_LOG_LEVEL", tt.value)
			if got := LevelFromEnv(); got != tt.want {
				t.Fatalf("LevelFromEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
