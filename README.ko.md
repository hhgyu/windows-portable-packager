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

`portablePackager` 설정 옵션:
```json
{
  "portablePackager": {
    "exeName": "MyApp.exe",
    "arch": "amd64",
    "splash": "build/splash.png",
    "splashMinDuration": 2000,
    "compression": "zstd",
    "level": 0
  }
}
```

| 필드 | 설명 | 기본값 |
|------|------|--------|
| `exeName` | 실행 파일 이름 | `<productName>.exe` |
| `arch` | 대상 아키텍처 | `amd64` |
| `splash` | 스플래시 이미지 경로 (png/jpg/gif/apng) | — |
| `splashMinDuration` | 스플래시 최소 표시 시간 (ms). `0`이거나 미지정이면 자식 앱 실행 즉시 닫힘. | `0` |
| `compression` | 압축 포맷: `zstd`, `gzip`, `xz` | `zstd` |
| `level` | 압축 레벨 (zstd: 1–19, gzip: 1–9, xz: 1–9, 0=기본값) | `0` |

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

### 전역 옵션
| 옵션 | 설명 |
|------|------|
| `-v`, `--verbose` | 상세 로그 출력 |

### Pack 옵션
| 옵션 | 설명 |
|------|------|
| `-app <name>` | 애플리케이션 이름 (필수) |
| `-o <path>` | 출력 파일 경로 (기본: `<app>-<version>-<arch>.kbpkg`) |
| `-v <version>` | 버전 문자열 (필수) |
| `-arch <arch>` | 대상 아키텍처: amd64, 386, arm64 (기본: amd64) |
| `-exe <name>` | 실행 파일 이름 (기본: `<app>.exe`) |
| `-splash <path>` | 스플래시 이미지 경로 (png/jpg/gif/apng) |
| `-compression <fmt>` | 압축 포맷: zstd, gzip, xz (기본: zstd) |
| `-level <n>` | 압축 레벨 (zstd: 1–19, gzip: 1–9, xz: 1–9, 0=기본값) |
| `-splash-min-duration <ms>` | 스플래시 최소 표시 시간 (ms). 기본 `0` = 즉시 닫힘 |

### Run 옵션
| 옵션 | 설명 |
|------|------|
| `-package <path>` | `.kbpkg` 파일 경로 (embed 없을 때 자동 탐색) |
| `-exe <name>` | 실행 파일 이름 오버라이드 |
| `-splash <path>` | 스플래시 이미지 경로 오버라이드 |

## 스플래시 화면
앱 실행 즉시 스플래시 이미지가 표시되며, 압축 해제 중에도 유지됩니다. 앱이 시작되면 자동으로 닫힙니다.

지원 포맷: PNG, JPG, GIF (애니메이션), APNG (애니메이션).

`portablePackager.splash`로 빌드 시 임베드하거나, 실행 시 `-splash` 플래그로 지정할 수 있습니다.

기본 동작은 자식 앱이 실행되는 즉시 스플래시를 닫는 것이라, 빠른 환경에서는 스플래시가 100ms 미만으로 깜빡이듯 사라질 수 있습니다. `portablePackager.splashMinDuration`(또는 `pack -splash-min-duration <ms>`)을 설정하면 스플래시가 최소 N 밀리초 동안 유지됩니다. 런처가 오류로 중단될 때는 이 설정과 무관하게 즉시 닫혀, 실패한 앱이 멈춘 스플래시 뒤에 사용자를 가두지 않도록 합니다.

## .kbpkg 패키지 포맷
`.kbpkg`는 압축된 tar 아카이브입니다. 첫 엔트리는 반드시 `_manifest.json`이어야 합니다. 기본 압축 포맷은 **zstd**이며, gzip/xz 패키지도 압축 해제 시 자동으로 감지됩니다.

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

## 다국어 지원
UI 메시지(다이얼로그, 로그 출력)는 시스템 로케일에 따라 **한국어** 또는 **영어**로 자동 표시됩니다. Windows에서는 `GetUserDefaultLocaleName`으로 감지하며, `ko-*` 로케일이면 한국어, 그 외에는 영어로 표시됩니다.

## 트러블슈팅

### 첫 실행 시 런처가 멈춰 있거나 더블클릭해도 아무 반응이 없습니다

이 런처는 **앱 전체 데이터를 임베드한 대용량(보통 100MB+) 미서명 실행 파일**입니다. 일부 보안 솔루션은 이러한 바이너리를 의심스럽게 분류하고 실행 전에 **심층 행위 분석**을 수행합니다. 분석이 끝나지 않으면 런처 프로세스는 OS 이미지 로더 단계에서 정지하며, 우리 코드는 한 줄도 실행되지 않은 상태가 됩니다.

**증상:**
- 런처 창이 끝내 뜨지 않음
- 작업 관리자에서 프로세스가 계속 살아있고 일정한 CPU를 사용
- `%APPDATA%\<productName>\app\` 디렉토리가 끝내 생성되지 않음
- `--help` / `-v` 같은 옵션도 아무 출력이 없음

**내장된 보호 장치:**
런처에는 위 시나리오의 피해를 줄이기 위한 두 가지 방어 장치가 포함되어 있습니다.
- **단일 인스턴스 뮤텍스** — 추가 더블클릭은 좀비 프로세스로 누적되는 대신 즉시 종료됩니다.
- **60초 시작 워치독** — 런처가 60초 안에 앱을 띄우지 못하면 스스로 종료되며 안내 다이얼로그를 띄웁니다.

**우선 순위 순 해결 방법:**
1. **사용 중인 보안 소프트웨어에 런처 예외를 등록**하세요 (런처 파일 단위 또는 출력 디렉토리 전체). 메뉴 명칭은 제품마다 다르며 "예외", "검사 제외", "신뢰할 수 있는 애플리케이션", "허용 목록" 같은 항목을 찾으면 됩니다.
2. **앱 격리 검사 / 행위 기반 분석을 비활성화**하세요. 단순 파일 예외로 부족할 때 사용합니다. 많은 엔드포인트 보안 제품은 파일 허용 목록과는 별개로 "격리"나 "심층 분석" 기능을 두고 있습니다.
3. **런처에 코드 서명을 적용**하세요. 자체 서명만으로도 대부분의 휴리스틱이 통과되며, EV 인증서는 Microsoft SmartScreen reputation도 함께 해결합니다. 사용자 환경 설정 없이 해결할 수 있는 유일한 방법입니다.

위 단계 후에도 런처가 멈춘다면 다음 정보와 함께 이슈를 등록해 주세요:
- OS 버전 (`winver`)
- 설치되어 있는 보안 소프트웨어
- 런처가 멈춘 상태에서 `Get-Process <appname>* | Select Id, CPU, WorkingSet64, Threads` 출력

## 요구 사항
- 수동 빌드 시 Go 설치가 필요합니다.
- 사전 빌드된 바이너리가 포함된 npm 패키지를 사용할 때는 Go가 필요하지 않습니다.
