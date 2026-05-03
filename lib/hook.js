"use strict";

const path = require("path");
const fs = require("fs");

function resolveConfig(pkgJsonPath) {
  const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, "utf8"));
  const ppConfig = pkg.portablePackager || {};
  return {
    version: pkg.version,
    productName: (pkg.build && pkg.build.productName) || pkg.name,
    exeName: ppConfig.exeName || ((pkg.build && pkg.build.productName) || pkg.name) + ".exe",
    goArch: ppConfig.arch || "amd64",
  };
}

function resolveBin(packagerDir, goArch) {
  const candidates = [
    path.join(packagerDir, "dist", "windows-portable-packager-" + goArch + ".exe"),
    path.join(packagerDir, "dist", "windows-portable-packager.exe"),
  ];
  return candidates.find(fs.existsSync) || null;
}

async function runHook(buildResult, { pkgJsonPath, packagerDir, exec, execGo } = {}) {
  const outDir = buildResult.outDir;
  const unpackedDir = path.join(outDir, "win-unpacked");

  if (!fs.existsSync(unpackedDir)) {
    return { skipped: true, reason: "win-unpacked directory not found" };
  }

  const resolvedPkgJson = pkgJsonPath || path.resolve("package.json");
  const { version, productName, exeName, goArch } = resolveConfig(resolvedPkgJson);

  if (!fs.existsSync(path.join(unpackedDir, exeName))) {
    return { skipped: true, reason: "exe not found: " + exeName };
  }

  const resolvedPackagerDir = packagerDir || path.resolve(__dirname, "..");
  const embedPath = path.join(resolvedPackagerDir, "embedded", "app.kbpkg");
  const finalExe = path.join(outDir, productName + "-" + version + "-" + goArch + ".exe");

  const packBin = resolveBin(resolvedPackagerDir, goArch);
  if (!packBin) {
    return { skipped: true, reason: "packager binary not found" };
  }

  try {
    const packArgs = [
      "pack", unpackedDir,
      "-app", productName,
      "-v", version,
      "-arch", goArch,
      "-exe", exeName,
      "-o", embedPath,
    ];

    if (exec) {
      exec(packBin, packArgs);
    } else {
      require("child_process").execFileSync(packBin, packArgs, { stdio: "inherit", cwd: process.cwd() });
    }

    const goCmd = `go build -ldflags="-s -w" -o "${finalExe}" .`;
    if (execGo) {
      execGo(goCmd, resolvedPackagerDir, goArch);
    } else {
      require("child_process").execSync(goCmd, {
        stdio: "inherit",
        cwd: resolvedPackagerDir,
        env: { ...process.env, GOARCH: goArch, GOOS: "windows" },
      });
    }

    return { skipped: false, finalExe };
  } catch (err) {
    return { skipped: true, reason: "build failed: " + err.message };
  } finally {
    fs.writeFileSync(embedPath, "PLACEHOLDER - replaced during build\n");
  }
}

module.exports = { runHook, resolveConfig, resolveBin };
