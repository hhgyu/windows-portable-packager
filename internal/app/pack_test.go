package app

import (
	"os"
	"path/filepath"
	"testing"
)

func createMockAppDir(t *testing.T, dir string) {
	t.Helper()
	writeTestFile(t, dir, "KeyBridge.exe", "fake binary content for testing")
	writeTestFile(t, dir, "resources/app.asar", "asar archive data")
	writeTestFile(t, dir, "resources/icon.png", "png icon data")
	writeTestFile(t, dir, "locales/en-US.pak", "locale pack")
}

func TestPackAndReadManifest(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)

	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")

	if err := Pack(srcDir, outputPath, "KeyBridge", "1.2.3", "amd64", "KeyBridge.exe"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	manifest, err := ReadPackageManifest(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	if manifest.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", manifest.Version, "1.2.3")
	}
	if manifest.Exe != "KeyBridge.exe" {
		t.Errorf("exe = %q, want %q", manifest.Exe, "KeyBridge.exe")
	}
	if len(manifest.Files) != 4 {
		t.Fatalf("files count = %d, want 4", len(manifest.Files))
	}
}

func TestPackMissingExe(t *testing.T) {
	srcDir := t.TempDir()
	writeTestFile(t, srcDir, "other.txt", "data")

	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")

	err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "NonExistent.exe")
	if err == nil {
		t.Fatal("expected error for missing exe")
	}
}

func TestPackNonexistentSource(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")
	err := Pack("/nonexistent/path", outputPath, "TestApp", "1.0.0", "amd64", "app.exe")
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestPackCreatesOutputDir(t *testing.T) {
	srcDir := t.TempDir()
	writeTestFile(t, srcDir, "app.exe", "binary")

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "nested", "deep", "output.kbpkg")

	if err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "app.exe"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output not created in nested dir: %v", err)
	}
}
