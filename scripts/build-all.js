"use strict";

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const archs = ["amd64", "386", "arm64"];
const outDir = path.resolve(__dirname, "..", "dist");

fs.mkdirSync(outDir, { recursive: true });

for (const arch of archs) {
  const out = path.join(outDir, `windows-portable-packager-${arch}.exe`);
  console.log(`Building ${arch} -> ${out}`);

  execSync(`go build -ldflags="-s -w" -o "${out}" .`, {
    stdio: "inherit",
    cwd: path.resolve(__dirname, ".."),
    env: { ...process.env, GOARCH: arch, GOOS: "windows" },
  });

  const sizeMB = (fs.statSync(out).size / 1024 / 1024).toFixed(2);
  console.log(`  OK (${sizeMB} MB)`);
}

console.log("\nAll builds complete.");
