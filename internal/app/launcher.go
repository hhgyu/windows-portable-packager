package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func Run(pkgPath, exeOverride string) error {
	if HasEmbeddedPackage() {
		return runFromEmbedded(exeOverride)
	}
	return runFromFile(pkgPath, exeOverride)
}

func runFromEmbedded(exeOverride string) error {
	manifest, err := ReadEmbeddedManifest()
	if err != nil {
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
			return launch(filepath.Join(config.VersionDir, exeName))
		}
	}

	Log(fmt.Sprintf(T(MsgInstalling), manifest.AppName, manifest.Version))
	LogVerbose(fmt.Sprintf(T(MsgExtracting), config.VersionDir))
	extracted, err := UnpackEmbedded(config.VersionDir)
	if err != nil {
		return fmt.Errorf("unpack: %w", err)
	}

	CleanOldVersions(config, []string{extracted.Version})

	exePath := filepath.Join(config.VersionDir, extracted.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	return launch(exePath)
}

func runFromFile(pkgPath, exeOverride string) error {
	pkg, err := FindPackage(pkgPath)
	if err != nil {
		LogVerbose(T(MsgNoPackageFound))
		return launchLatest(exeOverride)
	}

	manifest, err := ReadPackageManifest(pkg)
	if err != nil {
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
			return launch(filepath.Join(config.VersionDir, exeName))
		}
	}

	Log(fmt.Sprintf(T(MsgInstalling), manifest.AppName, manifest.Version))
	LogVerbose(fmt.Sprintf(T(MsgExtracting), config.VersionDir))
	extracted, err := Unpack(pkg, config.VersionDir)
	if err != nil {
		return fmt.Errorf("unpack: %w", err)
	}

	CleanOldVersions(config, []string{extracted.Version})

	exePath := filepath.Join(config.VersionDir, extracted.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	return launch(exePath)
}

func launchLatest(exeOverride string) error {
	config := NewConfig("", "", DetectArch())
	latest, err := GetLatestVersion(config)
	if err != nil {
		return fmt.Errorf("no installed version found — place a %s file next to this executable", PackageExt)
	}

	config.Version = latest
	manifest, err := LoadManifest(config.ManifestPath())
	if err != nil {
		return fmt.Errorf("load manifest for %s: %w", latest, err)
	}

	exePath := filepath.Join(config.VersionDir, manifest.Exe)
	if exeOverride != "" {
		exePath = filepath.Join(config.VersionDir, exeOverride)
	}
	return launch(exePath)
}

func launch(exePath string) error {
	if _, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("exe not found: %s", exePath)
	}

	LogVerbose(fmt.Sprintf(T(MsgLaunching), filepath.Base(exePath)))

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	setSysProcAttr(cmd)

	return cmd.Start()
}
