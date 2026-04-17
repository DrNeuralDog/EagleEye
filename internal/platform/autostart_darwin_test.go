//go:build darwin

package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildLaunchAgentPlistEscapesXML(t *testing.T) {
	plist, err := buildLaunchAgentPlist("com.eagleeye.test", `/Applications/Eagle&Eye.app/Contents/MacOS/Eagle"Eye`)
	if err != nil {
		t.Fatalf("buildLaunchAgentPlist() error = %v", err)
	}
	if !strings.Contains(plist, "Eagle&amp;Eye.app") {
		t.Fatalf("plist path was not XML-escaped:\n%s", plist)
	}
	if !strings.Contains(plist, "Eagle&quot;Eye") {
		t.Fatalf("plist quote was not XML-escaped:\n%s", plist)
	}
	if !strings.Contains(plist, "<string>--autostart</string>") {
		t.Fatalf("plist missing autostart argument:\n%s", plist)
	}
}

func TestBuildLaunchAgentPlistRejectsControlCharacters(t *testing.T) {
	if _, err := buildLaunchAgentPlist("com.eagleeye.test\nbad", "/Applications/EagleEye.app"); err == nil {
		t.Fatalf("buildLaunchAgentPlist() error = nil, want label control character error")
	}
	if _, err := buildLaunchAgentPlist("com.eagleeye.test", "/Applications/EagleEye.app\nbad"); err == nil {
		t.Fatalf("buildLaunchAgentPlist() error = nil, want path control character error")
	}
}

func TestEnableAutostartUsesPrivateFileModes(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	service := NewService()
	if err := service.EnableAutostart("EagleEye", "/Applications/EagleEye.app/Contents/MacOS/EagleEye"); err != nil {
		t.Fatalf("EnableAutostart() error = %v", err)
	}

	launchAgentsDir := filepath.Join(homeDir, "Library", "LaunchAgents")
	assertFileMode(t, launchAgentsDir, 0o700)
	assertFileMode(t, filepath.Join(launchAgentsDir, "com.eagleeye.eagleeye.plist"), 0o600)
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
