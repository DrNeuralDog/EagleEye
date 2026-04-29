package platform

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	activationSecretFileName = "single_instance.secret"
	activationSecretSize     = 32
	activationMessagePrefix  = "EAGLEEYE_ACTIVATE_V1"
	maxActivationMessageSize = 256
	activationReadTimeout    = 500 * time.Millisecond
	activationDebounce       = 500 * time.Millisecond
)

// ErrAlreadyRunning indicates another instance already holds the lock
var ErrAlreadyRunning = errors.New("instance already running")

var errInvalidActivationSecret = errors.New("activation secret has invalid format")

// InstanceGuard holds the localhost listener used as the process lock
type InstanceGuard struct {
	listener         net.Listener
	address          string
	activationSecret []byte
	activationMu     sync.Mutex
	lastActivation   time.Time
}

// AcquireSingleInstance reserves the app-specific localhost port
func AcquireSingleInstance(appName string) (*InstanceGuard, error) {
	port := portFromName(appName)
	address := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", address)

	if err != nil {
		return nil, ErrAlreadyRunning
	}

	secret, err := loadOrCreateActivationSecret(appName)
	if err != nil {
		_ = listener.Close()

		return nil, fmt.Errorf("prepare activation secret: %w", err)
	}

	return &InstanceGuard{listener: listener, address: address, activationSecret: secret}, nil
}

// Release frees the localhost listener lock
func (guard *InstanceGuard) Release() error {
	if guard == nil || guard.listener == nil {
		return nil
	}

	return guard.listener.Close()
}

// Address returns the listener address used for activation pings
func (guard *InstanceGuard) Address() string {
	if guard == nil {
		return ""
	}

	return guard.address
}

// ListenForActivation accepts valid second-instance activation pings
func (guard *InstanceGuard) ListenForActivation(ctx context.Context, onActivate func()) {
	if guard == nil || guard.listener == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	go func() {
		<-ctx.Done()
		_ = guard.listener.Close()
	}()

	go func() {
		for {
			conn, err := guard.listener.Accept()

			if err != nil {
				if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
					return
				}

				continue
			}

			if err := conn.SetReadDeadline(time.Now().Add(activationReadTimeout)); err != nil {
				_ = conn.Close()

				continue
			}

			message, err := io.ReadAll(io.LimitReader(conn, maxActivationMessageSize+1))
			if closeErr := conn.Close(); closeErr != nil {
				continue
			}

			if err != nil {
				continue
			}

			if onActivate != nil && guard.isValidActivationMessage(message) && guard.allowActivation() {
				onActivate()
			}
		}
	}()
}

// NotifyRunningInstance asks the already-running instance to show UI
func NotifyRunningInstance(appName string) error {
	address := fmt.Sprintf("127.0.0.1:%d", portFromName(appName))
	conn, err := net.DialTimeout("tcp", address, 800*time.Millisecond)

	if err != nil {
		return err
	}

	defer conn.Close()

	secret, err := readActivationSecret(appName)
	if err != nil {
		return err
	}

	message, err := buildActivationMessage(secret)
	if err != nil {
		return err
	}

	_, err = conn.Write(message)

	return err
}

// isValidActivationMessage checks a ping against the guard secret
func (guard *InstanceGuard) isValidActivationMessage(message []byte) bool {
	if len(message) == 0 || len(message) > maxActivationMessageSize {
		return false
	}

	return isValidActivationMessage(message, guard.activationSecret)
}

// allowActivation throttles repeated activation callbacks
func (guard *InstanceGuard) allowActivation() bool {
	guard.activationMu.Lock()
	defer guard.activationMu.Unlock()

	now := time.Now()

	if !guard.lastActivation.IsZero() && now.Sub(guard.lastActivation) < activationDebounce {
		return false
	}

	guard.lastActivation = now

	return true
}

// loadOrCreateActivationSecret returns the per-user activation secret
func loadOrCreateActivationSecret(appName string) ([]byte, error) {
	path, err := activationSecretPath(appName)

	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create activation secret directory: %w", err)
	}

	if secret, err := readActivationSecretFile(path); err == nil {
		_ = os.Chmod(path, 0o600)
		return secret, nil
	} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, errInvalidActivationSecret) {
		return nil, err
	}

	secret := make([]byte, activationSecretSize)

	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate activation secret: %w", err)
	}

	encoded := []byte(hex.EncodeToString(secret) + "\n")

	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return nil, fmt.Errorf("write activation secret: %w", err)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("secure activation secret permissions: %w", err)
	}

	return secret, nil
}

// readActivationSecret reads the activation secret for second-instance pings
func readActivationSecret(appName string) ([]byte, error) {
	path, err := activationSecretPath(appName)

	if err != nil {
		return nil, err
	}

	return readActivationSecretFile(path)
}

// readActivationSecretFile decodes the hex-encoded secret from disk
func readActivationSecretFile(path string) ([]byte, error) {
	rawData, err := os.ReadFile(path)

	if err != nil {
		return nil, fmt.Errorf("read activation secret: %w", err)
	}

	secret, err := hex.DecodeString(strings.TrimSpace(string(rawData)))

	if err != nil || len(secret) != activationSecretSize {
		return nil, errInvalidActivationSecret
	}

	return secret, nil
}

// activationSecretPath returns the per-user secret file path
func activationSecretPath(appName string) (string, error) {
	configDir, err := os.UserConfigDir()

	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	return filepath.Join(configDir, appName, activationSecretFileName), nil
}

// buildActivationMessage signs a one-shot activation ping
func buildActivationMessage(secret []byte) ([]byte, error) {
	nonce := make([]byte, 16)

	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate activation nonce: %w", err)
	}

	nonceHex := hex.EncodeToString(nonce)
	mac := computeActivationMAC(secret, nonceHex)
	message := fmt.Sprintf("%s:%s:%s\n", activationMessagePrefix, nonceHex, hex.EncodeToString(mac))

	return []byte(message), nil
}

// isValidActivationMessage verifies prefix, nonce, and HMAC signature
func isValidActivationMessage(message, secret []byte) bool {
	parts := strings.Split(strings.TrimSpace(string(message)), ":")

	if len(parts) != 3 || parts[0] != activationMessagePrefix {
		return false
	}

	if _, err := hex.DecodeString(parts[1]); err != nil {
		return false
	}

	providedMAC, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}

	expectedMAC := computeActivationMAC(secret, parts[1])

	return hmac.Equal(providedMAC, expectedMAC)
}

// computeActivationMAC signs the activation prefix and nonce
func computeActivationMAC(secret []byte, nonceHex string) []byte {
	mac := hmac.New(sha256.New, secret)

	_, _ = mac.Write([]byte(activationMessagePrefix))
	_, _ = mac.Write([]byte(":"))
	_, _ = mac.Write([]byte(nonceHex))

	return mac.Sum(nil)
}

// portFromName maps an app name to a stable localhost port
func portFromName(appName string) int {
	const (
		minPort = 20000
		maxPort = 39999
	)

	hash := fnv.New32a()
	_, _ = hash.Write([]byte(appName))
	rangeSize := maxPort - minPort + 1

	return minPort + int(hash.Sum32()%uint32(rangeSize))
}
