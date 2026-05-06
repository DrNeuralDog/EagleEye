//go:build windows

package platform

import "testing"

func TestValidateWindowsExecPath(t *testing.T) {
	if err := validateWindowsExecPath(`C:\Program Files\EagleEye\EagleEye.exe`); err != nil {
		t.Fatalf("validateWindowsExecPath() error = %v", err)
	}

	for _, path := range []string{
		`EagleEye.exe`,
		"C:\\Program Files\\EagleEye\\EagleEye.exe\n",
		`C:\Program Files\Eagle"Eye\EagleEye.exe`,
		`C:\Users\%USERNAME%\EagleEye\EagleEye.exe`,
	} {
		if err := validateWindowsExecPath(path); err == nil {
			t.Fatalf("validateWindowsExecPath(%q) error = nil, want error", path)
		}
	}
}

func TestQuoteWindowsPathDoesNotTrimInput(t *testing.T) {
	got := quoteWindowsPath(`C:\Program Files\EagleEye\EagleEye.exe`)
	want := `"C:\Program Files\EagleEye\EagleEye.exe"`

	if got != want {
		t.Fatalf("quoteWindowsPath() = %q, want %q", got, want)
	}
}

func TestBuildWindowsAutostartCommandIncludesAutostartArg(t *testing.T) {
	got := buildWindowsAutostartCommand(`C:\Program Files\EagleEye\EagleEye.exe`)
	want := `"C:\Program Files\EagleEye\EagleEye.exe" --autostart`

	if got != want {
		t.Fatalf("buildWindowsAutostartCommand() = %q, want %q", got, want)
	}
}
