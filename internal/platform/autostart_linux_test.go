//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDesktopEntryEscapesAndQuotesValues(t *testing.T) {
	entry, err := buildDesktopEntry("EagleEye", `/opt/Eagle Eye/eagle$eye%icon`)
	if err != nil {
		t.Fatalf("buildDesktopEntry() error = %v", err)
	}
	if !strings.Contains(entry, `Name=EagleEye`) {
		t.Fatalf("desktop entry missing escaped name:\n%s", entry)
	}
	if !strings.Contains(entry, `Exec="/opt/Eagle Eye/eagle\$eye%%icon" --autostart`) {
		t.Fatalf("desktop entry missing quoted exec path:\n%s", entry)
	}
}

func TestBuildDesktopEntryRejectsControlCharacters(t *testing.T) {
	if _, err := buildDesktopEntry("Eagle\nEye", "/usr/bin/eagleeye"); err == nil {
		t.Fatalf("buildDesktopEntry() error = nil, want app name control character error")
	}
	if _, err := buildDesktopEntry("EagleEye", "/usr/bin/eagleeye\n--bad"); err == nil {
		t.Fatalf("buildDesktopEntry() error = nil, want exec path control character error")
	}
}

func TestEnableAutostartUsesPrivateFileModes(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)

	service := NewService()
	if err := service.EnableAutostart("EagleEye", "/usr/bin/eagleeye"); err != nil {
		t.Fatalf("EnableAutostart() error = %v", err)
	}

	autostartDir := filepath.Join(configRoot, "autostart")
	assertFileMode(t, autostartDir, 0o700)
	assertFileMode(t, filepath.Join(autostartDir, "eagleeye.desktop"), 0o600)
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode(%q) = %o, want %o", path, got, want)
	}
}
