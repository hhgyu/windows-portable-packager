package app

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestComputeFileHash(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(f, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash, err := ComputeFileHash(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(hash) != 16 {
		t.Fatalf("expected 16-char hex hash, got %d chars", len(hash))
	}

	expected := "45ab6734b21e6968"
	if hash != expected {
		t.Fatalf("hash mismatch:\n  got:  %s\n  want: %s", hash, expected)
	}
}

func TestComputeFileHashNonexistent(t *testing.T) {
	_, err := ComputeFileHash("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestGenerateManifest(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "app.exe", "binary content")
	writeTestFile(t, dir, "resources/app.asar", "asar content")
	writeTestFile(t, dir, "locales/en.pak", "locale data")

	manifest, err := GenerateManifest(dir, "TestApp", "1.0.0", "amd64", "app.exe", "")
	if err != nil {
		t.Fatal(err)
	}

	if manifest.AppName != "TestApp" {
		t.Errorf("appName = %q, want %q", manifest.AppName, "TestApp")
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("version = %q, want %q", manifest.Version, "1.0.0")
	}
	if manifest.Exe != "app.exe" {
		t.Errorf("exe = %q, want %q", manifest.Exe, "app.exe")
	}
	if len(manifest.Files) != 3 {
		t.Fatalf("files count = %d, want 3", len(manifest.Files))
	}

	for _, name := range []string{"app.exe", "resources/app.asar", "locales/en.pak"} {
		entry, ok := manifest.Files[name]
		if !ok {
			t.Errorf("missing file in manifest: %s", name)
			continue
		}
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if entry.Size != info.Size() {
			t.Errorf("file %q size = %d, want %d", name, entry.Size, info.Size())
		}
	}

	if manifest.Timestamp == "" {
		t.Error("timestamp is empty")
	}
}

func TestManifestSaveLoad(t *testing.T) {
	dir := t.TempDir()

	original := &Manifest{
		AppName:   "MyApp",
		Version:   "2.0.0",
		Arch:      "arm64",
		Exe:       "MyApp.exe",
		Timestamp: "2026-01-01T00:00:00Z",
		Files: map[string]FileEntry{
			"MyApp.exe":          {Hash: "abc123", Size: 12},
			"resources/app.asar": {Hash: "def456", Size: 34},
		},
	}

	path := filepath.Join(dir, ManifestName)
	if err := original.Save(path); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.AppName != original.AppName {
		t.Errorf("appName mismatch: %q vs %q", loaded.AppName, original.AppName)
	}
	if loaded.Version != original.Version {
		t.Errorf("version mismatch: %q vs %q", loaded.Version, original.Version)
	}
	if loaded.Arch != original.Arch {
		t.Errorf("arch mismatch: %q vs %q", loaded.Arch, original.Arch)
	}
	if loaded.Exe != original.Exe {
		t.Errorf("exe mismatch: %q vs %q", loaded.Exe, original.Exe)
	}
	if len(loaded.Files) != len(original.Files) {
		t.Fatalf("files count mismatch: %d vs %d", len(loaded.Files), len(original.Files))
	}
	for k, v := range original.Files {
		if loaded.Files[k] != v {
			t.Errorf("file %q entry mismatch: %#v vs %#v", k, loaded.Files[k], v)
		}
	}
}

func TestManifestVerify(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "aaa")
	writeTestFile(t, dir, "b.txt", "bbb")

	manifest, err := GenerateManifest(dir, "TestApp", "1.0.0", "amd64", "a.txt", "")
	if err != nil {
		t.Fatal(err)
	}

	mismatches, err := manifest.Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mismatches) != 0 {
		t.Fatalf("unexpected mismatches: %v", mismatches)
	}
}

func TestManifestVerifyDetectsTamper(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "original")

	manifest, err := GenerateManifest(dir, "TestApp", "1.0.0", "amd64", "a.txt", "")
	if err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, dir, "a.txt", "tampered")

	mismatches, err := manifest.Verify(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mismatches) != 1 {
		t.Fatalf("expected 1 mismatch, got %d: %v", len(mismatches), mismatches)
	}
	if mismatches[0] != "a.txt" {
		t.Errorf("mismatch file = %q, want %q", mismatches[0], "a.txt")
	}
}

func TestManifestVerifyMissingFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "aaa")

	manifest, err := GenerateManifest(dir, "TestApp", "1.0.0", "amd64", "a.txt", "")
	if err != nil {
		t.Fatal(err)
	}

	os.Remove(filepath.Join(dir, "a.txt"))

	_, err = manifest.Verify(dir)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestManifestVerifySingle(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "good.txt", "good")
	writeTestFile(t, dir, "bad.txt", "original")

	manifest, err := GenerateManifest(dir, "TestApp", "1.0.0", "amd64", "good.txt", "")
	if err != nil {
		t.Fatal(err)
	}

	ok, err := manifest.VerifySingle(dir, "good.txt")
	if err != nil || !ok {
		t.Fatalf("good.txt should verify: ok=%v err=%v", ok, err)
	}

	ok, err = manifest.VerifySingle(dir, "nonexistent.txt")
	if err == nil {
		t.Fatal("expected error for file not in manifest")
	}

	writeTestFile(t, dir, "bad.txt", "tampered")
	ok, err = manifest.VerifySingle(dir, "bad.txt")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("tampered file should not verify")
	}
}
