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
		LogVerbose(fmt.Sprintf(T(MsgRemovingOldVersion), version))

		if err := removeWithRetry(config.AppName, version, versionDir); err != nil {
			Log(fmt.Sprintf(T(MsgFailedToRemove), version, err))
			continue
		}
		removed++
	}

	return removed, nil
}

func removeWithRetry(appName, version, versionDir string) error {
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		err := os.RemoveAll(versionDir)
		if err == nil {
			return nil
		}

		if i == maxRetries-1 {
			return err
		}

		title := fmt.Sprintf(T(MsgRetryTitle), appName)
		message := fmt.Sprintf(T(MsgRetryBody), version)
		if IsTerminal() {
			Log(fmt.Sprintf(T(MsgOldVersionInUse), version, i+1, maxRetries-1))
			time.Sleep(3 * time.Second)
		} else {
			if !ShowRetryDialog(title, message) {
				return err
			}
		}
	}
	return nil
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
