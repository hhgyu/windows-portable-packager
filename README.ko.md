# windows-portable-packager
[English](./README.md)

Electron 앱을 위한 Go 기반 Windows 포터블 런처.

Electron의 기본 portable 빌드는 **실행할 때마다** 압축을 해제합니다. 이 도구는 `%APPDATA%` 하위에 **한 번만 압축 해제**하고, SHA256 해시로 파일 무결성을 검증하며, 이전 버전을 자동으로 정리합니다.

## 동작 방식
앱 이름, 버전, 아키텍처 등 모든 정보는 `.kbpkg` 매니페스트에서 런타임에 읽어옵니다. 이 도구 자체에는 앱 관련 정보가 하드코딩되어 있지 않습니다.

electron-builder와 함께 사용할 때는 `productName`, `version`, 빌드 타겟 아키텍처 등 모든 설정이 자동으로 electron-builder 설정에서 읽어옵니다.

- **첫 실행:** exe 내 embed 데이터 → `%APPDATA%\<productName>\app\<version>\` 에 압축 해제 → 해시 검증 → 실행
- **재실행:** 설치 디렉토리 확인 → exe 해시만 빠르게 검증 → 즉시 실행
- **업데이트:** 새 버전 exe 배포 → 다음 실행 때 자동 교체 → 이전 버전 자동 삭제

### 단일 exe 빌드 파이프라인
1. `pnpm dist` (electron-builder)
2. `afterAllArtifactBuild` 훅 자동 실행:
   - `win-unpacked/` → `embedded/app.kbpkg` 생성
   - `go build` (`go:embed` 포함) → `<productName>-<version>-<arch>.exe`
   - `embedded/app.kbpkg` 초기화 복원

## 설치
```bash
pnpm add -D windows-portable-packager
```

설치 후 `init`으로 프로젝트를 자동 설정합니다:

```bash
npx windows-portable-packager init
```

`package.json`에 `afterAllArtifactBuild` 훅을 자동으로 추가합니다.

## 사용법 — electron-builder 통합 (권장)
`package.json`에 훅 한 줄 추가:
```json
{
  "build": {
    "productName": "MyApp",
    "afterAllArtifactBuild": "windows-portable-packager/hooks/afterAllArtifactBuild"
  }
}
```

훅이 자동으로 가져오는 값:
- **앱 이름:** `build.productName`
- **버전:** `version`
- **아키텍처:** 빌드 타겟 아키텍처 (x64 → amd64, ia32 → 386, arm64 → arm64)
- **exe 이름:** `build.productName + ".exe"`

`portablePackager` 설정 오버라이드:
```json
{
  "portablePackager": {
    "exeName": "MyApp.exe",
    "arch": "amd64"
  }
}
```

## 사용법 — 수동 빌드
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

## CLI 커맨드
| 커맨드 | 설명 |
|--------|------|
| `init` | 프로젝트 `package.json`에 electron-builder 훅 자동 추가 |
| `pack <dir>` | 빌드 디렉토리를 `.kbpkg` 패키지로 압축 |
| `run` | 압축 해제(필요 시) 후 앱 실행 (기본값) |
| `verify` | 설치된 파일의 무결성 검증 |
| `clean` | 이전 설치 버전 삭제 |
| `version` | 버전 출력 |
| `help` | 도움말 출력 |

### Pack 옵션
| 옵션 | 설명 |
|------|------|
| `-app <name>` | 애플리케이션 이름 (필수) |
| `-o <path>` | 출력 파일 경로 (기본: `<app>-<version>-<arch>.kbpkg`) |
| `-v <version>` | 버전 문자열 (필수) |
| `-arch <arch>` | 대상 아키텍처: amd64, 386, arm64 (기본: amd64) |
| `-exe <name>` | 실행 파일 이름 (기본: `<app>.exe`) |

### Run 옵션
| 옵션 | 설명 |
|------|------|
| `-package <path>` | `.kbpkg` 파일 경로 (embed 없을 때 자동 탐색) |
| `-exe <name>` | 실행 파일 이름 오버라이드 |

## .kbpkg 패키지 포맷
`.kbpkg`는 gzip 압축 tar 아카이브. 첫 엔트리는 반드시 `_manifest.json`.
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

## 설치 경로
```
%APPDATA%\<productName>\
└── app\
    ├── 1.0.0\          ← 이전 버전 (자동 삭제됨)
    └── 1.1.0\          ← 현재 버전
        ├── _manifest.json
        ├── <productName>.exe
        └── resources\
```

## 지원 아키텍처
| GOARCH | Windows 아키텍처 | electron-builder 타겟 |
|--------|-----------------|----------------------|
| amd64  | x64 (64-bit)    | x64                  |
| 386    | x86 (32-bit)    | ia32                 |
| arm64  | ARM64           | arm64                |

## 요구 사항
- 수동 빌드 시 Go 설치가 필요합니다.
- 사전 빌드된 바이너리가 포함된 npm 패키지를 사용할 때는 Go가 필요하지 않습니다.
