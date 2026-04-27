package overlay

import "fyne.io/fyne/v2"

const (
	overlayWidthFraction    = float32(0.14)
	overlayHeightFraction   = float32(0.18)
	defaultScreenWidth      = float32(1920)
	defaultScreenHeight     = float32(1080)
	overlayCardCornerRadius = float32(32)
	overlayBottomClearance  = float32(12)
	skipModeImageScale      = float32(1.05)
)

func (overlay *Window) resizeToScreenFraction() {
	screenSize := overlay.resolveScreenSize()
	overlaySize := calculateOverlayCardSize(screenSize, overlay.window.Content().MinSize())
	overlay.window.Resize(overlaySize)
	overlay.window.CenterOnScreen()
}

func (overlay *Window) resolveScreenSize() fyne.Size {
	screenSize := fyne.NewSize(defaultScreenWidth, defaultScreenHeight)
	canvasSize := overlay.window.Canvas().Size()
	// Canvas size can be reused as a proxy for monitor size when it is clearly screen-like.
	if canvasSize.Width >= 1024 && canvasSize.Height >= 720 {
		screenSize = canvasSize
	}
	return screenSize
}

func calculateOverlayCardSize(screenSize, minSize fyne.Size) fyne.Size {
	if screenSize.Width <= 0 {
		screenSize.Width = defaultScreenWidth
	}
	if screenSize.Height <= 0 {
		screenSize.Height = defaultScreenHeight
	}

	width := screenSize.Width * overlayWidthFraction
	height := screenSize.Height*overlayHeightFraction + overlayBottomClearance
	if width < minSize.Width {
		width = minSize.Width
	}
	if height < minSize.Height {
		height = minSize.Height
	}
	if width > screenSize.Width {
		width = screenSize.Width
	}
	if height > screenSize.Height {
		height = screenSize.Height
	}
	return fyne.NewSize(width, height)
}

type rightPanelLayout struct {
	spriteTransform spriteTransform
}

func (layout *rightPanelLayout) SetSpriteScale(scale float32) {
	layout.SetSpriteTransform(spriteTransform{scaleX: 1, scaleY: scale})
}

func (layout *rightPanelLayout) SetSpriteTransform(transform spriteTransform) {
	layout.spriteTransform = normalizeSpriteTransform(transform)
}

func (layout *rightPanelLayout) currentSpriteTransform() spriteTransform {
	return normalizeSpriteTransform(layout.spriteTransform)
}

func (layout *rightPanelLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 2 {
		return
	}
	image := objects[0]
	skip := objects[1]
	transform := layout.currentSpriteTransform()

	skipSize := skip.MinSize()
	if !skip.Visible() {
		baseSide := size.Height
		if baseSide > size.Width {
			baseSide = size.Width
		}
		height := baseSide * transform.scaleY
		width := height * transform.scaleX
		x := size.Width - width
		if x < 0 {
			x = 0
		}
		y := (size.Height-baseSide)/2 + (baseSide - height) + baseSide*transform.offsetYFraction
		if y < 0 {
			y = 0
		}
		image.Move(fyne.NewPos(x, y))
		image.Resize(fyne.NewSize(width, height))
		return
	}

	skipHeight := skipSize.Height
	if skipHeight > size.Height*0.25 {
		skipHeight = size.Height * 0.25
	}
	bottomClearance := overlayBottomClearance
	if bottomClearance > size.Height-skipHeight {
		bottomClearance = size.Height - skipHeight
	}
	if bottomClearance < 0 {
		bottomClearance = 0
	}
	imageAreaHeight := size.Height - skipHeight - bottomClearance
	if imageAreaHeight < 0 {
		imageAreaHeight = 0
	}

	margin := imageAreaHeight * 0.05
	baseSide := imageAreaHeight * 0.90 * skipModeImageScale
	if baseSide > size.Width-margin {
		baseSide = size.Width - margin
	}
	if baseSide < 0 {
		baseSide = 0
	}
	height := baseSide * transform.scaleY
	width := height * transform.scaleX
	if width > size.Width-margin {
		width = size.Width - margin
	}
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	x := size.Width - margin - width
	if x < 0 {
		x = 0
	}
	y := margin + (baseSide - height) + baseSide*transform.offsetYFraction
	image.Move(fyne.NewPos(x, y))
	image.Resize(fyne.NewSize(width, height))

	skipWidth := skipSize.Width
	skipWidth = skipWidth * 1.4
	if skipWidth > size.Width {
		skipWidth = size.Width
	}
	skipX := x + width - skipWidth
	if skipX < 0 {
		skipX = 0
	}
	skipY := imageAreaHeight + (skipHeight-skipSize.Height)/2
	maxSkipY := size.Height - bottomClearance - skipSize.Height
	if skipY > maxSkipY {
		skipY = maxSkipY
	}
	if skipY < 0 {
		skipY = 0
	}
	skip.Move(fyne.NewPos(skipX, skipY))
	skip.Resize(fyne.NewSize(skipWidth, skipSize.Height))
}

