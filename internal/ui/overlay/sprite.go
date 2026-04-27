package overlay

import "fyne.io/fyne/v2"

func spriteScaleForResource(resource fyne.Resource) float32 {
	return 1
}

func spriteTransformForResource(resource fyne.Resource) spriteTransform {
	scale := spriteScaleForResource(resource)
	return spriteTransform{
		scaleX: 1,
		scaleY: scale,
	}
}

type spriteTransform struct {
	scaleX          float32
	scaleY          float32
	offsetYFraction float32
	stretch         bool
}

func normalizeSpriteTransform(transform spriteTransform) spriteTransform {
	if transform.scaleX <= 0 {
		transform.scaleX = 1
	}
	if transform.scaleY <= 0 {
		transform.scaleY = 1
	}
	return transform
}
