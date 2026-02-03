package platform

import (
	"errors"
	"fmt"
	"hash/fnv"
	"net"
)

// ErrAlreadyRunning indicates another instance already holds the lock.
var ErrAlreadyRunning = errors.New("instance already running")

// InstanceGuard holds the single-instance lock.
type InstanceGuard struct {
	listener net.Listener
	address  string
}

// AcquireSingleInstance attempts to bind a deterministic localhost port.
func AcquireSingleInstance(appName string) (*InstanceGuard, error) {
	port := portFromName(appName)
	address := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, ErrAlreadyRunning
	}
	return &InstanceGuard{listener: listener, address: address}, nil
}

// Release frees the single instance lock.
func (guard *InstanceGuard) Release() error {
	if guard == nil || guard.listener == nil {
		return nil
	}
	return guard.listener.Close()
}

// Address returns the bound address.
func (guard *InstanceGuard) Address() string {
	if guard == nil {
		return ""
	}
	return guard.address
}

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
