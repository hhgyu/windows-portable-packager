"use strict";

const path = require("path");
const fs = require("fs");
const { resolveIconPath, generateSyso } = require("./resource");

const SPLASH_EXTS = new Set([".png", ".apng", ".gif", ".jpg", ".jpeg"]);

function resolveConfig(pkgJsonPath) {
  const pkg = JSON.parse(fs.readFileSync(pkgJsonPath, "utf8"));
  const ppConfig = pkg.portablePackager || {};
  return {
    version: pkg.version,
    productName: (pkg.build && pkg.build.productName) || pkg.name,
    exeName: ppConfig.exeName || ((pkg.build && pkg.build.productName) || pkg.name) + ".exe",
    goArch: ppConfig.arch || "amd64",
    splashPath: ppConfig.splash || null,
    compression: ["zstd", "gzip", "xz"].includes(ppConfig.compression) ? ppConfig.compression : "zstd",
    level: ppConfig.level || 0,
  };
}

function resolveSplashPath(projectDir, splashPath) {
  if (!splashPath) return null;
  const ext = path.extname(splashPath).toLowerCase();
  if (!SPLASH_EXTS.has(ext)) return null;
  const resolved = path.resolve(projectDir, splashPath);
  return fs.existsSync(resolved) ? resolved : null;
}

function resolveBin(packagerDir, goArch) {
  const candidates = [
    path.join(packagerDir, "dist", "windows-portable-packager-" + goArch + ".exe"),
    path.join(packagerDir, "dist", "windows-portable-packager.exe"),
  ];
  return candidates.find(fs.existsSync) || null;
}

async function runHook(buildResult, { pkgJsonPath, packagerDir, exec, execGo, generateSysoFn } = {}) {
  const outDir = buildResult.outDir;
  const unpackedDir = path.join(outDir, "win-unpacked");

  if (!fs.existsSync(unpackedDir)) {
    return { skipped: true, reason: "win-unpacked directory not found" };
  }

  const projectDir = buildResult.configuration && buildResult.configuration.directories && buildResult.configuration.directories.output
    ? path.resolve(outDir, "..")
    : (buildResult.projectDir || path.resolve(outDir, ".."));

  const resolvedPkgJson = pkgJsonPath || path.join(projectDir, "package.json");
  const pkg = JSON.parse(fs.readFileSync(resolvedPkgJson, "utf8"));
  const { version, productName, exeName, goArch, splashPath, compression, level } = resolveConfig(resolvedPkgJson);

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
    const resolvedSplash = resolveSplashPath(projectDir, splashPath);
    const splashDatPath = path.join(resolvedPackagerDir, "embedded", "splash.dat");
    const splashExtPath = path.join(resolvedPackagerDir, "embedded", "splash.ext");

    if (resolvedSplash) {
      const ext = path.extname(resolvedSplash).toLowerCase();
      fs.copyFileSync(resolvedSplash, splashDatPath);
      fs.writeFileSync(splashExtPath, ext, "utf8");
    } else {
      fs.writeFileSync(splashDatPath, "PLACEHOLDER\n", "utf8");
      fs.writeFileSync(splashExtPath, "", "utf8");
    }

    const packArgs = [
      "pack",
      "-app", productName,
      "-v", version,
      "-arch", goArch,
      "-exe", exeName,
      "-o", embedPath,
      "-compression", compression,
    ];
    if (level > 0) packArgs.push("-level", String(level));
    packArgs.push(unpackedDir);

    if (exec) {
      exec(packBin, packArgs);
    } else {
      require("child_process").execFileSync(packBin, packArgs, { stdio: "inherit", cwd: process.cwd() });
    }

    const sysoPath = path.join(resolvedPackagerDir, "resource.syso");
    const iconPath = resolveIconPath(projectDir, pkg);
    const sysoFn = generateSysoFn || generateSyso;
    sysoFn(projectDir, pkg, goArch, sysoPath, iconPath);

    const goCmd = `go build -ldflags="-s -w -H=windowsgui" -o "${finalExe}" .`;
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
    fs.writeFileSync(path.join(resolvedPackagerDir, "embedded", "splash.dat"), "PLACEHOLDER\n", "utf8");
    fs.writeFileSync(path.join(resolvedPackagerDir, "embedded", "splash.ext"), "", "utf8");
    const sysoPath = path.join(resolvedPackagerDir, "resource.syso");
    if (fs.existsSync(sysoPath)) fs.unlinkSync(sysoPath);
  }
}

module.exports = { runHook, resolveConfig, resolveBin };
