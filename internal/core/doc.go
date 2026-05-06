// Package core defines EagleEye's domain model and break-timer state machine.
//
// Subpackages contain scheduling configuration and TimeKeeper logic. Core code
// should stay independent of Fyne, persistence, and platform integrations;
// higher level packages such as internal/app wire it to UI, storage, and OS
// services.
package core
