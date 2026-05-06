// Package storage persists EagleEye user data on the local filesystem.
//
// The package currently stores preferences as YAML under the per-user config
// directory, supports EAGLEEYE_CONFIG_PATH for settings overrides, and resolves
// the JSONL application log path. Higher-level packages own application
// behavior; storage is responsible for paths, serialization, and file I/O.
package storage
