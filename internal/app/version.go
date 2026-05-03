package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func ListInstalledVersions(config *Config) ([]string, error) {
	entries, err := os.ReadDir(config.AppDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var versions []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(config.AppDir, entry.Name(), ManifestName)
		if _, err := os.Stat(manifestPath); err == nil {
			versions = append(versions, entry.Name())
		}
	}

	sort.Strings(versions)
	return versions, nil
}

func CleanOldVersions(config *Config, keepVersions []string) (int, error) {
	installed, err := ListInstalledVersions(config)
	if err != nil {
		return 0, err
	}

	keepMap := make(map[string]bool, len(keepVersions))
	for _, v := range keepVersions {
		keepMap[v] = true
	}

	removed := 0
	for _, version := range installed {
		if keepMap[version] {
			continue
		}

		versionDir := filepath.Join(config.AppDir, version)
		fmt.Printf("Removing old version: %s\n", version)
		if err := os.RemoveAll(versionDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to remove %s: %v\n", version, err)
			continue
		}
		removed++
	}

	return removed, nil
}

func GetLatestVersion(config *Config) (string, error) {
	entries, err := os.ReadDir(config.AppDir)
	if err != nil {
		return "", err
	}

	var latest string
	var latestMod time.Time

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(config.AppDir, entry.Name(), ManifestName)
		info, err := os.Stat(manifestPath)
		if err != nil {
			continue
		}

		if info.ModTime().After(latestMod) {
			latestMod = info.ModTime()
			latest = entry.Name()
		}
	}

	if latest == "" {
		return "", fmt.Errorf("no installed version found in %s", config.AppDir)
	}

	return latest, nil
}
