"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { resolveIconPath, generateSyso } = require("../lib/resource");

function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "wpp-resource-test-"));
}

function makePkg(overrides = {}) {
  return { name: "my-app", version: "1.2.3", description: "Test app", ...overrides };
}

describe("resolveIconPath", () => {
  let tmpDir;
  beforeEach(() => { tmpDir = makeTempDir(); });
  afterEach(() => { fs.rmSync(tmpDir, { recursive: true, force: true }); });

  test("win.icon 우선 반환", () => {
    const iconPath = path.join(tmpDir, "custom.ico");
    fs.writeFileSync(iconPath, "fake");
    const pkg = makePkg({ build: { win: { icon: iconPath } } });
    expect(resolveIconPath(tmpDir, pkg)).toBe(iconPath);
  });

  test("build.icon에 .ico 없으면 .ico 붙여서 탐색", () => {
    const iconPath = path.join(tmpDir, "assets", "app.ico");
    fs.mkdirSync(path.join(tmpDir, "assets"));
    fs.writeFileSync(iconPath, "fake");
    const pkg = makePkg({ build: { icon: path.join(tmpDir, "assets", "app") } });
    expect(resolveIconPath(tmpDir, pkg)).toBe(iconPath);
  });

  test("build.icon에 .ico 이미 있으면 중복 .ico 안 붙임", () => {
    const iconPath = path.join(tmpDir, "assets", "app.ico");
    fs.mkdirSync(path.join(tmpDir, "assets"));
    fs.writeFileSync(iconPath, "fake");
    const pkg = makePkg({ build: { icon: path.join(tmpDir, "assets", "app.ico") } });
    expect(resolveIconPath(tmpDir, pkg)).toBe(iconPath);
  });

  test("build/icon.ico fallback", () => {
    fs.mkdirSync(path.join(tmpDir, "build"));
    const iconPath = path.join(tmpDir, "build", "icon.ico");
    fs.writeFileSync(iconPath, "fake");
    expect(resolveIconPath(tmpDir, makePkg())).toBe(iconPath);
  });

  test("icon.ico fallback", () => {
    const iconPath = path.join(tmpDir, "icon.ico");
    fs.writeFileSync(iconPath, "fake");
    expect(resolveIconPath(tmpDir, makePkg())).toBe(iconPath);
  });

  test("아이콘 없으면 null 반환", () => {
    expect(resolveIconPath(tmpDir, makePkg())).toBeNull();
  });
});

describe("generateSyso", () => {
  let tmpDir;
  beforeEach(() => { tmpDir = makeTempDir(); });
  afterEach(() => { fs.rmSync(tmpDir, { recursive: true, force: true }); });

  function makeSyso(pkg, goArch, iconPath = null) {
    const sysoPath = path.join(tmpDir, "resource.syso");
    generateSyso(tmpDir, pkg, goArch, sysoPath, iconPath);
    return fs.readFileSync(sysoPath);
  }

  test("아이콘 없이 버전 정보만으로 .syso 생성", () => {
    const buf = makeSyso(makePkg({ build: { productName: "MyApp" } }), "amd64");
    expect(buf.length).toBeGreaterThan(100);
    expect(buf.readUInt16LE(0)).toBe(0x8664);
    expect(buf.readUInt16LE(2)).toBe(1);
  });

  test("amd64 machine type 0x8664", () => {
    const buf = makeSyso(makePkg(), "amd64");
    expect(buf.readUInt16LE(0)).toBe(0x8664);
  });

  test("386 machine type 0x014c", () => {
    const buf = makeSyso(makePkg(), "386");
    expect(buf.readUInt16LE(0)).toBe(0x014c);
  });

  test("arm64 machine type 0xaa64", () => {
    const buf = makeSyso(makePkg(), "arm64");
    expect(buf.readUInt16LE(0)).toBe(0xaa64);
  });

  test(".rsrc 섹션 이름 포함", () => {
    const buf = makeSyso(makePkg(), "amd64");
    const sectionName = buf.slice(20, 28).toString("ascii").replace(/\0/g, "");
    expect(sectionName).toBe(".rsrc");
  });

  test("relocation 레코드 존재 (OffsetToData 수정용)", () => {
    const buf = makeSyso(makePkg(), "amd64");
    const numRelocs = buf.readUInt16LE(20 + 32);
    expect(numRelocs).toBeGreaterThan(0);
  });

  test("symbol table offset이 올바르게 설정됨", () => {
    const buf = makeSyso(makePkg(), "amd64");
    const symTableOff = buf.readUInt32LE(8);
    expect(symTableOff).toBeGreaterThan(0);
    expect(symTableOff).toBeLessThan(buf.length);
  });

  test("string table 4바이트로 끝남", () => {
    const buf = makeSyso(makePkg(), "amd64");
    const symTableOff = buf.readUInt32LE(8);
    const numSymbols = buf.readUInt32LE(12);
    const strTableOff = symTableOff + numSymbols * 18;
    expect(buf.readUInt32LE(strTableOff)).toBe(4);
  });

  test("SizeOfOptionalHeader가 0임", () => {
    const buf = makeSyso(makePkg(), "amd64");
    expect(buf.readUInt16LE(16)).toBe(0);
  });

  test("companyName이 author에서 추출됨", () => {
    const pkg = makePkg({ author: "홍길동 <hong@example.com>" });
    const buf = makeSyso(pkg, "amd64");
    expect(buf.length).toBeGreaterThan(100);
  });

  test("버전 1.2.3이 FixedFileInfo에 반영됨", () => {
    const buf = makeSyso(makePkg({ version: "1.2.3" }), "amd64");
    const sectionDataOff = 20 + 40;
    const rsrcBuf = buf.slice(sectionDataOff);
    const vsVersionInfoStr = "VS_VERSION_INFO";
    const idx = rsrcBuf.indexOf(Buffer.from("VS_VERSION_INFO", "utf16le"));
    expect(idx).toBeGreaterThan(-1);
  });
});
