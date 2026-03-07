//go:build !windows

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindAllowedExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, ok := findAllowedExecutable("tool", []string{dir})
	if !ok {
		t.Fatalf("findAllowedExecutable() ok = false, want true")
	}
	if got != path {
		t.Fatalf("findAllowedExecutable() = %q, want %q", got, path)
	}

	if _, ok := findAllowedExecutable("../tool", []string{dir}); ok {
		t.Fatalf("findAllowedExecutable() accepted path traversal name")
	}
}
