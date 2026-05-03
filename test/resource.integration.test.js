"use strict";

const fs = require("fs");
const os = require("os");
const path = require("path");
const { execSync } = require("child_process");
const { generateSyso } = require("../lib/resource");

function makeTempDir() {
  return fs.mkdtempSync(path.join(os.tmpdir(), "wpp-resource-integration-"));
}

function getGoversioninfoBin() {
  try {
    const gopath = execSync("go env GOPATH", { encoding: "utf8" }).trim();
    const candidates = [
      path.join(gopath, "bin", "goversioninfo.exe"),
      path.join(gopath, "bin", "goversioninfo"),
    ];
    return candidates.find(fs.existsSync) || null;
  } catch {
    return null;
  }
}

function ensureGoversioninfo() {
  if (!getGoversioninfoBin()) {
    console.log("Installing goversioninfo...");
    execSync("go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest", { stdio: "inherit" });
  }
}

let goversioninfoCounter = 0;

function generateWithGoversioninfo(tmpDir, pkg, goArch, iconPath) {
  const bin = getGoversioninfoBin();
  const productName = (pkg.build && pkg.build.productName) || pkg.name || "";
  const ver = String(pkg.version || "0.0.0").replace(/[^0-9.]/g, "").split(".").map(Number);
  while (ver.length < 4) ver.push(0);

  const versionInfo = {
    FixedFileInfo: {
      FileVersion: { Major: ver[0], Minor: ver[1], Patch: ver[2], Build: ver[3] },
      ProductVersion: { Major: ver[0], Minor: ver[1], Patch: ver[2], Build: ver[3] },
      FileFlagsMask: "3f", FileFlags: "00", FileOS: "040004", FileType: "01", FileSubType: "00",
    },
    StringFileInfo: {
      CompanyName: "",
      FileDescription: pkg.description || productName,
      FileVersion: pkg.version || "0.0.0",
      InternalName: productName,
      LegalCopyright: "",
      OriginalFilename: productName + ".exe",
      ProductName: productName,
      ProductVersion: pkg.version || "0.0.0",
    },
    VarFileInfo: { Translation: { LangID: "0409", CharsetID: "04B0" } },
  };

  if (iconPath) versionInfo.IconPath = iconPath;

  const id = ++goversioninfoCounter;
  const jsonPath = path.join(tmpDir, `versioninfo-${id}.json`);
  const sysoPath = path.join(tmpDir, `goversioninfo-${id}.syso`);
  fs.writeFileSync(jsonPath, JSON.stringify(versionInfo, null, 2));

  const archFlag = goArch === "amd64" ? "-64" : goArch === "arm64" ? "-arm" : "";
  execSync(`"${bin}" ${archFlag} -o "${sysoPath}" "${jsonPath}"`, { stdio: "inherit" });
  fs.unlinkSync(jsonPath);
  return sysoPath;
}

function parseCoffHeader(buf) {
  return {
    machine: buf.readUInt16LE(0),
    numberOfSections: buf.readUInt16LE(2),
    timeDateStamp: buf.readUInt32LE(4),
    pointerToSymbolTable: buf.readUInt32LE(8),
    numberOfSymbols: buf.readUInt32LE(12),
    sizeOfOptionalHeader: buf.readUInt16LE(16),
    characteristics: buf.readUInt16LE(18),
  };
}

function parseCoffSections(buf) {
  const numSections = buf.readUInt16LE(2);
  const sections = [];
  for (let i = 0; i < numSections; i++) {
    const off = 20 + i * 40;
    const name = buf.slice(off, off + 8).toString("ascii").replace(/\0/g, "");
    const rawSize = buf.readUInt32LE(off + 16);
    const rawOff = buf.readUInt32LE(off + 20);
    const relocOff = buf.readUInt32LE(off + 24);
    const numRelocs = buf.readUInt16LE(off + 32);
    sections.push({ name, rawSize, rawOff, relocOff, numRelocs });
  }
  return sections;
}

function extractRsrc(buf) {
  const sections = parseCoffSections(buf);
  const rsrc = sections.find(s => s.name === ".rsrc");
  if (!rsrc) return null;
  return buf.slice(rsrc.rawOff, rsrc.rawOff + rsrc.rawSize);
}

