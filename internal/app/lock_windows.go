//go:build windows

package app

import (
	"os"
	"path/filepath"
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

		f, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				locked = append(locked, path)
			} else {
				locked = append(locked, filepath.ToSlash(rel))
			}
			return nil
		}
		f.Close()
		return nil
	})

	return locked, err
}
