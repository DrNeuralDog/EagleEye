package resources

import (
	"embed"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
)

const (
	spriteDir = "sprites/"
	logoDir   = "logo/"
)

//go:embed sprites/*.png
var spriteFS embed.FS

//go:embed logo/*.png
var logoFS embed.FS

// Cache loads and memoizes embedded Fyne resources.
type Cache struct {
	sprites sync.Map
	logos   sync.Map
}

var defaultCache = NewCache()

// NewCache creates an isolated resource cache.
func NewCache() *Cache {
	return &Cache{}
}

// Sprite returns a Fyne resource for the given sprite file.
func Sprite(fileName string) (fyne.Resource, error) {
	return defaultCache.Sprite(fileName)
}

// MustSprite returns a Fyne resource or panics on error.
func MustSprite(fileName string) fyne.Resource {
	return defaultCache.MustSprite(fileName)
}

// Logo returns a Fyne resource for the given logo file.
func Logo(fileName string) (fyne.Resource, error) {
	return defaultCache.Logo(fileName)
}

// MustLogo returns a Fyne resource or panics on error.
func MustLogo(fileName string) fyne.Resource {
	return defaultCache.MustLogo(fileName)
}

// Sprite returns a Fyne resource for the given sprite file.
func (cache *Cache) Sprite(fileName string) (fyne.Resource, error) {
	return loadResource(spriteFS, spriteDir+fileName, &cache.sprites)
}

// MustSprite returns a Fyne resource or panics on error.
func (cache *Cache) MustSprite(fileName string) fyne.Resource {
	resource, err := cache.Sprite(fileName)
	if err != nil {
		panic(err)
	}
	return resource
}

// Logo returns a Fyne resource for the given logo file.
func (cache *Cache) Logo(fileName string) (fyne.Resource, error) {
	return loadResource(logoFS, logoDir+fileName, &cache.logos)
}

// MustLogo returns a Fyne resource or panics on error.
func (cache *Cache) MustLogo(fileName string) fyne.Resource {
	resource, err := cache.Logo(fileName)
	if err != nil {
		panic(err)
	}
	return resource
}

func loadResource(fs embed.FS, path string, cache *sync.Map) (fyne.Resource, error) {
	if cached, ok := cache.Load(path); ok {
		return cached.(fyne.Resource), nil
	}

	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load resource %s: %w", path, err)
	}

	resource := fyne.NewStaticResource(path, data)
	cache.Store(path, resource)
	return resource, nil
}
