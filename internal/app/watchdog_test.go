package app

import (
	"testing"
	"time"
)

func resetWatchdogState(t *testing.T) {
	t.Helper()
	watchdogMu.Lock()
	if watchdogTimer != nil {
		watchdogTimer.Stop()
		watchdogTimer = nil
	}
	watchdogAppName = ""
	watchdogTimeout = 0
	watchdogMu.Unlock()
	watchdogDisarmed.Store(false)
}

func TestWatchdogDisarmBeforeFire(t *testing.T) {
	resetWatchdogState(t)
	defer resetWatchdogState(t)

	StartWatchdog("test", 30*time.Millisecond)
	DisarmWatchdog()
	time.Sleep(60 * time.Millisecond)

	if !watchdogDisarmed.Load() {
		t.Fatal("watchdog should be disarmed after DisarmWatchdog")
	}
}

func TestPauseStopsTimer(t *testing.T) {
	resetWatchdogState(t)
	defer resetWatchdogState(t)

	StartWatchdog("test", 30*time.Millisecond)
	PauseWatchdog()
	time.Sleep(60 * time.Millisecond)

	if watchdogDisarmed.Load() {
		t.Fatal("paused watchdog must NOT fire (would have called os.Exit otherwise)")
	}
}

func TestResumeRearmsFullTimeout(t *testing.T) {
	resetWatchdogState(t)
	defer resetWatchdogState(t)

	StartWatchdog("test", 50*time.Millisecond)
	time.Sleep(40 * time.Millisecond)
	PauseWatchdog()
	time.Sleep(100 * time.Millisecond)
	ResumeWatchdog()

	watchdogMu.Lock()
	hasTimer := watchdogTimer != nil
	watchdogMu.Unlock()
	if !hasTimer {
		t.Fatal("Resume should arm a new timer")
	}
	if watchdogDisarmed.Load() {
		t.Fatal("Resume must not fire while still pending")
	}

	DisarmWatchdog()
}

func TestPauseResumeCycle(t *testing.T) {
	resetWatchdogState(t)
	defer resetWatchdogState(t)

	StartWatchdog("test", 100*time.Millisecond)
	for i := 0; i < 5; i++ {
		PauseWatchdog()
		time.Sleep(50 * time.Millisecond)
		ResumeWatchdog()
	}
	DisarmWatchdog()
	time.Sleep(150 * time.Millisecond)
}

func TestDisarmedPauseResumeAreNoops(t *testing.T) {
	resetWatchdogState(t)
	defer resetWatchdogState(t)

	StartWatchdog("test", 100*time.Millisecond)
	DisarmWatchdog()
	PauseWatchdog()
	ResumeWatchdog()

	watchdogMu.Lock()
	hasNewTimer := watchdogTimer != nil && !watchdogDisarmed.Load()
	watchdogMu.Unlock()
	if hasNewTimer {
		t.Fatal("Resume after Disarm must remain disarmed (no new timer firing)")
	}
}

func TestStartWithZeroTimeoutIsNoop(t *testing.T) {
	resetWatchdogState(t)
	defer resetWatchdogState(t)

	StartWatchdog("test", 0)
	watchdogMu.Lock()
	hasTimer := watchdogTimer != nil
	watchdogMu.Unlock()
	if hasTimer {
		t.Fatal("StartWatchdog with timeout<=0 must not arm a timer")
	}
}
