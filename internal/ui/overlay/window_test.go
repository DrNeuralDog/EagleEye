package overlay

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestCalculateOverlayCardSizeBasicFraction(t *testing.T) {
	screenSize := fyne.NewSize(1920, 1080)
	minSize := fyne.NewSize(100, 100)

	got := calculateOverlayCardSize(screenSize, minSize)
	want := fyne.NewSize(1920*overlayWidthFraction, 1080*overlayHeightFraction)

	assertSizeEquals(t, got, want)
}

func TestCalculateOverlayCardSizeAppliesMinSize(t *testing.T) {
	screenSize := fyne.NewSize(1920, 1080)
	minSize := fyne.NewSize(400, 250)

	got := calculateOverlayCardSize(screenSize, minSize)
	want := fyne.NewSize(400, 250)

	assertSizeEquals(t, got, want)
}

func TestCalculateOverlayCardSizeCapsByScreen(t *testing.T) {
	screenSize := fyne.NewSize(300, 200)
	minSize := fyne.NewSize(500, 250)

	got := calculateOverlayCardSize(screenSize, minSize)
	want := fyne.NewSize(300, 200)

	assertSizeEquals(t, got, want)
}

func assertSizeEquals(t *testing.T, got fyne.Size, want fyne.Size) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected size: got=%v want=%v", got, want)
	}
}