func (layout *rightPanelLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 2 {
		return fyne.NewSize(0, 0)
	}
	imageMin := objects[0].MinSize()
	skipMin := objects[1].MinSize()
	if !objects[1].Visible() {
		return imageMin
	}
	width := imageMin.Width
	if skipMin.Width > width {
		width = skipMin.Width
	}
	return fyne.NewSize(width, imageMin.Height+skipMin.Height+overlayBottomClearance)
}

type leftPanelLayout struct{}

func (layout *leftPanelLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 4 {
		return
	}
	title := objects[0]
	subtitle := objects[1]
	exercise := objects[2]
	timer := objects[3]

	pad := size.Height * 0.05
	availableWidth := size.Width - pad*2
	if availableWidth < 0 {
		availableWidth = 0
	}

	titleSize := title.MinSize()
	title.Move(fyne.NewPos(pad, pad))
	title.Resize(fyne.NewSize(availableWidth, titleSize.Height))

	subtitleSize := subtitle.MinSize()
	subtitleY := pad + titleSize.Height + 6
	subtitle.Move(fyne.NewPos(pad, subtitleY))
	subtitle.Resize(fyne.NewSize(availableWidth, subtitleSize.Height))

	exerciseSize := exercise.MinSize()
	exerciseY := subtitleY + subtitleSize.Height + 8
	exercise.Move(fyne.NewPos(pad, exerciseY))
	exercise.Resize(fyne.NewSize(availableWidth, exerciseSize.Height))

	timerSize := timer.MinSize()
	timerY := size.Height - pad - timerSize.Height
	if timerY < 0 {
		timerY = 0
	}
	timer.Move(fyne.NewPos(pad, timerY))
	timer.Resize(timerSize)
}

func (layout *leftPanelLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 4 {
		return fyne.NewSize(0, 0)
	}
	titleSize := objects[0].MinSize()
	subtitleSize := objects[1].MinSize()
	exerciseSize := objects[2].MinSize()
	timerSize := objects[3].MinSize()

	width := titleSize.Width
	if subtitleSize.Width > width {
		width = subtitleSize.Width
	}
	if exerciseSize.Width > width {
		width = exerciseSize.Width
	}
	if timerSize.Width > width {
		width = timerSize.Width
	}
	height := titleSize.Height + subtitleSize.Height + exerciseSize.Height + timerSize.Height + 40
	return fyne.NewSize(width+20, height)
}

type overlayCardHostLayout struct {
	fullscreen bool
}

func (layout *overlayCardHostLayout) SetFullscreen(fullscreen bool) {
	layout.fullscreen = fullscreen
}

func (layout *overlayCardHostLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	card := objects[0]
	if !layout.fullscreen {
		card.Move(fyne.NewPos(0, 0))
		card.Resize(size)
		return
	}

	cardSize := calculateOverlayCardSize(size, card.MinSize())
	x := (size.Width - cardSize.Width) / 2
	if x < 0 {
		x = 0
	}
	y := (size.Height - cardSize.Height) / 2
	if y < 0 {
		y = 0
	}
	card.Move(fyne.NewPos(x, y))
	card.Resize(cardSize)
}

func (layout *overlayCardHostLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	return objects[0].MinSize()
}
