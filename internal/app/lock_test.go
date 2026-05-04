package app

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestDetectLockedFilesUnlocked(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "app.exe", "ok")
	writeTestFile(t, dir, "resources/app.asar", "ok")

	locked, err := detectLockedFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(locked) != 0 {
		t.Fatalf("expected no locked files, got %v", locked)
	}
}

func TestWaitForVersionDirUnlocked(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, filepath.Join("resources", "app.asar"), "ok")

	if err := waitForVersionDirUnlocked(dir); err != nil {
		t.Fatal(err)
	}
}

func TestLockedExecutablesFiltersExe(t *testing.T) {
	input := []string{
		"resources/app.asar.unpacked/node_modules/uiohook-napi/build/Release/uiohook_napi.node",
		"KeyBridge.exe",
		"chrome_100_percent.pak",
		"d3dcompiler_47.dll",
		"resources/electron.exe",
	}
	want := []string{"KeyBridge.exe", "resources/electron.exe"}
	got := lockedExecutables(input)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lockedExecutables = %v, want %v", got, want)
	}
}

func TestLockedExecutablesNoExeReturnsEmpty(t *testing.T) {
	input := []string{"a.dll", "b.pak", "c.node"}
	got := lockedExecutables(input)
	if len(got) != 0 {
		t.Fatalf("expected empty, got %v", got)
	}
}

func TestConfigureLockDetectTogglesLenient(t *testing.T) {
	defer ConfigureLockDetect(false)

	ConfigureLockDetect(true)
	if !isLockDetectLenient() {
		t.Fatal("ConfigureLockDetect(true) must enable lenient mode")
	}

	ConfigureLockDetect(false)
	if isLockDetectLenient() {
		t.Fatal("ConfigureLockDetect(false) must disable lenient mode")
	}
}
