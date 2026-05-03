#!/usr/bin/env node

const path = require("path");
const { execFileSync } = require("child_process");
const fs = require("fs");

const args = process.argv.slice(2);

if (args[0] === "init") {
  const { runInit } = require("../lib/init");
  const result = runInit(process.cwd(), { readLine: readLineSync });
  console.log(result.message);
  process.exit(result.code);
}

const archMap = { x64: "amd64", ia32: "386", arm64: "arm64" };
const arch = archMap[process.arch] || process.arch;
const binPath = path.resolve(__dirname, "..", "dist", `windows-portable-packager-${arch}.exe`);

const fallbackPath = path.resolve(__dirname, "..", "dist", "windows-portable-packager.exe");
const resolvedPath = fs.existsSync(binPath) ? binPath : (fs.existsSync(fallbackPath) ? fallbackPath : null);

if (!resolvedPath) {
  console.error(
    "windows-portable-packager: binary not found.\n" +
    "Run 'npm run build' or 'npm run build:all' first."
  );
  process.exit(1);
}

try {
  execFileSync(resolvedPath, args, {
    stdio: "inherit",
    cwd: process.cwd(),
  });
} catch (err) {
  process.exit(err.status || 1);
}

function readLineSync() {
  const buf = Buffer.alloc(64);
  let result = "";
  try {
    const fd = fs.openSync("/dev/stdin", "rs");
    const n = fs.readSync(fd, buf, 0, buf.length, null);
    fs.closeSync(fd);
    result = buf.slice(0, n).toString().trim();
  } catch {
    try {
      result = require("child_process")
        .execSync("choice /C YN /N /M \"\"", { stdio: ["inherit", "pipe", "pipe"] })
        .toString()
        .trim();
      result = result === "Y" ? "y" : "n";
    } catch {
      result = "";
    }
  }
  return result;
}
