package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// applySplashMin pushes the manifest's SplashMinMs onto the splash window so
// that a successful Close honours it. SetMinVisible is a no-op when zero so
// callers do not need to branch on whether the manifest set the field.
func applySplashMin(splash *SplashWindow, m *Manifest) {
	if splash == nil || m == nil {
		return
	}
	splash.SetMinVisible(time.Duration(m.SplashMinMs) * time.Millisecond)
}

func Run(pkgPath, exeOverride, splashOverride string) error {
	var splash *SplashWindow
	if splashOverride != "" {
		splash, _ = ShowSplash(splashOverride)
	} else if HasEmbeddedSplash() {
		data, ext := GetEmbeddedSplash()
		splash, _ = ShowSplashFromData(data, ext)
	}

	var err error
	if HasEmbeddedPackage() {
		err = runFromEmbedded(exeOverride, splash)
	} else {
		err = runFromFile(pkgPath, exeOverride, splash)
	}
	return err
}

func runFromEmbedded(exeOverride string, splash *SplashWindow) error {
	manifest, err := ReadEmbeddedManifest()
	if err != nil {
		splash.ForceClose()
		return fmt.Errorf("read embedded package: %w", err)
	}
	applySplashMin(splash, manifest)
	ConfigureLockDetect(manifest.LenientLockDetect)
	LogVerbose(fmt.Sprintf("Package: %s %s (%s)", manifest.AppName, manifest.Version, manifest.Arch))

	config := NewConfig(manifest.AppName, manifest.Version, manifest.Arch)

	installedManifest, err := LoadManifest(config.ManifestPath())
	if err == nil {
		exeName := manifest.Exe
		if exeOverride != "" {
			exeName = exeOverride
		}
		if !manifest.EqualForInstall(installedManifest) {
			LogVerbose(T(MsgInstalledContentChanged))
		} else {
			mismatches, verifyErr := manifest.Verify(config.VersionDir)
			if verifyErr == nil && len(mismatches) == 0 {
				LogVerbose(T(MsgAlreadyInstalled))
				CleanOldVersions(config, []string{manifest.Version})
				err := launch(filepath.Join(config.VersionDir, exeName))
				if err != nil {
					splash.ForceClose()
				} else {
					splash.Close()
				}
				return err
			}
		}
	}

	Log(fmt.Sprintf(T(MsgInstalling), manifest.AppName, manifest.Version))
	LogVerbose(fmt.Sprintf(T(MsgExtracting), config.VersionDir))
	extracted, err := UnpackEmbedded(config.VersionDir)
	if err != nil {
		splash.ForceClose()
		return fmt.Errorf("unpack: %w", err)
	}

	CleanOldVersions(config, []string{extracted.Version})

	exePath := filepath.Join(config.VersionDir, extracted.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	err = launch(exePath)
	if err != nil {
		splash.ForceClose()
	} else {
		splash.Close()
	}
	return err
}

func runFromFile(pkgPath, exeOverride string, splash *SplashWindow) error {
	pkg, err := FindPackage(pkgPath)
	if err != nil {
		LogVerbose(T(MsgNoPackageFound))
		return launchLatest(exeOverride, splash)
	}

	manifest, err := ReadPackageManifest(pkg)
	if err != nil {
		splash.ForceClose()
		return fmt.Errorf("read package: %w", err)
	}
	applySplashMin(splash, manifest)
	ConfigureLockDetect(manifest.LenientLockDetect)
	LogVerbose(fmt.Sprintf("Package: %s %s (%s)", manifest.AppName, manifest.Version, manifest.Arch))

	config := NewConfig(manifest.AppName, manifest.Version, manifest.Arch)

	installedManifest, err := LoadManifest(config.ManifestPath())
	if err == nil {
		exeName := manifest.Exe
		if exeOverride != "" {
			exeName = exeOverride
		}
		if !manifest.EqualForInstall(installedManifest) {
			LogVerbose(T(MsgInstalledContentChanged))
		} else {
			mismatches, verifyErr := manifest.Verify(config.VersionDir)
			if verifyErr == nil && len(mismatches) == 0 {
				LogVerbose(T(MsgAlreadyInstalled))
				CleanOldVersions(config, []string{manifest.Version})
				err := launch(filepath.Join(config.VersionDir, exeName))
				if err != nil {
					splash.ForceClose()
				} else {
					splash.Close()
				}
				return err
			}
		}
	}

	Log(fmt.Sprintf(T(MsgInstalling), manifest.AppName, manifest.Version))
	LogVerbose(fmt.Sprintf(T(MsgExtracting), config.VersionDir))
	extracted, err := Unpack(pkg, config.VersionDir)
	if err != nil {
		splash.ForceClose()
		return fmt.Errorf("unpack: %w", err)
	}

	CleanOldVersions(config, []string{extracted.Version})

	exePath := filepath.Join(config.VersionDir, extracted.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	err = launch(exePath)
	if err != nil {
		splash.ForceClose()
	} else {
		splash.Close()
	}
	return err
}

func launchLatest(exeOverride string, splash *SplashWindow) error {
	config := NewConfig("", "", DetectArch())
	latest, err := GetLatestVersion(config)
	if err != nil {
		splash.ForceClose()
		return fmt.Errorf("no installed version found — place a %s file next to this executable", PackageExt)
	}

	config.Version = latest
	manifest, err := LoadManifest(config.ManifestPath())
	if err != nil {
		splash.ForceClose()
		return fmt.Errorf("load manifest for %s: %w", latest, err)
	}
	applySplashMin(splash, manifest)

	exePath := filepath.Join(config.VersionDir, manifest.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	err = launch(exePath)
	if err != nil {
		splash.ForceClose()
	} else {
		splash.Close()
	}
	return err
}

func launch(exePath string) error {
	if _, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("exe not found: %s", exePath)
	}

	LogVerbose(fmt.Sprintf(T(MsgLaunching), filepath.Base(exePath)))

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	setSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
