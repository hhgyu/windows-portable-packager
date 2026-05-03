package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	ManifestName = "_manifest.json"
	PackageExt   = ".kbpkg"
)

type Config struct {
	AppName    string
	Version    string
	Arch       string
	AppDataDir string
	AppDir     string
	VersionDir string
	ExeName    string
}

func NewConfig(appName, version, arch string) *Config {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}

	base := filepath.Join(appData, appName)
	appDir := filepath.Join(base, "app")

	return &Config{
		AppName:    appName,
		Version:    version,
		Arch:       arch,
		AppDataDir: base,
		AppDir:     appDir,
		VersionDir: filepath.Join(appDir, version),
		ExeName:    appName + ".exe",
	}
}

func (c *Config) ManifestPath() string {
	return filepath.Join(c.VersionDir, ManifestName)
}

func (c *Config) ExePath() string {
	return filepath.Join(c.VersionDir, c.ExeName)
}

func FindPackage(customPath string) (string, error) {
	if customPath != "" {
		if _, err := os.Stat(customPath); err != nil {
			return "", fmt.Errorf("package not found: %s: %w", customPath, err)
		}
		return customPath, nil
	}

	exe, err := os.Executable()
	if err != nil {
		return "", err
	}

	dir := filepath.Dir(exe)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(name), PackageExt) {
			return filepath.Join(dir, name), nil
		}
	}

	return "", fmt.Errorf("no %s file found next to %s", PackageExt, exe)
}

func DetectArch() string {
	return runtime.GOARCH
}
