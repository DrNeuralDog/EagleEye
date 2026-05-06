package platform

import (
	"context"
	"net"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// TestSingleInstanceActivationRequiresValidMessage verifies only signed pings trigger activation
func TestSingleInstanceActivationRequiresValidMessage(t *testing.T) {
	setPlatformUserConfigEnv(t, t.TempDir())

	appName := "EagleEyeIPCTest" + strconv.FormatInt(time.Now().UnixNano(), 10)
	guard, err := AcquireSingleInstance(appName)

	if err != nil {
		t.Fatalf("AcquireSingleInstance() error = %v", err)
	}

	defer func() {
		_ = guard.Release()
	}()

	activated := make(chan struct{}, 1)

	guard.ListenForActivation(context.Background(), func() {
		activated <- struct{}{}
	})

	if err := writeRawActivationMessage(guard.Address(), []byte("invalid\n")); err != nil {
		t.Fatalf("writeRawActivationMessage() error = %v", err)
	}

	select {
	case <-activated:
		t.Fatalf("invalid activation message triggered callback")
	case <-time.After(150 * time.Millisecond):
	}

	if err := NotifyRunningInstance(appName); err != nil {
		t.Fatalf("NotifyRunningInstance() error = %v", err)
	}
	select {
	case <-activated:
	case <-time.After(time.Second):
		t.Fatalf("valid activation message did not trigger callback")
	}
}

// TestSingleInstanceActivationIsDebounced verifies rapid repeated pings are throttled
func TestSingleInstanceActivationIsDebounced(t *testing.T) {
	setPlatformUserConfigEnv(t, t.TempDir())

	appName := "EagleEyeDebounceTest" + strconv.FormatInt(time.Now().UnixNano(), 10)
	guard, err := AcquireSingleInstance(appName)

	if err != nil {
		t.Fatalf("AcquireSingleInstance() error = %v", err)
	}

	defer func() {
		_ = guard.Release()
	}()

	activated := make(chan struct{}, 3)
	guard.ListenForActivation(context.Background(), func() {
		activated <- struct{}{}
	})

	if err := NotifyRunningInstance(appName); err != nil {
		t.Fatalf("NotifyRunningInstance() first error = %v", err)
	}

	expectActivation(t, activated)

	time.Sleep(activationDebounce + 50*time.Millisecond)
	if err := NotifyRunningInstance(appName); err != nil {
		t.Fatalf("NotifyRunningInstance() second error = %v", err)
	}

	expectActivation(t, activated)

	if err := NotifyRunningInstance(appName); err != nil {
		t.Fatalf("NotifyRunningInstance() third error = %v", err)
	}

	select {
	case <-activated:
		t.Fatalf("activation was not debounced")
	case <-time.After(150 * time.Millisecond):
	}
}

// TestSingleInstanceActivationStopsWhenContextCanceled verifies canceling context releases the port
func TestSingleInstanceActivationStopsWhenContextCanceled(t *testing.T) {
	setPlatformUserConfigEnv(t, t.TempDir())

	appName := "EagleEyeCancelTest" + strconv.FormatInt(time.Now().UnixNano(), 10)
	guard, err := AcquireSingleInstance(appName)

	if err != nil {
		t.Fatalf("AcquireSingleInstance() error = %v", err)
	}

	defer func() {
		_ = guard.Release()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	guard.ListenForActivation(ctx, nil)

	cancel()

	deadline := time.After(time.Second)
	for {
		replacement, err := AcquireSingleInstance(appName)

		if err == nil {
			_ = replacement.Release()

			return
		}

		select {
		case <-deadline:
			t.Fatalf("listener did not stop after context cancel")
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// expectActivation waits for the activation callback signal
func expectActivation(t *testing.T, activated <-chan struct{}) {
	t.Helper()

	select {
	case <-activated:
	case <-time.After(time.Second):
		t.Fatalf("activation callback was not called")
	}
}

// writeRawActivationMessage sends a custom activation payload to the listener
func writeRawActivationMessage(address string, message []byte) error {
	conn, err := net.DialTimeout("tcp", address, 800*time.Millisecond)

	if err != nil {
		return err
	}

	defer conn.Close()

	_, err = conn.Write(message)

	return err
}

// setPlatformUserConfigEnv isolates platform config paths inside the test temp dir
func setPlatformUserConfigEnv(t *testing.T, path string) {
	t.Helper()

	switch runtime.GOOS {
	case "windows":
		t.Setenv("APPDATA", path)
	default:
		t.Setenv("XDG_CONFIG_HOME", path)
	}
}
