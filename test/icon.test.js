"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { resolveIconPath } = require("../lib/icon");

function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "wpp-icon-test-"));
}

function makePkg(build = {}, portablePackager = {}) {
  return { name: "my-app", version: "1.0.0", build, portablePackager };
}

describe("resolveIconPath", () => {
  let tmpDir;

  beforeEach(() => { tmpDir = makeTempDir(); });
  afterEach(() => { fs.rmSync(tmpDir, { recursive: true, force: true }); });

  test("win.icon 경로 우선 반환", () => {
    const iconPath = path.join(tmpDir, "custom.ico");
    fs.writeFileSync(iconPath, "fake");
    const pkg = makePkg({ win: { icon: iconPath } });

    expect(resolveIconPath(tmpDir, pkg)).toBe(iconPath);
  });

  test("build.icon에 .ico 없으면 .ico 붙여서 탐색", () => {
    const iconPath = path.join(tmpDir, "assets", "app.ico");
    fs.mkdirSync(path.join(tmpDir, "assets"));
    fs.writeFileSync(iconPath, "fake");
    const pkg = makePkg({ icon: path.join(tmpDir, "assets", "app") });

    expect(resolveIconPath(tmpDir, pkg)).toBe(iconPath);
  });

  test("build.icon에 .ico 이미 있으면 중복 .ico 안 붙임", () => {
    const iconPath = path.join(tmpDir, "assets", "app.ico");
    fs.mkdirSync(path.join(tmpDir, "assets"));
    fs.writeFileSync(iconPath, "fake");
    const pkg = makePkg({ icon: path.join(tmpDir, "assets", "app.ico") });

    expect(resolveIconPath(tmpDir, pkg)).toBe(iconPath);
  });

  test("build/icon.ico fallback", () => {
    const buildDir = path.join(tmpDir, "build");
    fs.mkdirSync(buildDir);
    const iconPath = path.join(buildDir, "icon.ico");
    fs.writeFileSync(iconPath, "fake");

    expect(resolveIconPath(tmpDir, makePkg())).toBe(iconPath);
  });

  test("icon.ico fallback", () => {
    const iconPath = path.join(tmpDir, "icon.ico");
    fs.writeFileSync(iconPath, "fake");

    expect(resolveIconPath(tmpDir, makePkg())).toBe(iconPath);
  });

  test("아이콘 파일 없으면 null 반환", () => {
    expect(resolveIconPath(tmpDir, makePkg())).toBeNull();
  });

  test("win.icon 파일 없으면 다음 후보로 넘어감", () => {
    const iconPath = path.join(tmpDir, "build", "icon.ico");
    fs.mkdirSync(path.join(tmpDir, "build"));
    fs.writeFileSync(iconPath, "fake");
    const pkg = makePkg({ win: { icon: path.join(tmpDir, "nonexistent.ico") } });

    expect(resolveIconPath(tmpDir, pkg)).toBe(iconPath);
  });
});
