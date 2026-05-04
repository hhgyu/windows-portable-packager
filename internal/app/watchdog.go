package app

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

const DefaultWatchdogTimeout = 60 * time.Second

var watchdogDisarmed atomic.Bool

// StartWatchdog launches a background timer that terminates the process if
// DisarmWatchdog is not called within timeout. Used to recover from cases
// where the launcher hangs before reaching the child-process spawn step
// (e.g. AV behavioural sandboxing on unsigned binaries).
//
// appName is included in the error dialog title when the watchdog fires.
func StartWatchdog(appName string, timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	go func() {
		time.Sleep(timeout)
		if watchdogDisarmed.Load() {
			return
		}
		seconds := int(timeout / time.Second)
		message := fmt.Sprintf(T(MsgStartupTimeout), seconds)
		title := fmt.Sprintf(T(MsgStartupTimeoutTitle), appName)
		fmt.Fprintln(os.Stderr, message)
		if !IsTerminal() {
			ShowErrorDialog(title, message)
		}
		os.Exit(2)
	}()
}

// DisarmWatchdog cancels the pending watchdog. Call this once the launcher
// has reached a known-good state (e.g. the child process has been spawned).
func DisarmWatchdog() {
	watchdogDisarmed.Store(true)
}
