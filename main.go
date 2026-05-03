package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hhgyu/windows-portable-packager/internal/app"
)

func main() {
	if len(os.Args) < 2 {
		runDefault()
		return
	}

	switch os.Args[1] {
	case "pack":
		packCmd(os.Args[2:])
	case "run":
		runCmd(os.Args[2:])
	case "verify":
		verifyCmd(os.Args[2:])
	case "clean":
		cleanCmd(os.Args[2:])
	case "version":
		fmt.Println("windows-portable-packager 1.0.0")
	case "help", "--help", "-h":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printHelp()
		os.Exit(1)
	}
}

func runDefault() {
	runCmd(os.Args[1:])
}

func packCmd(args []string) {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	output := fs.String("o", "", "Output .kbpkg file path")
	appName := fs.String("app", "", "Application name (required)")
	version := fs.String("v", "", "Version string (required)")
	arch := fs.String("arch", "amd64", "Target architecture: amd64, 386, arm64")
	exeName := fs.String("exe", "", "Main executable name (default: <app>.exe)")
	fs.Parse(args)

	if fs.NArg() < 1 || *version == "" || *appName == "" {
		fmt.Fprintln(os.Stderr, "Usage: windows-portable-packager pack <source-dir> -app <name> -v <version> [-o <output>] [-arch amd64|386|arm64]")
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintln(os.Stderr, "  windows-portable-packager pack dist/win-unpacked -app KeyBridge -v 1.0.0 -arch amd64")
		if *appName == "" {
			fmt.Fprintln(os.Stderr, "\nError: app name is required (-app)")
		}
		if *version == "" {
			fmt.Fprintln(os.Stderr, "Error: version is required (-v)")
		}
		os.Exit(1)
	}

	srcDir := fs.Arg(0)
	exe := *exeName
	if exe == "" {
		exe = *appName + ".exe"
	}
	if *output == "" {
		*output = fmt.Sprintf("%s-%s-%s%s", *appName, *version, *arch, app.PackageExt)
	}

	if err := app.Pack(srcDir, *output, *appName, *version, *arch, exe); err != nil {
		fmt.Fprintf(os.Stderr, "Pack error: %v\n", err)
		os.Exit(1)
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	pkgPath := fs.String("package", "", "Path to .kbpkg file (auto-detected if empty)")
	exeOverride := fs.String("exe", "", "Override main executable name")
	fs.Parse(args)

	if err := app.Run(*pkgPath, *exeOverride); err != nil {
		fmt.Fprintf(os.Stderr, "Run error: %v\n", err)
		os.Exit(1)
	}
}

func verifyCmd(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	version := fs.String("v", "", "Version to verify (auto-detected from package if empty)")
	fs.Parse(args)

	pkgManifest, err := resolvePackageManifest(*version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	config := app.NewConfig(pkgManifest.AppName, pkgManifest.Version, "")
	installed, err := app.LoadManifest(config.ManifestPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: no installation found for version %s\n", pkgManifest.Version)
		os.Exit(1)
	}

	mismatches, err := installed.Verify(config.VersionDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification error: %v\n", err)
		os.Exit(1)
	}

	if len(mismatches) > 0 {
		fmt.Fprintf(os.Stderr, "Verification FAILED: %d file(s) mismatched\n", len(mismatches))
		for _, f := range mismatches {
			fmt.Fprintf(os.Stderr, "  - %s\n", f)
		}
		os.Exit(1)
	}

	fmt.Printf("Verification passed: %d files OK (%s %s, arch %s)\n", len(installed.Files), installed.AppName, installed.Version, installed.Arch)
}

func resolvePackageManifest(version string) (*app.Manifest, error) {
	if app.HasEmbeddedPackage() {
		return app.ReadEmbeddedManifest()
	}
	pkg, err := app.FindPackage("")
	if err != nil {
		return nil, err
	}
	return app.ReadPackageManifest(pkg)
}

func cleanCmd(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	keepCurrent := fs.Bool("current", true, "Keep the version from the current package")
	fs.Parse(args)

	var keepVersions []string
	appName := ""

	if manifest, err := resolvePackageManifest(""); err == nil {
		appName = manifest.AppName
		if *keepCurrent {
			keepVersions = append(keepVersions, manifest.Version)
			fmt.Printf("Keeping current version: %s\n", manifest.Version)
		}
	}

	config := app.NewConfig(appName, "", "")
	removed, err := app.CleanOldVersions(config, keepVersions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Cleaned %d old version(s)\n", removed)
}

func printHelp() {
	fmt.Printf(`windows-portable-packager - Go-based portable launcher for Electron apps

Usage:
  windows-portable-packager <command> [options]

Commands:
  pack <dir>     Create portable package from build directory
  run            Extract (if needed) and launch the app (default)
  verify         Verify integrity of installed files
  clean          Remove old installed versions
  version        Show version
  help           Show this help

Pack options:
  -app <name>    Application name (required)
  -o <path>      Output .kbpkg file (default: <app>-<version>-<arch>.kbpkg)
  -v <version>   Version string (required)
  -arch <arch>   Target architecture: amd64, 386, arm64 (default: amd64)
  -exe <name>    Main executable name (default: <app>.exe)
`)
}
