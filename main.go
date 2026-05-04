package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hhgyu/windows-portable-packager/internal/app"
)

func main() {
	args := os.Args[1:]
	args = parseGlobalFlags(args)

	if len(args) == 0 {
		runDefault()
		return
	}

	switch args[0] {
	case "pack":
		packCmd(args[1:])
	case "run":
		runCmd(args[1:])
	case "verify":
		verifyCmd(args[1:])
	case "clean":
		cleanCmd(args[1:])
	case "version":
		fmt.Println("windows-portable-packager 1.0.0")
	case "help", "--help", "-h":
		printHelp()
	default:
		showError(fmt.Sprintf("Unknown command: %s", args[0]))
		printHelp()
		os.Exit(1)
	}
}

// guardLauncherStartup acquires a per-app singleton mutex and arms a watchdog
// timer. Returns a release function the caller MUST defer. If another instance
// is already running, the process exits immediately with code 0 (we treat
// double-launch as benign user behaviour, not an error).
func guardLauncherStartup() func() {
	manifest, err := resolvePackageManifest("")
	appName := "windows-portable-packager"
	if err == nil && manifest.AppName != "" {
		appName = manifest.AppName
	}

	mutex, acquired, err := app.AcquireSingleton(appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: singleton mutex unavailable: %v\n", err)
	} else if !acquired {
		message := app.T(app.MsgAlreadyRunning)
		title := fmt.Sprintf(app.T(app.MsgAlreadyRunningTitle), appName)
		fmt.Fprintln(os.Stderr, message)
		if !app.IsTerminal() {
			app.ShowInfoDialog(title, message)
		}
		os.Exit(0)
	}

	app.StartWatchdog(appName, app.DefaultWatchdogTimeout)

	return func() {
		app.DisarmWatchdog()
		mutex.Release()
	}
}

func showError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	if !app.IsTerminal() {
		app.ShowErrorDialog("Error", msg)
	}
}

func runDefault() {
	runCmd(nil)
}

func runCmd(args []string) {
	release := guardLauncherStartup()
	defer release()

	fs := flag.NewFlagSet("run", flag.ExitOnError)
	pkgPath := fs.String("package", "", "Path to .kbpkg file (auto-detected if empty)")
	exeOverride := fs.String("exe", "", "Override main executable name")
	splashPath := fs.String("splash", "", "Path to splash image (png/jpg/gif/apng)")
	fs.Parse(args)

	if err := app.Run(*pkgPath, *exeOverride, *splashPath); err != nil {
		showError(fmt.Sprintf("Run error: %v", err))
		os.Exit(1)
	}
}

func parseGlobalFlags(args []string) []string {
	var remaining []string
	for i, arg := range args {
		if arg == "-v" || arg == "--verbose" {
			app.SetLogLevel(app.LogLevelVerbose)
		} else {
			remaining = append(remaining, arg)
			if !strings.HasPrefix(arg, "-") {
				remaining = append(remaining, args[i+1:]...)
				break
			}
		}
	}
	return remaining
}

func reorderFlags(args []string) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}
func packCmd(args []string) {
	fs := flag.NewFlagSet("pack", flag.ExitOnError)
	output := fs.String("o", "", "Output .kbpkg file path")
	appName := fs.String("app", "", "Application name (required)")
	version := fs.String("v", "", "Version string (required)")
	arch := fs.String("arch", "amd64", "Target architecture: amd64, 386, arm64")
	exeName := fs.String("exe", "", "Main executable name (default: <app>.exe)")
	splashPath := fs.String("splash", "", "Splash image path (png/jpg/gif/apng)")
	splashMinDur := fs.Int("splash-min-duration", 0, "Minimum splash visible time in ms (default 0 = close immediately)")
	compression := fs.String("compression", "zstd", "Compression format: zstd, gzip")
	level := fs.Int("level", 0, "Compression level (zstd: 1-19, gzip: 1-9, 0=default)")
	fs.Parse(reorderFlags(args))

	srcDir := ""
	if fs.NArg() >= 1 {
		srcDir = fs.Arg(0)
	}

	if srcDir == "" || *version == "" || *appName == "" {
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

	srcDir = fs.Arg(0)
	exe := *exeName
	if exe == "" {
		exe = *appName + ".exe"
	}
	if *output == "" {
		*output = fmt.Sprintf("%s-%s-%s%s", *appName, *version, *arch, app.PackageExt)
	}

	opts := app.PackOptions{Level: *level, SplashMinMs: *splashMinDur}
	switch *compression {
	case "gzip":
		opts.Compression = app.CompressionGzip
	case "xz":
		opts.Compression = app.CompressionXZ
	default:
		opts.Compression = app.CompressionZstd
	}

	if err := app.Pack(srcDir, *output, *appName, *version, *arch, exe, *splashPath, opts); err != nil {
		showError(fmt.Sprintf("Pack error: %v", err))
		os.Exit(1)
	}
}

func verifyCmd(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	version := fs.String("v", "", "Version to verify (auto-detected from package if empty)")
	fs.Parse(args)

	pkgManifest, err := resolvePackageManifest(*version)
	if err != nil {
		showError(fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	config := app.NewConfig(pkgManifest.AppName, pkgManifest.Version, "")
	installed, err := app.LoadManifest(config.ManifestPath())
	if err != nil {
		showError(fmt.Sprintf("Error: no installation found for version %s", pkgManifest.Version))
		os.Exit(1)
	}

	mismatches, err := installed.Verify(config.VersionDir)
	if err != nil {
		showError(fmt.Sprintf("Verification error: %v", err))
		os.Exit(1)
	}

	if len(mismatches) > 0 {
		showError(fmt.Sprintf("Verification FAILED: %d file(s) mismatched", len(mismatches)))
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
		showError(fmt.Sprintf("Error: %v", err))
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
