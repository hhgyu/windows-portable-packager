"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { runHook, resolveConfig, resolveBin } = require("../lib/hook");

function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "wpp-hook-test-"));
}

function makeUnpacked(outDir, exeName = "MyApp.exe") {
  const unpackedDir = path.join(outDir, "win-unpacked");
  fs.mkdirSync(unpackedDir, { recursive: true });
  fs.writeFileSync(path.join(unpackedDir, exeName), "fake-exe");
  return unpackedDir;
}

function makePackagerDir(tmpDir, arch = "amd64") {
  const distDir = path.join(tmpDir, "packager", "dist");
  const embedDir = path.join(tmpDir, "packager", "internal", "embed");
  fs.mkdirSync(distDir, { recursive: true });
  fs.mkdirSync(embedDir, { recursive: true });
  fs.writeFileSync(path.join(distDir, `windows-portable-packager-${arch}.exe`), "fake-bin");
  fs.writeFileSync(path.join(embedDir, "app.kbpkg"), "PLACEHOLDER\n");
  return path.join(tmpDir, "packager");
}

function makePkgJson(dir, content) {
  const pkgPath = path.join(dir, "package.json");
  fs.writeFileSync(pkgPath, JSON.stringify(content, null, 2) + "\n", "utf8");
  return pkgPath;
}

describe("resolveConfig", () => {
  let tmpDir;

  beforeEach(() => { tmpDir = makeTempDir(); });
  afterEach(() => { fs.rmSync(tmpDir, { recursive: true, force: true }); });

  test("productName과 version을 package.json에서 읽음", () => {
    const pkgPath = makePkgJson(tmpDir, {
      name: "my-app",
      version: "1.2.3",
      build: { productName: "MyApp" },
    });
    const config = resolveConfig(pkgPath);
    expect(config.productName).toBe("MyApp");
    expect(config.version).toBe("1.2.3");
    expect(config.exeName).toBe("MyApp.exe");
    expect(config.goArch).toBe("amd64");
  });

  test("build.productName 없으면 name 사용", () => {
    const pkgPath = makePkgJson(tmpDir, { name: "my-app", version: "1.0.0" });
    const config = resolveConfig(pkgPath);
    expect(config.productName).toBe("my-app");
    expect(config.exeName).toBe("my-app.exe");
  });

  test("portablePackager로 exeName과 arch 오버라이드", () => {
    const pkgPath = makePkgJson(tmpDir, {
      name: "my-app",
      version: "1.0.0",
      build: { productName: "MyApp" },
      portablePackager: { exeName: "Custom.exe", arch: "386" },
    });
    const config = resolveConfig(pkgPath);
    expect(config.exeName).toBe("Custom.exe");
    expect(config.goArch).toBe("386");
  });
});

describe("resolveBin", () => {
  let tmpDir;

  beforeEach(() => { tmpDir = makeTempDir(); });
  afterEach(() => { fs.rmSync(tmpDir, { recursive: true, force: true }); });

  test("arch별 바이너리를 우선 반환", () => {
    const distDir = path.join(tmpDir, "dist");
    fs.mkdirSync(distDir);
    fs.writeFileSync(path.join(distDir, "windows-portable-packager-amd64.exe"), "");
    fs.writeFileSync(path.join(distDir, "windows-portable-packager.exe"), "");

    expect(resolveBin(tmpDir, "amd64")).toBe(path.join(distDir, "windows-portable-packager-amd64.exe"));
  });

  test("arch별 바이너리 없으면 fallback 반환", () => {
    const distDir = path.join(tmpDir, "dist");
    fs.mkdirSync(distDir);
    fs.writeFileSync(path.join(distDir, "windows-portable-packager.exe"), "");

    expect(resolveBin(tmpDir, "arm64")).toBe(path.join(distDir, "windows-portable-packager.exe"));
  });

  test("바이너리 없으면 null 반환", () => {
    fs.mkdirSync(path.join(tmpDir, "dist"));
    expect(resolveBin(tmpDir, "amd64")).toBeNull();
  });
});

