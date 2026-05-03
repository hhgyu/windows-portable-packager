package app

import (
	"path/filepath"
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
