package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUnpackBasic(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)

	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")
	if err := Pack(srcDir, outputPath, "KeyBridge", "3.0.0", "amd64", "KeyBridge.exe"); err != nil {
		t.Fatal(err)
	}

	versionDir := filepath.Join(t.TempDir(), "app", "3.0.0")
	manifest, err := Unpack(outputPath, versionDir)
	if err != nil {
		t.Fatalf("Unpack failed: %v", err)
	}

	if manifest.Version != "3.0.0" {
		t.Errorf("version = %q, want %q", manifest.Version, "3.0.0")
	}

	expectedFiles := []string{
		"KeyBridge.exe",
		"resources/app.asar",
		"resources/icon.png",
		"locales/en-US.pak",
		ManifestName,
	}

	for _, f := range expectedFiles {
		p := filepath.Join(versionDir, filepath.FromSlash(f))
		if _, err := os.Stat(p); err != nil {
			t.Errorf("file not extracted: %s (%v)", f, err)
		}
	}
}

func TestUnpackAndVerify(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)

	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")
	if err := Pack(srcDir, outputPath, "KeyBridge", "1.0.0", "amd64", "KeyBridge.exe"); err != nil {
		t.Fatal(err)
	}

	versionDir := filepath.Join(t.TempDir(), "app", "1.0.0")
	manifest, err := Unpack(outputPath, versionDir)
	if err != nil {
		t.Fatal(err)
	}

	mismatches, err := manifest.Verify(versionDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mismatches) != 0 {
		t.Fatalf("post-unpack verification failed: %v", mismatches)
	}
}

func TestUnpackRemovesPreviousInstall(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)

	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")
	if err := Pack(srcDir, outputPath, "KeyBridge", "1.0.0", "amd64", "KeyBridge.exe"); err != nil {
		t.Fatal(err)
	}

	versionDir := filepath.Join(t.TempDir(), "app", "1.0.0")

	// Create a stale file that should be removed
	staleFile := filepath.Join(versionDir, "stale.dat")
	os.MkdirAll(versionDir, 0755)
	os.WriteFile(staleFile, []byte("old"), 0644)

	_, err := Unpack(outputPath, versionDir)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(staleFile); err == nil {
		t.Error("stale file should have been removed")
	}
}

func TestUnpackRejectsTamperedPackage(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)

	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")
	if err := Pack(srcDir, outputPath, "KeyBridge", "1.0.0", "amd64", "KeyBridge.exe"); err != nil {
		t.Fatal(err)
	}



	versionDir := filepath.Join(t.TempDir(), "app", "1.0.0")
	manifest, err := Unpack(outputPath, versionDir)
	if err != nil {
		t.Fatal(err)
	}

	writeTestFile(t, versionDir, "KeyBridge.exe", "TAMPERED")

	mismatches, err := manifest.Verify(versionDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(mismatches) == 0 {
		t.Fatal("tampered file should be detected")
	}
}

func TestReadPackageManifestInvalidFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "invalid.kbpkg")
	os.WriteFile(tmpFile, []byte("not a tar.gz"), 0644)

	_, err := ReadPackageManifest(tmpFile)
	if err == nil {
		t.Fatal("expected error for invalid package file")
	}
}
