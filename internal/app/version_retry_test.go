package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveWithRetrySuccess(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "v1.0.0")
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("data"), 0644)

	err := removeWithRetry("TestApp", "v1.0.0", dir)
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("expected directory to be removed")
	}
}

func TestRemoveWithRetryNonExistent(t *testing.T) {
	err := removeWithRetry("TestApp", "v0.0.0", "/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Fatalf("RemoveAll on nonexistent path should succeed, got: %v", err)
	}
}

func TestCleanOldVersionsWithRetry(t *testing.T) {
	tmp := t.TempDir()
	appDir := filepath.Join(tmp, "app")

	for _, v := range []string{"1.0.0", "1.1.0", "1.2.0"} {
		dir := filepath.Join(appDir, v)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, ManifestName), []byte("{}"), 0644)
	}

	config := &Config{AppName: "TestApp", AppDir: appDir}
	removed, err := CleanOldVersions(config, []string{"1.2.0"})
	if err != nil {
		t.Fatalf("CleanOldVersions error: %v", err)
	}
	if removed != 2 {
		t.Errorf("expected 2 removed, got %d", removed)
	}

	if _, err := os.Stat(filepath.Join(appDir, "1.2.0")); os.IsNotExist(err) {
		t.Error("kept version 1.2.0 should still exist")
	}
	if _, err := os.Stat(filepath.Join(appDir, "1.0.0")); !os.IsNotExist(err) {
		t.Error("old version 1.0.0 should be removed")
	}
	if _, err := os.Stat(filepath.Join(appDir, "1.1.0")); !os.IsNotExist(err) {
		t.Error("old version 1.1.0 should be removed")
	}
}
