"use strict";

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

function resolveIconPath(projectDir, pkg) {
  const candidates = [];

  const winIcon = pkg.build && pkg.build.win && pkg.build.win.icon;
  if (winIcon) candidates.push(path.resolve(projectDir, winIcon));

  const buildIcon = pkg.build && pkg.build.icon;
  if (buildIcon) {
    if (!buildIcon.endsWith(".ico")) {
      candidates.push(path.resolve(projectDir, buildIcon + ".ico"));
    }
    candidates.push(path.resolve(projectDir, buildIcon));
  }

  candidates.push(path.join(projectDir, "build", "icon.ico"));
  candidates.push(path.join(projectDir, "icon.ico"));

  return candidates.find(fs.existsSync) || null;
}

function generateSyso(icoPath, sysoPath, goArch) {
  const rsrcBin = resolveRsrcBin();
  if (!rsrcBin) {
    throw new Error(
      "windows-portable-packager: 'rsrc' tool not found.\n" +
      "Install it with: go install github.com/akavel/rsrc@latest"
    );
  }

  execSync(`"${rsrcBin}" -arch ${goArch} -ico "${icoPath}" -o "${sysoPath}"`, {
    stdio: "inherit",
  });
}

function resolveRsrcBin() {
  const candidates = [];

  try {
    const gobin = execSync("go env GOBIN", { encoding: "utf8" }).trim();
    if (gobin) candidates.push(path.join(gobin, "rsrc.exe"), path.join(gobin, "rsrc"));
  } catch {}

  try {
    const gopath = execSync("go env GOPATH", { encoding: "utf8" }).trim();
    if (gopath) candidates.push(path.join(gopath, "bin", "rsrc.exe"), path.join(gopath, "bin", "rsrc"));
  } catch {}

  return candidates.find(fs.existsSync) || null;
}

function installRsrc() {
  execSync("go install github.com/akavel/rsrc@latest", { stdio: "inherit" });
}

function ensureRsrc() {
  if (!resolveRsrcBin()) {
    installRsrc();
  }
}

module.exports = { resolveIconPath, generateSyso, ensureRsrc };
