"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { runInit, HOOK } = require("../lib/init");

function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "wpp-init-test-"));
}

function writePkg(dir, content) {
  fs.writeFileSync(path.join(dir, "package.json"), JSON.stringify(content, null, 2) + "\n", "utf8");
}

function readPkg(dir) {
  return JSON.parse(fs.readFileSync(path.join(dir, "package.json"), "utf8"));
}

describe("runInit", () => {
  let tmpDir;

  beforeEach(() => {
    tmpDir = makeTempDir();
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  test("package.json이 없으면 code 1 반환", () => {
    const result = runInit(tmpDir);
    expect(result.code).toBe(1);
    expect(result.message).toMatch("package.json not found");
  });

  test("package.json이 깨져 있으면 code 1 반환", () => {
    fs.writeFileSync(path.join(tmpDir, "package.json"), "{ invalid json", "utf8");
    const result = runInit(tmpDir);
    expect(result.code).toBe(1);
    expect(result.message).toMatch("failed to parse package.json");
  });

  test("build 섹션 없을 때 훅 추가 후 code 0 반환", () => {
    writePkg(tmpDir, { name: "my-app", version: "1.0.0" });

    const result = runInit(tmpDir);

    expect(result.code).toBe(0);
    expect(result.message).toMatch("configured successfully");
    expect(readPkg(tmpDir).build.afterAllArtifactBuild).toBe(HOOK);
  });

  test("build 섹션 있지만 afterAllArtifactBuild 없을 때 훅 추가", () => {
    writePkg(tmpDir, { name: "my-app", version: "1.0.0", build: { productName: "MyApp" } });

    const result = runInit(tmpDir);

    expect(result.code).toBe(0);
    const pkg = readPkg(tmpDir);
    expect(pkg.build.afterAllArtifactBuild).toBe(HOOK);
    expect(pkg.build.productName).toBe("MyApp");
  });

  test("이미 동일한 훅이 설정돼 있으면 파일 변경 없이 code 0 반환", () => {
    writePkg(tmpDir, { name: "my-app", build: { afterAllArtifactBuild: HOOK } });
    const before = fs.statSync(path.join(tmpDir, "package.json")).mtimeMs;

    const result = runInit(tmpDir);

    expect(result.code).toBe(0);
    expect(result.message).toMatch("already configured");
    expect(fs.statSync(path.join(tmpDir, "package.json")).mtimeMs).toBe(before);
  });

  test("다른 훅이 있을 때 y 입력하면 덮어쓰기", () => {
    writePkg(tmpDir, { name: "my-app", build: { afterAllArtifactBuild: "other-hook" } });

    const result = runInit(tmpDir, { readLine: () => "y" });

    expect(result.code).toBe(0);
    expect(readPkg(tmpDir).build.afterAllArtifactBuild).toBe(HOOK);
  });

  test("다른 훅이 있을 때 n 입력하면 중단", () => {
    writePkg(tmpDir, { name: "my-app", build: { afterAllArtifactBuild: "other-hook" } });

    const result = runInit(tmpDir, { readLine: () => "n" });

    expect(result.code).toBe(0);
    expect(result.message).toBe("Aborted.");
    expect(readPkg(tmpDir).build.afterAllArtifactBuild).toBe("other-hook");
  });

  test("다른 훅이 있을 때 빈 입력이면 중단", () => {
    writePkg(tmpDir, { name: "my-app", build: { afterAllArtifactBuild: "other-hook" } });

    const result = runInit(tmpDir, { readLine: () => "" });

    expect(result.code).toBe(0);
    expect(result.message).toBe("Aborted.");
  });

  test("기존 package.json 필드가 보존됨", () => {
    writePkg(tmpDir, {
      name: "my-app",
      version: "2.0.0",
      description: "test app",
      scripts: { build: "electron-builder" },
    });

    runInit(tmpDir);

    const pkg = readPkg(tmpDir);
    expect(pkg.name).toBe("my-app");
    expect(pkg.version).toBe("2.0.0");
    expect(pkg.description).toBe("test app");
    expect(pkg.scripts.build).toBe("electron-builder");
    expect(pkg.build.afterAllArtifactBuild).toBe(HOOK);
  });
});
