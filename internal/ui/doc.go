package ui

// Package ui contains presentation layer components for EagleEye.
//
// Subpackages provide the overlay window, preferences window, system tray,
// animation engine, and localization helpers. Runtime/business state belongs
// to internal/app; UI packages should stay focused on rendering and callbacks.
// Mutations from background goroutines must be scheduled onto Fyne's UI thread,
// usually with fyne.Do, unless they already run inside a Fyne callback.