describe("runHook", () => {
  let tmpDir;

  beforeEach(() => { tmpDir = makeTempDir(); });
  afterEach(() => { fs.rmSync(tmpDir, { recursive: true, force: true }); });

  test("win-unpacked 없으면 skipped 반환", async () => {
    const outDir = path.join(tmpDir, "out");
    fs.mkdirSync(outDir);

    const result = await runHook({ outDir }, {
      pkgJsonPath: makePkgJson(tmpDir, { name: "my-app", version: "1.0.0" }),
      packagerDir: makePackagerDir(tmpDir),
    });

    expect(result.skipped).toBe(true);
    expect(result.reason).toMatch("win-unpacked");
  });

  test("exe 파일 없으면 skipped 반환", async () => {
    const outDir = path.join(tmpDir, "out");
    fs.mkdirSync(path.join(outDir, "win-unpacked"), { recursive: true });

    const result = await runHook({ outDir }, {
      pkgJsonPath: makePkgJson(tmpDir, { name: "my-app", version: "1.0.0", build: { productName: "MyApp" } }),
      packagerDir: makePackagerDir(tmpDir),
    });

    expect(result.skipped).toBe(true);
    expect(result.reason).toMatch("exe not found");
  });

  test("packager 바이너리 없으면 skipped 반환", async () => {
    const outDir = path.join(tmpDir, "out");
    makeUnpacked(outDir, "MyApp.exe");

    const emptyPackagerDir = path.join(tmpDir, "empty-packager");
    const embedDir = path.join(emptyPackagerDir, "internal", "embed");
    fs.mkdirSync(path.join(emptyPackagerDir, "dist"), { recursive: true });
    fs.mkdirSync(embedDir, { recursive: true });
    fs.writeFileSync(path.join(embedDir, "app.kbpkg"), "PLACEHOLDER\n");

    const result = await runHook({ outDir }, {
      pkgJsonPath: makePkgJson(tmpDir, { name: "my-app", version: "1.0.0", build: { productName: "MyApp" } }),
      packagerDir: emptyPackagerDir,
    });

    expect(result.skipped).toBe(true);
    expect(result.reason).toMatch("packager binary not found");
  });

  test("정상 흐름 — exec 호출되고 finalExe 경로 반환", async () => {
    const outDir = path.join(tmpDir, "out");
    makeUnpacked(outDir, "MyApp.exe");
    const packagerDir = makePackagerDir(tmpDir);
    const pkgJsonPath = makePkgJson(tmpDir, {
      name: "my-app",
      version: "2.0.0",
      build: { productName: "MyApp" },
    });

    const execCalls = [];
    const execGoCalls = [];

    const result = await runHook({ outDir }, {
      pkgJsonPath,
      packagerDir,
      exec: (bin, args) => execCalls.push({ bin, args }),
      execGo: (cmd, cwd, arch) => execGoCalls.push({ cmd, cwd, arch }),
    });

    expect(result.skipped).toBe(false);
    expect(result.finalExe).toBe(path.join(outDir, "MyApp-2.0.0-amd64.exe"));
    expect(execCalls).toHaveLength(1);
    expect(execCalls[0].args).toContain("MyApp");
    expect(execCalls[0].args).toContain("2.0.0");
    expect(execGoCalls).toHaveLength(1);
  });

  test("exec 실패 시 skipped 반환 + embedPath 복원", async () => {
    const outDir = path.join(tmpDir, "out");
    makeUnpacked(outDir, "MyApp.exe");
    const packagerDir = makePackagerDir(tmpDir);
    const embedPath = path.join(packagerDir, "internal", "embed", "app.kbpkg");

    const result = await runHook({ outDir }, {
      pkgJsonPath: makePkgJson(tmpDir, { name: "my-app", version: "1.0.0", build: { productName: "MyApp" } }),
      packagerDir,
      exec: () => { throw new Error("pack failed"); },
      execGo: () => {},
    });

    expect(result.skipped).toBe(true);
    expect(result.reason).toMatch("pack failed");
    expect(fs.readFileSync(embedPath, "utf8")).toMatch("PLACEHOLDER");
  });

  test("빌드 성공 후에도 embedPath가 PLACEHOLDER로 복원됨", async () => {
    const outDir = path.join(tmpDir, "out");
    makeUnpacked(outDir, "MyApp.exe");
    const packagerDir = makePackagerDir(tmpDir);
    const embedPath = path.join(packagerDir, "internal", "embed", "app.kbpkg");

    await runHook({ outDir }, {
      pkgJsonPath: makePkgJson(tmpDir, { name: "my-app", version: "1.0.0", build: { productName: "MyApp" } }),
      packagerDir,
      exec: () => {},
      execGo: () => {},
    });

    expect(fs.readFileSync(embedPath, "utf8")).toMatch("PLACEHOLDER");
  });
});