describe("resource.js vs goversioninfo integration", () => {
  let tmpDir;

  beforeAll(() => {
    ensureGoversioninfo();
    tmpDir = makeTempDir();
  });

  afterAll(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  test("COFF machine type이 goversioninfo와 동일", () => {
    const pkg = { name: "TestApp", version: "1.2.3", description: "Test" };
    const ourSyso = path.join(tmpDir, "ours.syso");
    const theirSyso = generateWithGoversioninfo(tmpDir, pkg, "amd64", null);

    generateSyso(tmpDir, pkg, "amd64", ourSyso, null);

    const ourBuf = fs.readFileSync(ourSyso);
    const theirBuf = fs.readFileSync(theirSyso);

    expect(ourBuf.readUInt16LE(0)).toBe(theirBuf.readUInt16LE(0));
  });

  test("COFF 헤더 필드가 goversioninfo와 동일", () => {
    const pkg = { name: "TestApp", version: "1.2.3", description: "Test" };
    const ourSyso = path.join(tmpDir, "ours-hdr.syso");
    const theirSyso = generateWithGoversioninfo(tmpDir, pkg, "amd64", null);

    generateSyso(tmpDir, pkg, "amd64", ourSyso, null);

    const ourHdr = parseCoffHeader(fs.readFileSync(ourSyso));
    const theirHdr = parseCoffHeader(fs.readFileSync(theirSyso));

    expect(ourHdr.machine).toBe(theirHdr.machine);
    expect(ourHdr.numberOfSections).toBe(theirHdr.numberOfSections);
    expect(ourHdr.sizeOfOptionalHeader).toBe(theirHdr.sizeOfOptionalHeader);
    expect(ourHdr.sizeOfOptionalHeader).toBe(0);
    expect(ourHdr.numberOfSymbols).toBe(theirHdr.numberOfSymbols);
  });

  test("생성된 .syso로 go build 성공", () => {
    const pkg = { name: "TestApp", version: "1.2.3", description: "Test" };
    const sysoPath = path.join(tmpDir, "resource.syso");
    generateSyso(tmpDir, pkg, "amd64", sysoPath, null);

    const goSrc = path.join(tmpDir, "gobuild");
    fs.mkdirSync(goSrc);
    fs.writeFileSync(path.join(goSrc, "go.mod"), "module testbuild\n\ngo 1.21\n");
    fs.writeFileSync(path.join(goSrc, "main.go"), "package main\n\nfunc main() {}\n");
    fs.copyFileSync(sysoPath, path.join(goSrc, "resource.syso"));

    const exePath = path.join(tmpDir, "testbuild.exe");
    execSync(`go build -ldflags="-s -w" -o "${exePath}" .`, {
      cwd: goSrc,
      env: { ...process.env, GOARCH: "amd64", GOOS: "windows" },
      stdio: "pipe",
    });

    expect(fs.existsSync(exePath)).toBe(true);
  });

  test("relocation 수가 goversioninfo와 동일", () => {
    const pkg = { name: "TestApp", version: "1.2.3", description: "Test" };
    const ourSyso = path.join(tmpDir, "ours2.syso");
    const theirSyso = generateWithGoversioninfo(tmpDir, pkg, "amd64", null);

    generateSyso(tmpDir, pkg, "amd64", ourSyso, null);

    const ourBuf = fs.readFileSync(ourSyso);
    const theirBuf = fs.readFileSync(theirSyso);

    const ourSections = parseCoffSections(ourBuf);
    const theirSections = parseCoffSections(theirBuf);

    const ourRsrc = ourSections.find(s => s.name === ".rsrc");
    const theirRsrc = theirSections.find(s => s.name === ".rsrc");

    expect(ourRsrc).toBeTruthy();
    expect(theirRsrc).toBeTruthy();
    expect(ourRsrc.numRelocs).toBe(theirRsrc.numRelocs);
  });

  test(".rsrc 섹션 크기가 goversioninfo와 유사 (±20%)", () => {
    const pkg = { name: "TestApp", version: "1.2.3", description: "Test" };
    const ourSyso = path.join(tmpDir, "ours3.syso");
    const theirSyso = generateWithGoversioninfo(tmpDir, pkg, "amd64", null);

    generateSyso(tmpDir, pkg, "amd64", ourSyso, null);

    const ourRsrc = extractRsrc(fs.readFileSync(ourSyso));
    const theirRsrc = extractRsrc(fs.readFileSync(theirSyso));

    expect(ourRsrc).toBeTruthy();
    expect(theirRsrc).toBeTruthy();

    const ratio = ourRsrc.length / theirRsrc.length;
    expect(ratio).toBeGreaterThan(0.8);
    expect(ratio).toBeLessThan(1.2);
  });

  test("VS_VERSION_INFO 문자열이 .rsrc에 포함됨", () => {
    const pkg = { name: "TestApp", version: "2.0.0", description: "Test" };
    const ourSyso = path.join(tmpDir, "ours4.syso");
    generateSyso(tmpDir, pkg, "amd64", ourSyso, null);

    const rsrc = extractRsrc(fs.readFileSync(ourSyso));
    const marker = Buffer.from("VS_VERSION_INFO", "utf16le");
    expect(rsrc.indexOf(marker)).toBeGreaterThan(-1);
  });

  test("ProductName이 .rsrc에 포함됨", () => {
    const pkg = { name: "MyProduct", version: "1.0.0", build: { productName: "MyProduct" } };
    const ourSyso = path.join(tmpDir, "ours5.syso");
    generateSyso(tmpDir, pkg, "amd64", ourSyso, null);

    const rsrc = extractRsrc(fs.readFileSync(ourSyso));
    const marker = Buffer.from("MyProduct", "utf16le");
    expect(rsrc.indexOf(marker)).toBeGreaterThan(-1);
  });
});
