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

Optional `portablePackager` config override:
```json
{
  "portablePackager": {
    "exeName": "MyApp.exe",
    "arch": "amd64"
  }
}
```

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

### Pack Options
| Option | Description |
|--------|-------------|
| `-app <name>` | Application name (Required) |
| `-o <path>` | Output file path (Default: `<app>-<version>-<arch>.kbpkg`) |
| `-v <version>` | Version string (Required) |
| `-arch <arch>` | Target architecture: amd64, 386, arm64 (Default: amd64) |
| `-exe <name>` | Executable name (Default: `<app>.exe`) |

### Run Options
| Option | Description |
|--------|-------------|
| `-package <path>` | Path to `.kbpkg` file (Auto-searches if no embed) |
| `-exe <name>` | Override executable name |

## .kbpkg Package Format
`.kbpkg` is a gzip-compressed tar archive. The first entry must be `_manifest.json`.
```json
{
  "appName": "MyApp",
  "version": "1.0.0",
  "arch": "amd64",
  "exe": "MyApp.exe",
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

## Requirements
- Go must be installed for manual builds.
- Go is not required when using the npm package with pre-built binaries.
