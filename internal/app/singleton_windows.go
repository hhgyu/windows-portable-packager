//go:build windows

package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/sys/windows"
)

// SingletonHandle owns a Windows named mutex that prevents multiple launcher
// instances from running concurrently for the same app.
type SingletonHandle struct {
	handle windows.Handle
	name   string
}

// AcquireSingleton tries to claim a per-user named mutex tied to appName.
// Returns (handle, true, nil) when this process is the first owner,
// (nil, false, nil) when another instance already holds the mutex, or
// (nil, false, err) on Windows API failure.
func AcquireSingleton(appName string) (*SingletonHandle, bool, error) {
	mutexName := buildMutexName(appName)

	namePtr, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return nil, false, fmt.Errorf("encode mutex name: %w", err)
	}

	h, err := windows.CreateMutex(nil, true, namePtr)
	if h == 0 {
		return nil, false, fmt.Errorf("create mutex: %w", err)
	}

	// CreateMutex returns a valid handle even when the mutex already exists;
	// the only signal of "not first" is err == ERROR_ALREADY_EXISTS. We
	// must close that handle to avoid leaking it.
	if err == windows.ERROR_ALREADY_EXISTS {
		windows.CloseHandle(h)
		return nil, false, nil
	}

	return &SingletonHandle{handle: h, name: mutexName}, true, nil
}

// Release closes the mutex handle. Safe to call on a nil receiver or twice.
func (s *SingletonHandle) Release() {
	if s == nil || s.handle == 0 {
		return
	}
	windows.CloseHandle(s.handle)
	s.handle = 0
}

// buildMutexName returns a Local\-namespaced mutex name derived from a
// per-package namespace tag plus appName, hashed for stability and to avoid
// reserved characters in kernel object names.
func buildMutexName(appName string) string {
	if appName == "" {
		appName = "windows-portable-packager"
	}
	const nsTag = "github.com/hhgyu/windows-portable-packager"
	sum := sha256.Sum256([]byte(nsTag + "::" + appName))
	return "Local\\WPP-Singleton-" + hex.EncodeToString(sum[:8])
}
