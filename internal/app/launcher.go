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
	fmt.Printf("Package: %s %s (%s)\n", manifest.AppName, manifest.Version, manifest.Arch)

	config := NewConfig(manifest.AppName, manifest.Version, manifest.Arch)

	installedManifest, err := LoadManifest(config.ManifestPath())
	if err == nil {
		exeName := manifest.Exe
		if exeOverride != "" {
			exeName = exeOverride
		}
		ok, verifyErr := installedManifest.VerifySingle(config.VersionDir, exeName)
		if verifyErr == nil && ok {
			fmt.Println("Already installed and verified, launching...")
			CleanOldVersions(config, []string{manifest.Version})
			return launch(filepath.Join(config.VersionDir, exeName))
		}
	}

	fmt.Printf("Extracting to %s...\n", config.VersionDir)
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
		fmt.Println("No package found, looking for installed version...")
		return launchLatest(exeOverride)
	}

	manifest, err := ReadPackageManifest(pkg)
	if err != nil {
		return fmt.Errorf("read package: %w", err)
	}
	fmt.Printf("Package: %s %s (%s)\n", manifest.AppName, manifest.Version, manifest.Arch)

	config := NewConfig(manifest.AppName, manifest.Version, manifest.Arch)

	installedManifest, err := LoadManifest(config.ManifestPath())
	if err == nil {
		exeName := manifest.Exe
		if exeOverride != "" {
			exeName = exeOverride
		}
		ok, verifyErr := installedManifest.VerifySingle(config.VersionDir, exeName)
		if verifyErr == nil && ok {
			fmt.Println("Already installed and verified, launching...")
			CleanOldVersions(config, []string{manifest.Version})
			return launch(filepath.Join(config.VersionDir, exeName))
		}
	}

	fmt.Printf("Extracting to %s...\n", config.VersionDir)
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

	fmt.Printf("Launching %s\n", filepath.Base(exePath))

	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	setSysProcAttr(cmd)

	return cmd.Start()
}
