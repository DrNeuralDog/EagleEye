package platform

import (
	"eagleeye/internal/core/timekeeper"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

type idleProvider struct{}

type lastInputInfo struct {
	cbSize uint32
	dwTime uint32
}

func newIdleProvider() timekeeper.IdleChecker {
	return &idleProvider{}
}

func (provider *idleProvider) IdleDuration() (time.Duration, error) {
	info := lastInputInfo{cbSize: uint32(unsafe.Sizeof(lastInputInfo{}))}

	user32 := syscall.NewLazyDLL("user32.dll")
	getLastInputInfo := user32.NewProc("GetLastInputInfo")
	result, _, err := getLastInputInfo.Call(uintptr(unsafe.Pointer(&info)))

	if result == 0 {
		if err != nil {
			return 0, fmt.Errorf("get last input info: %w", err)
		}

		return 0, fmt.Errorf("get last input info: unknown error")
	}

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getTickCount64 := kernel32.NewProc("GetTickCount64")
	tickResult, _, tickErr := getTickCount64.Call()

	if tickResult == 0 && tickErr != nil {
		return 0, fmt.Errorf("get tick count: %w", tickErr)
	}

	idleMillis := uint64(tickResult) - uint64(info.dwTime)
	const maxDurationMillis = uint64(1<<63-1) / uint64(time.Millisecond)

	if idleMillis > maxDurationMillis {
		return time.Duration(1<<63 - 1), nil
	}

	return time.Duration(idleMillis) * time.Millisecond, nil
}
