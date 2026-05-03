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
