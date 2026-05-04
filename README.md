# windows-portable-packager
[한국어](./README.ko.md)

Go-based Windows portable launcher for Electron apps.

Electron's default portable build extracts files **every time** it runs. This tool extracts to `%APPDATA%` **only once**, verifies file integrity using SHA256 hashes, and automatically cleans up previous versions.

## How it works
All information such as app name, version, and architecture is read from the `.kbpkg` manifest at runtime. No app-specific information is hardcoded into the tool itself.

When used with electron-builder, settings like `productName`, `version`, and target architecture are automatically retrieved from the electron-builder configuration.

- **First run:** Extracts embedded data from exe to `%APPDATA%\<productName>\app\<version>\` → Verifies hashes → Launches app.
- **Re-run:** Checks installation directory → Quickly verifies exe hash → Launches immediately.
- **Update:** New exe version distributed → Replaces automatically on next run → Deletes old versions.

### Single EXE Build Pipeline
1. `pnpm dist` (electron-builder)
2. `afterAllArtifactBuild` hook runs automatically:
   - `win-unpacked/` → Generates `embedded/app.kbpkg`
   - `go build` (with `go:embed`) → `<productName>-<version>-<arch>.exe`
   - Restores `embedded/app.kbpkg` to placeholder

## Installation
```bash
pnpm add -D windows-portable-packager
```

Then run `init` to configure your project automatically:

```bash
npx windows-portable-packager init
```

This adds the `afterAllArtifactBuild` hook to your `package.json` automatically.

## Usage — electron-builder integration (Recommended)
Add a single line to your `package.json`:
```json
{
  "build": {
    "productName": "MyApp",
    "afterAllArtifactBuild": "windows-portable-packager/hooks/afterAllArtifactBuild"
  }
}
```

Values automatically detected by the hook:
- **App Name:** `build.productName`
- **Version:** `version`
- **Architecture:** Target architecture (x64 → amd64, ia32 → 386, arm64 → arm64)
- **EXE Name:** `build.productName + ".exe"`

Optional `portablePackager` config:
```json
{
  "portablePackager": {
    "exeName": "MyApp.exe",
    "arch": "amd64",
    "splash": "build/splash.png",
    "compression": "zstd",
    "level": 0
  }
}
```

| Field | Description | Default |
|-------|-------------|---------|
| `exeName` | Executable filename | `<productName>.exe` |
| `arch` | Target architecture | `amd64` |
| `splash` | Splash image path (png/jpg/gif/apng) | — |
| `compression` | Compression format: `zstd`, `gzip`, `xz` | `zstd` |
| `level` | Compression level (zstd: 1–19, gzip: 1–9, xz: 1–9, 0=default) | `0` |

## Usage — manual build
### 1. Pack
```bash
windows-portable-packager pack dist/win-unpacked \
  -app MyApp -v 1.0.0 -arch amd64 \
  -o windows-portable-packager/embedded/app.kbpkg
