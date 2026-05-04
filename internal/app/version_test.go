package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListInstalledVersionsEmpty(t *testing.T) {
	config := NewConfig("TestApp", "1.0.0", "amd64")
	config.AppDir = t.TempDir()

	versions, err := ListInstalledVersions(config)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Fatalf("expected empty, got %v", versions)
	}
}

func TestListInstalledVersions(t *testing.T) {
	config := NewConfig("TestApp", "", "")
	config.AppDir = t.TempDir()

	for _, v := range []string{"1.0.0", "2.0.0", "1.5.0"} {
		vDir := filepath.Join(config.AppDir, v)
		os.MkdirAll(vDir, 0755)
		m := &Manifest{Version: v, Exe: "app.exe", Files: map[string]FileEntry{}}
		m.Save(filepath.Join(vDir, ManifestName))
	}

	versions, err := ListInstalledVersions(config)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"1.0.0", "1.5.0", "2.0.0"}
	if len(versions) != len(expected) {
		t.Fatalf("count = %d, want %d", len(versions), len(expected))
	}
	for i, v := range expected {
		if versions[i] != v {
			t.Errorf("versions[%d] = %q, want %q", i, versions[i], v)
		}
	}
}

func TestListInstalledVersionsSkipsDirs(t *testing.T) {
	config := NewConfig("TestApp", "", "")
	config.AppDir = t.TempDir()

	vDir := filepath.Join(config.AppDir, "1.0.0")
	os.MkdirAll(vDir, 0755)
	m := &Manifest{Version: "1.0.0", Exe: "app.exe", Files: map[string]FileEntry{}}
	m.Save(filepath.Join(vDir, ManifestName))

	os.MkdirAll(filepath.Join(config.AppDir, "incomplete"), 0755)
	os.WriteFile(filepath.Join(config.AppDir, "notes.txt"), []byte("notes"), 0644)

	versions, err := ListInstalledVersions(config)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0] != "1.0.0" {
		t.Fatalf("expected [1.0.0], got %v", versions)
	}
}

func TestCleanOldVersions(t *testing.T) {
	config := NewConfig("TestApp", "", "")
	config.AppDir = t.TempDir()

	for _, v := range []string{"1.0.0", "2.0.0", "3.0.0"} {
		vDir := filepath.Join(config.AppDir, v)
		os.MkdirAll(vDir, 0755)
		m := &Manifest{Version: v, Exe: "app.exe", Files: map[string]FileEntry{}}
		m.Save(filepath.Join(vDir, ManifestName))
	}

	removed, err := CleanOldVersions(config, []string{"3.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2", removed)
	}

	remaining, _ := ListInstalledVersions(config)
	if len(remaining) != 1 || remaining[0] != "3.0.0" {
		t.Fatalf("expected [3.0.0] remaining, got %v", remaining)
	}
}

func TestCleanOldVersionsKeepAll(t *testing.T) {
	config := NewConfig("TestApp", "", "")
	config.AppDir = t.TempDir()

	for _, v := range []string{"1.0.0", "2.0.0"} {
		vDir := filepath.Join(config.AppDir, v)
		os.MkdirAll(vDir, 0755)
		m := &Manifest{Version: v, Exe: "app.exe", Files: map[string]FileEntry{}}
		m.Save(filepath.Join(vDir, ManifestName))
	}

	removed, err := CleanOldVersions(config, []string{"1.0.0", "2.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatalf("removed = %d, want 0", removed)
	}
}

func TestGetLatestVersion(t *testing.T) {
	config := NewConfig("TestApp", "", "")
	config.AppDir = t.TempDir()

	os.MkdirAll(filepath.Join(config.AppDir, "1.0.0", "sub"), 0755)
	m1 := &Manifest{Version: "1.0.0", Exe: "app.exe", Files: map[string]FileEntry{}}
	m1.Save(filepath.Join(config.AppDir, "1.0.0", ManifestName))

	os.MkdirAll(filepath.Join(config.AppDir, "2.0.0", "sub"), 0755)
	m2 := &Manifest{Version: "2.0.0", Exe: "app.exe", Files: map[string]FileEntry{}}
	m2.Save(filepath.Join(config.AppDir, "2.0.0", ManifestName))

	// GetLatestVersion picks the manifest with the newest mtime. NTFS mtime
	// resolution can collapse two near-instant writes to the same value, so
	// pin explicit timestamps to keep the test deterministic.
	older := time.Now().Add(-2 * time.Hour)
	newer := time.Now().Add(-1 * time.Hour)
	if err := os.Chtimes(filepath.Join(config.AppDir, "1.0.0", ManifestName), older, older); err != nil {
		t.Fatalf("chtimes 1.0.0: %v", err)
	}
	if err := os.Chtimes(filepath.Join(config.AppDir, "2.0.0", ManifestName), newer, newer); err != nil {
		t.Fatalf("chtimes 2.0.0: %v", err)
	}

	latest, err := GetLatestVersion(config)
	if err != nil {
		t.Fatal(err)
	}
	if latest != "2.0.0" {
		t.Errorf("latest = %q, want %q", latest, "2.0.0")
	}
}

func TestGetLatestVersionEmpty(t *testing.T) {
	config := NewConfig("TestApp", "", "")
	config.AppDir = t.TempDir()

	_, err := GetLatestVersion(config)
	if err == nil {
		t.Fatal("expected error for empty app dir")
	}
}
