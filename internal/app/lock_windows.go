//go:build windows

package app

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func detectLockedFiles(dir string) ([]string, error) {
	var locked []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if isFileLocked(path) {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				locked = append(locked, path)
			} else {
				locked = append(locked, filepath.ToSlash(rel))
			}
		}
		return nil
	})

	return locked, err
}

func isFileLocked(path string) bool {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}

	access := uint32(windows.GENERIC_READ)
	share := uint32(0)
	if isLockDetectLenient() {
		access = windows.DELETE
		share = windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE | windows.FILE_SHARE_DELETE
	}

	h, err := windows.CreateFile(
		p,
		access,
		share,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		return errors.Is(err, windows.ERROR_SHARING_VIOLATION)
	}
	windows.CloseHandle(h)
	return false
}
