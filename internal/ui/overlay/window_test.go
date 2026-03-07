package overlay

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
)

func TestCalculateOverlayCardSizeBasicFraction(t *testing.T) {
	screenSize := fyne.NewSize(1920, 1080)
	minSize := fyne.NewSize(100, 100)

	got := calculateOverlayCardSize(screenSize, minSize)
	want := fyne.NewSize(1920*overlayWidthFraction, 1080*overlayHeightFraction+overlayBottomClearance)

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

func TestSpriteScaleForOversizedDirectionalSprites(t *testing.T) {
	tests := []struct {
		name string
		want float32
	}{
		{name: "sprites/Falcon looks down.png", want: oversizedSpriteScale},
		{name: "sprites/Falcon looks left.png", want: oversizedSpriteScale},
		{name: "sprites/Falcon looks right.png", want: 1},
		{name: "sprites/Falcon looks straight ahead.png", want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := fyne.NewStaticResource(tt.name, nil)

			got := spriteScaleForResource(resource)

			if got != tt.want {
				t.Fatalf("unexpected sprite scale: got=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestSpriteTransformForOversizedDirectionalSprites(t *testing.T) {
	resource := fyne.NewStaticResource("sprites/Falcon looks down.png", nil)

	got := spriteTransformForResource(resource)

	if got.scaleY != oversizedSpriteScale {
		t.Fatalf("unexpected sprite height scale: got=%v want=%v", got.scaleY, oversizedSpriteScale)
	}
	if got.scaleX != oversizedSpriteWidth {
		t.Fatalf("unexpected sprite width scale: got=%v want=%v", got.scaleX, oversizedSpriteWidth)
	}
	if got.offsetYFraction != oversizedSpriteOffsetY {
		t.Fatalf("unexpected sprite y offset: got=%v want=%v", got.offsetYFraction, oversizedSpriteOffsetY)
	}
	if !got.stretch {
		t.Fatal("expected oversized directional sprite to use stretch fill")
	}
}

func TestRightPanelLayoutKeepsClearanceUnderSkip(t *testing.T) {
	image := newFixedCanvasObject(fyne.NewSize(0, 0))
	skip := newFixedCanvasObject(fyne.NewSize(50, 30))
	layout := &rightPanelLayout{}
	panelSize := fyne.NewSize(160, 200)

	layout.Layout([]fyne.CanvasObject{image, skip}, panelSize)

	skipBottom := skip.Position().Y + skip.Size().Height
	maxBottom := panelSize.Height - overlayBottomClearance
	if skipBottom > maxBottom+0.01 {
		t.Fatalf("skip button exceeds bottom clearance: got bottom=%v max=%v", skipBottom, maxBottom)
	}
}

func TestRightPanelLayoutAppliesDirectionalSpriteOffsetAndWidth(t *testing.T) {
	defaultImage := newFixedCanvasObject(fyne.NewSize(0, 0))
	defaultSkip := newFixedCanvasObject(fyne.NewSize(50, 30))
	defaultLayout := &rightPanelLayout{}
	panelSize := fyne.NewSize(160, 200)
	defaultLayout.Layout([]fyne.CanvasObject{defaultImage, defaultSkip}, panelSize)

	scaledImage := newFixedCanvasObject(fyne.NewSize(0, 0))
	scaledSkip := newFixedCanvasObject(fyne.NewSize(50, 30))
	scaledLayout := &rightPanelLayout{}
	scaledLayout.SetSpriteTransform(spriteTransformForResource(fyne.NewStaticResource("sprites/Falcon looks left.png", nil)))
	scaledLayout.Layout([]fyne.CanvasObject{scaledImage, scaledSkip}, panelSize)

	if scaledImage.Size().Height >= defaultImage.Size().Height {
		t.Fatalf("scaled sprite was not reduced: got=%v default=%v", scaledImage.Size().Height, defaultImage.Size().Height)
	}
	if scaledImage.Size().Width <= scaledImage.Size().Height {
		t.Fatalf("scaled sprite was not widened: got width=%v height=%v", scaledImage.Size().Width, scaledImage.Size().Height)
	}
	wantWidth := scaledImage.Size().Height * oversizedSpriteWidth
	if math.Abs(float64(scaledImage.Size().Width-wantWidth)) > 0.01 {
		t.Fatalf("unexpected scaled sprite width: got=%v want=%v", scaledImage.Size().Width, wantWidth)
	}
	if scaledImage.Position().Y <= defaultImage.Position().Y {
		t.Fatalf("scaled sprite was not moved lower: got y=%v default y=%v", scaledImage.Position().Y, defaultImage.Position().Y)
	}
}

type fixedCanvasObject struct {
	minSize fyne.Size
	pos     fyne.Position
	size    fyne.Size
	visible bool
}

func newFixedCanvasObject(minSize fyne.Size) *fixedCanvasObject {
	return &fixedCanvasObject{minSize: minSize, visible: true}
}

func (object *fixedCanvasObject) MinSize() fyne.Size {
	return object.minSize
}

func (object *fixedCanvasObject) Move(pos fyne.Position) {
	object.pos = pos
}

func (object *fixedCanvasObject) Position() fyne.Position {
	return object.pos
}

func (object *fixedCanvasObject) Resize(size fyne.Size) {
	object.size = size
}

func (object *fixedCanvasObject) Size() fyne.Size {
	return object.size
}

func (object *fixedCanvasObject) Hide() {
	object.visible = false
}

func (object *fixedCanvasObject) Show() {
	object.visible = true
}

func (object *fixedCanvasObject) Visible() bool {
	return object.visible
}

func (object *fixedCanvasObject) Refresh() {}

func assertSizeEquals(t *testing.T, got fyne.Size, want fyne.Size) {
	t.Helper()
	if got != want {
		t.Fatalf("unexpected size: got=%v want=%v", got, want)
	}
}
