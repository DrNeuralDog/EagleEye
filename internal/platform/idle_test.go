package platform

import (
	"eagleeye/internal/core/timekeeper"
	"testing"
)

func TestNewIdleCheckerSatisfiesTimeKeeperContract(t *testing.T) {
	var checker timekeeper.IdleChecker = NewIdleChecker()
	if checker == nil {
		t.Fatalf("NewIdleChecker() = nil, want idle checker")
	}
}
