package app

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const DefaultWatchdogTimeout = 60 * time.Second

var (
	watchdogMu       sync.Mutex
	watchdogTimer    *time.Timer
	watchdogAppName  string
	watchdogTimeout  time.Duration
	watchdogDisarmed atomic.Bool
)

// StartWatchdog launches a background timer that terminates the process if
// DisarmWatchdog is not called within timeout. Used to recover from cases
// where the launcher hangs before reaching the child-process spawn step
// (e.g. AV behavioural sandboxing on unsigned binaries).
//
// User-blocking UI between StartWatchdog and DisarmWatchdog must be wrapped
// with PauseWatchdog and ResumeWatchdog so the user's dialog wait time does
// not eat into the launcher's startup budget.
//
// appName is included in the error dialog title when the watchdog fires.
func StartWatchdog(appName string, timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	watchdogMu.Lock()
	defer watchdogMu.Unlock()

	watchdogDisarmed.Store(false)
	watchdogAppName = appName
	watchdogTimeout = timeout

	if watchdogTimer != nil {
		watchdogTimer.Stop()
	}
	watchdogTimer = time.AfterFunc(timeout, fireWatchdog)
}

func fireWatchdog() {
	if watchdogDisarmed.Load() {
		return
	}
	watchdogDisarmed.Store(true)

	watchdogMu.Lock()
	timeout := watchdogTimeout
	appName := watchdogAppName
	watchdogMu.Unlock()

	seconds := int(timeout / time.Second)
	message := fmt.Sprintf(T(MsgStartupTimeout), seconds)
	title := fmt.Sprintf(T(MsgStartupTimeoutTitle), appName)
	fmt.Fprintln(os.Stderr, message)
	if !IsTerminal() {
		ShowErrorDialog(title, message)
	}
	os.Exit(2)
}

// DisarmWatchdog cancels the pending watchdog. Call this once the launcher
// has reached a known-good state (e.g. the child process has been spawned).
// After Disarm, Pause/Resume become no-ops.
func DisarmWatchdog() {
	watchdogDisarmed.Store(true)
	watchdogMu.Lock()
	if watchdogTimer != nil {
		watchdogTimer.Stop()
	}
	watchdogMu.Unlock()
}

// PauseWatchdog stops the pending timer without firing. Used to exclude
// user-blocking UI dwell time from the watchdog budget. Returns immediately
// if the watchdog is already disarmed or was never started.
func PauseWatchdog() {
	if watchdogDisarmed.Load() {
		return
	}
	watchdogMu.Lock()
	if watchdogTimer != nil {
		watchdogTimer.Stop()
	}
	watchdogMu.Unlock()
}

// ResumeWatchdog re-arms the watchdog with a fresh full timeout, granting
// the launcher another full budget after a user-blocking pause. Returns
// immediately if the watchdog is already disarmed or was never started.
func ResumeWatchdog() {
	if watchdogDisarmed.Load() {
		return
	}
	watchdogMu.Lock()
	defer watchdogMu.Unlock()
	if watchdogTimeout <= 0 {
		return
	}
	if watchdogTimer != nil {
		watchdogTimer.Stop()
	}
	watchdogTimer = time.AfterFunc(watchdogTimeout, fireWatchdog)
}
