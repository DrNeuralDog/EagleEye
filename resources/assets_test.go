package resources

import "testing"

func TestSpriteLoadsAndCachesResource(t *testing.T) {
	cache := NewCache()

	first, err := cache.Sprite("Falcon looks straight ahead.png")
	if err != nil {
		t.Fatalf("Cache.Sprite() error = %v", err)
	}
	second, err := cache.Sprite("Falcon looks straight ahead.png")
	if err != nil {
		t.Fatalf("Cache.Sprite() second error = %v", err)
	}
	if first != second {
		t.Fatalf("Cache.Sprite() did not return cached resource")
	}
}

func TestLogoLoadsResource(t *testing.T) {
	cache := NewCache()

	logo, err := cache.Logo("Logo_Optimal_Gradient.png")
	if err != nil {
		t.Fatalf("Cache.Logo() error = %v", err)
	}
	if logo == nil {
		t.Fatalf("Cache.Logo() = nil, want resource")
	}
}

func TestMissingResourceReturnsError(t *testing.T) {
	cache := NewCache()

	if _, err := cache.Sprite("missing.png"); err == nil {
		t.Fatalf("Cache.Sprite() error = nil, want missing resource error")
	}
}

func TestMustSpritePanicsForMissingResource(t *testing.T) {
	cache := NewCache()

	assertPanics(t, func() {
		cache.MustSprite("missing.png")
	})
}

func TestMustLogoPanicsForMissingResource(t *testing.T) {
	cache := NewCache()

	assertPanics(t, func() {
		cache.MustLogo("missing.png")
	})
}

func TestPackageLevelAPIsUseDefaultCache(t *testing.T) {
	sprite, err := Sprite("Falcon looks straight ahead.png")
	if err != nil {
		t.Fatalf("Sprite() error = %v", err)
	}
	if sprite == nil {
		t.Fatalf("Sprite() = nil, want package-level resource")
	}
	logo, err := Logo("Logo_Optimal_Gradient.png")
	if err != nil {
		t.Fatalf("Logo() error = %v", err)
	}
	if logo == nil {
		t.Fatalf("Logo() = nil, want package-level resource")
	}
}

func assertPanics(t *testing.T, run func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("function did not panic")
		}
	}()
	run()
}
