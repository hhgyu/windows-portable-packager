package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

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
		splash.Close()
		return fmt.Errorf("read embedded package: %w", err)
	}
	LogVerbose(fmt.Sprintf("Package: %s %s (%s)", manifest.AppName, manifest.Version, manifest.Arch))

	config := NewConfig(manifest.AppName, manifest.Version, manifest.Arch)

	installedManifest, err := LoadManifest(config.ManifestPath())
	if err == nil {
		exeName := manifest.Exe
		if exeOverride != "" {
			exeName = exeOverride
		}
		ok, verifyErr := installedManifest.VerifySingle(config.VersionDir, exeName)
		if verifyErr == nil && ok {
			LogVerbose(T(MsgAlreadyInstalled))
			CleanOldVersions(config, []string{manifest.Version})
			err := launch(filepath.Join(config.VersionDir, exeName))
			splash.Close()
			return err
		}
	}

	Log(fmt.Sprintf(T(MsgInstalling), manifest.AppName, manifest.Version))
	LogVerbose(fmt.Sprintf(T(MsgExtracting), config.VersionDir))
	extracted, err := UnpackEmbedded(config.VersionDir)
	if err != nil {
		splash.Close()
		return fmt.Errorf("unpack: %w", err)
	}

	CleanOldVersions(config, []string{extracted.Version})

	exePath := filepath.Join(config.VersionDir, extracted.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	err = launch(exePath)
	splash.Close()
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
		splash.Close()
		return fmt.Errorf("read package: %w", err)
	}
	LogVerbose(fmt.Sprintf("Package: %s %s (%s)", manifest.AppName, manifest.Version, manifest.Arch))

	config := NewConfig(manifest.AppName, manifest.Version, manifest.Arch)

	installedManifest, err := LoadManifest(config.ManifestPath())
	if err == nil {
		exeName := manifest.Exe
		if exeOverride != "" {
			exeName = exeOverride
		}
		ok, verifyErr := installedManifest.VerifySingle(config.VersionDir, exeName)
		if verifyErr == nil && ok {
			LogVerbose(T(MsgAlreadyInstalled))
			CleanOldVersions(config, []string{manifest.Version})
			err := launch(filepath.Join(config.VersionDir, exeName))
			splash.Close()
			return err
		}
	}

	Log(fmt.Sprintf(T(MsgInstalling), manifest.AppName, manifest.Version))
	LogVerbose(fmt.Sprintf(T(MsgExtracting), config.VersionDir))
	extracted, err := Unpack(pkg, config.VersionDir)
	if err != nil {
		splash.Close()
		return fmt.Errorf("unpack: %w", err)
	}

	CleanOldVersions(config, []string{extracted.Version})

	exePath := filepath.Join(config.VersionDir, extracted.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	err = launch(exePath)
	splash.Close()
	return err
}

func launchLatest(exeOverride string, splash *SplashWindow) error {
	config := NewConfig("", "", DetectArch())
	latest, err := GetLatestVersion(config)
	if err != nil {
		splash.Close()
		return fmt.Errorf("no installed version found — place a %s file next to this executable", PackageExt)
	}

	config.Version = latest
	manifest, err := LoadManifest(config.ManifestPath())
	if err != nil {
		splash.Close()
		return fmt.Errorf("load manifest for %s: %w", latest, err)
	}

	exePath := filepath.Join(config.VersionDir, manifest.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	err = launch(exePath)
	splash.Close()
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
