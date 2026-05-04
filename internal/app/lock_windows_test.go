//go:build windows

package app

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestDetectLockedFilesLocked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.exe")
	if err := os.WriteFile(path, []byte("locked"), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)

	locked, err := detectLockedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked) != 1 || locked[0] != "app.exe" {
		t.Fatalf("locked = %v, want [app.exe]", locked)
	}
}

func TestDetectLockedFilesReadOnlyNotLocked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "resource.pak")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}

	locked, err := detectLockedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked) != 0 {
		t.Fatalf("read-only file falsely reported as locked: %v", locked)
	}
}

func TestDetectLockedFilesSharedReadIsLocked(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "running.exe")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(
		p,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)

	locked, err := detectLockedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked) != 1 || locked[0] != "running.exe" {
		t.Fatalf("running.exe held by another handle should be locked, got %v", locked)
	}
}

func TestDetectLockedFilesAsarLockedExeFree(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "KeyBridge.exe")
	if err := os.WriteFile(exePath, []byte("exe"), 0644); err != nil {
		t.Fatal(err)
	}
	asarPath := filepath.Join(dir, "resources", "app.asar")
	if err := os.MkdirAll(filepath.Dir(asarPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(asarPath, []byte("asar"), 0644); err != nil {
		t.Fatal(err)
	}

	p, err := windows.UTF16PtrFromString(asarPath)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(
		p,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(h)

	locked, err := detectLockedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked) != 1 || locked[0] != "resources/app.asar" {
		t.Fatalf("expected only resources/app.asar locked, got %v", locked)
	}
}

func TestDetectLockedFilesLenientSkipsRunningExe(t *testing.T) {
	defer ConfigureLockDetect(false)
	ConfigureLockDetect(true)

	dir := t.TempDir()
	exePath := filepath.Join(dir, "KeyBridge.exe")
	if err := os.WriteFile(exePath, []byte("exe"), 0644); err != nil {
		t.Fatal(err)
	}

	exePtr, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		t.Fatal(err)
	}
	exeHandle, err := windows.CreateFile(
		exePtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(exeHandle)

	locked, err := detectLockedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked) != 0 {
		t.Fatalf("lenient probe should not catch PE-loader-style .exe, got %v", locked)
	}
}

func TestDetectLockedFilesPELoaderEmulation(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "KeyBridge.exe")
	if err := os.WriteFile(exePath, []byte("exe"), 0644); err != nil {
		t.Fatal(err)
	}
	asarPath := filepath.Join(dir, "resources", "app.asar")
	if err := os.MkdirAll(filepath.Dir(asarPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(asarPath, []byte("asar"), 0644); err != nil {
		t.Fatal(err)
	}

	exePtr, err := windows.UTF16PtrFromString(exePath)
	if err != nil {
		t.Fatal(err)
	}
	exeHandle, err := windows.CreateFile(
		exePtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_DELETE,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(exeHandle)

	asarPtr, err := windows.UTF16PtrFromString(asarPath)
	if err != nil {
		t.Fatal(err)
	}
	asarHandle, err := windows.CreateFile(
		asarPtr,
		windows.GENERIC_READ,
		windows.FILE_SHARE_READ,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(asarHandle)

	locked, err := detectLockedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked) != 2 {
		t.Fatalf("expected both .exe and .asar locked, got %v", locked)
	}

	exes := lockedExecutables(locked)
	if len(exes) != 1 || exes[0] != "KeyBridge.exe" {
		t.Fatalf("lockedExecutables should isolate KeyBridge.exe, got %v", exes)
	}
}
