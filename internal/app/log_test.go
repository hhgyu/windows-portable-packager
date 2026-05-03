package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func captureLog(fn func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestLogNormalLevel(t *testing.T) {
	SetLogLevel(LogLevelNormal)
	defer SetLogLevel(LogLevelNormal)

	out := captureLog(func() {
		Log("hello %s", "world")
	})
	if !strings.Contains(out, "hello world") {
		t.Errorf("expected 'hello world' in output, got: %q", out)
	}
}

func TestLogVerboseSuppressedAtNormal(t *testing.T) {
	SetLogLevel(LogLevelNormal)
	defer SetLogLevel(LogLevelNormal)

	out := captureLog(func() {
		LogVerbose("secret verbose")
	})
	if out != "" {
		t.Errorf("expected no output at normal level, got: %q", out)
	}
}

func TestLogVerboseShownAtVerboseLevel(t *testing.T) {
	SetLogLevel(LogLevelVerbose)
	defer SetLogLevel(LogLevelNormal)

	out := captureLog(func() {
		LogVerbose("detail info")
	})
	if !strings.Contains(out, "detail info") {
		t.Errorf("expected 'detail info' in verbose output, got: %q", out)
	}
	if !strings.Contains(out, "[verbose]") {
		t.Errorf("expected '[verbose]' prefix in output, got: %q", out)
	}
}

func TestLogError(t *testing.T) {
	out := captureLog(func() {
		LogError("something went wrong: %v", fmt.Errorf("disk full"))
	})
	if !strings.Contains(out, "Error:") {
		t.Errorf("expected 'Error:' prefix, got: %q", out)
	}
	if !strings.Contains(out, "disk full") {
		t.Errorf("expected error message in output, got: %q", out)
	}
}