```

### 2. Go build
```bash
cd windows-portable-packager
set GOARCH=amd64 && go build -ldflags="-s -w" -o dist/MyApp-1.0.0-amd64.exe .
```

## CLI Commands
| Command | Description |
|---------|-------------|
| `init` | Add the electron-builder hook to your project's `package.json` |
| `pack <dir>` | Compress build directory into a `.kbpkg` package |
| `run` | Extract (if needed) and run the app (default) |
| `verify` | Verify integrity of installed files |
| `clean` | Delete previous installation versions |
| `version` | Print version |
| `help` | Print help |

### Global Options
| Option | Description |
|--------|-------------|
| `-v`, `--verbose` | Enable verbose logging |

### Pack Options
| Option | Description |
|--------|-------------|
| `-app <name>` | Application name (Required) |
| `-o <path>` | Output file path (Default: `<app>-<version>-<arch>.kbpkg`) |
| `-v <version>` | Version string (Required) |
| `-arch <arch>` | Target architecture: amd64, 386, arm64 (Default: amd64) |
| `-exe <name>` | Executable name (Default: `<app>.exe`) |
| `-splash <path>` | Splash image path (png/jpg/gif/apng) |
| `-compression <fmt>` | Compression format: zstd, gzip, xz (Default: zstd) |
| `-level <n>` | Compression level (zstd: 1–19, gzip: 1–9, xz: 1–9, 0=default) |

### Run Options
| Option | Description |
|--------|-------------|
| `-package <path>` | Path to `.kbpkg` file (Auto-searches if no embed) |
| `-exe <name>` | Override executable name |
| `-splash <path>` | Override splash image path |

## Splash Screen
A splash image is displayed immediately on launch and stays visible during extraction. It closes automatically when the app starts.

Supported formats: PNG, JPG, GIF (animated), APNG (animated).

Embed via `portablePackager.splash` in your electron-builder `package.json`, or pass `-splash` at runtime.

## .kbpkg Package Format
`.kbpkg` is a compressed tar archive. The first entry must be `_manifest.json`. Default compression is **zstd**; gzip and xz packages are auto-detected on unpack.

```json
{
  "appName": "MyApp",
  "version": "1.0.0",
  "arch": "amd64",
  "exe": "MyApp.exe",
  "splash": "_splash.png",
  "timestamp": "2026-01-01T00:00:00Z",
  "files": {
    "MyApp.exe": "sha256hex...",
    "resources/app.asar": "sha256hex...",
    "locales/en-US.pak": "sha256hex..."
  }
}
```

## Installation Path Layout
```
%APPDATA%\<productName>\
└── app\
    ├── 1.0.0\          ← Previous version (auto-deleted)
    └── 1.1.0\          ← Current version
        ├── _manifest.json
        ├── <productName>.exe
        └── resources\
```

## Supported Architectures
| GOARCH | Windows Arch | electron-builder Target |
|--------|--------------|-------------------------|
| amd64  | x64 (64-bit) | x64                     |
| 386    | x86 (32-bit) | ia32                    |
| arm64  | ARM64        | arm64                   |

## Localization
UI messages (dialogs, log output) are automatically displayed in **Korean** or **English** based on the system locale. Korean (`ko-*`) is detected on Windows via `GetUserDefaultLocaleName`; all other locales fall back to English.

## Troubleshooting

### The launcher hangs on first run / nothing happens when I double-click

The launcher is a **large unsigned executable** (it embeds the entire app payload, often 100+ MB). Some security products treat such binaries as suspicious and run **deep behavioural analysis** before allowing execution. When that analysis stalls, the launcher process is blocked at the OS image-loader stage — before any of our code runs.

**Symptoms:**
- Launcher window never appears
- Process stays alive in Task Manager with steady CPU usage
- No `%APPDATA%\<productName>\app\` directory ever gets created
- Even `--help` / `-v` produce no output

**Built-in safeguards:**
The launcher includes two defenses to limit the damage of this scenario:
- **Single-instance mutex** — additional double-clicks exit immediately instead of piling up zombie processes
- **60-second startup watchdog** — if the launcher cannot start the app within 60 seconds it terminates itself and shows a dialog

**Resolutions, in order of preference:**
1. **Add an exclusion in your security software** for the launcher executable (or the entire output directory). The exact menu varies by product; look for "Exclusions", "Exceptions", "Trusted applications", "Allow list", or similar terms.
2. **Disable application sandboxing / behavioural analysis** if a per-file exclusion is not enough. Many endpoint protection suites have a separate "isolation" or "deep analysis" feature that is not covered by simple file allow-lists.
3. **Sign the launcher** with a code-signing certificate (self-signed is enough to clear most heuristics; an EV certificate clears Microsoft SmartScreen reputation as well). This is the only resolution that does not require user-side configuration.

If the launcher still hangs after these steps, please open an issue with:
- Your OS version (`winver`)
- The security software you have installed
- Output of `Get-Process <appname>* | Select Id, CPU, WorkingSet64, Threads` while the launcher is hung

## Requirements
- Go must be installed for manual builds.
- Go is not required when using the npm package with pre-built binaries.
