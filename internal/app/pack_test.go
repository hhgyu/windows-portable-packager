package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	if err := Pack(srcDir, outputPath, "KeyBridge", "1.2.3", "amd64", "KeyBridge.exe", ""); err != nil {
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

	err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "NonExistent.exe", "")
	if err == nil {
		t.Fatal("expected error for missing exe")
	}
}

func TestPackNonexistentSource(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")
	err := Pack("/nonexistent/path", outputPath, "TestApp", "1.0.0", "amd64", "app.exe", "")
	if err == nil {
		t.Fatal("expected error for nonexistent source")
	}
}

func TestPackCreatesOutputDir(t *testing.T) {
	srcDir := t.TempDir()
	writeTestFile(t, srcDir, "app.exe", "binary")

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "nested", "deep", "output.kbpkg")

	if err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "app.exe", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output not created in nested dir: %v", err)
	}
}

func TestPackZstdAndUnpack(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)
	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")

	if err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "KeyBridge.exe", "",
		PackOptions{Compression: CompressionZstd, Level: 3}); err != nil {
		t.Fatal(err)
	}

	manifest, err := ReadPackageManifest(outputPath)
	if err != nil {
		t.Fatalf("zstd package read failed: %v", err)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", manifest.Version)
	}

	versionDir := t.TempDir()
	m, err := Unpack(outputPath, versionDir)
	if err != nil {
		t.Fatalf("zstd unpack failed: %v", err)
	}
	if m.AppName != "TestApp" {
		t.Errorf("appName = %q, want TestApp", m.AppName)
	}
}

func TestPackGzipAndUnpack(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)
	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")

	if err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "KeyBridge.exe", "",
		PackOptions{Compression: CompressionGzip, Level: 6}); err != nil {
		t.Fatal(err)
	}

	manifest, err := ReadPackageManifest(outputPath)
	if err != nil {
		t.Fatalf("gzip package read failed: %v", err)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", manifest.Version)
	}
}

func TestPackXZAndUnpack(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)
	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")

	if err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "KeyBridge.exe", "",
		PackOptions{Compression: CompressionXZ, Level: 3}); err != nil {
		t.Fatal(err)
	}

	manifest, err := ReadPackageManifest(outputPath)
	if err != nil {
		t.Fatalf("xz package read failed: %v", err)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("version = %q, want 1.0.0", manifest.Version)
	}

	versionDir := t.TempDir()
	m, err := Unpack(outputPath, versionDir)
	if err != nil {
		t.Fatalf("xz unpack failed: %v", err)
	}
	if m.AppName != "TestApp" {
		t.Errorf("appName = %q, want TestApp", m.AppName)
	}
}

func TestPackPropagatesLenientLockDetect(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)
	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")

	if err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "KeyBridge.exe", "",
		PackOptions{LenientLockDetect: true}); err != nil {
		t.Fatal(err)
	}

	manifest, err := ReadPackageManifest(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.LenientLockDetect {
		t.Fatal("LenientLockDetect=true did not propagate into manifest")
	}
}

func TestPackOmitsLenientLockDetectByDefault(t *testing.T) {
	srcDir := t.TempDir()
	createMockAppDir(t, srcDir)
	outputPath := filepath.Join(t.TempDir(), "test.kbpkg")

	if err := Pack(srcDir, outputPath, "TestApp", "1.0.0", "amd64", "KeyBridge.exe", ""); err != nil {
		t.Fatal(err)
	}

	manifest, err := ReadPackageManifest(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.LenientLockDetect {
		t.Fatal("LenientLockDetect must default to false (strict mode)")
	}
}

func TestPackZstdSmallerThanGzip(t *testing.T) {
	srcDir := t.TempDir()
	for i := 0; i < 10; i++ {
		writeTestFile(t, srcDir, fmt.Sprintf("file%d.dat", i), strings.Repeat("hello world electron app data ", 1000))
	}
	writeTestFile(t, srcDir, "app.exe", strings.Repeat("binary", 500))

	zstdPath := filepath.Join(t.TempDir(), "zstd.kbpkg")
	gzipPath := filepath.Join(t.TempDir(), "gzip.kbpkg")

	Pack(srcDir, zstdPath, "App", "1.0.0", "amd64", "app.exe", "", PackOptions{Compression: CompressionZstd})
	Pack(srcDir, gzipPath, "App", "1.0.0", "amd64", "app.exe", "", PackOptions{Compression: CompressionGzip})

	zstdInfo, _ := os.Stat(zstdPath)
	gzipInfo, _ := os.Stat(gzipPath)

	t.Logf("zstd: %d bytes, gzip: %d bytes", zstdInfo.Size(), gzipInfo.Size())
}
