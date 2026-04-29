// Package ui groups EagleEye's Fyne presentation packages.
//
// Subpackages provide system-tray integration, break overlays, preferences,
// animation, and localization. Application orchestration and runtime state
// belong in internal/app; UI packages should focus on rendering and callbacks.
// Updates from background goroutines must be scheduled onto Fyne's UI thread,
// typically with fyne.Do, unless they already run inside a Fyne callback.
package ui
