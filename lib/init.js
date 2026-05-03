"use strict";

const path = require("path");
const fs = require("fs");

const HOOK = "windows-portable-packager/hooks/afterAllArtifactBuild";

function runInit(cwd, { readLine } = {}) {
  const pkgPath = path.resolve(cwd, "package.json");

  if (!fs.existsSync(pkgPath)) {
    return { code: 1, message: "windows-portable-packager init: package.json not found in current directory." };
  }

  let pkg;
  try {
    pkg = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
  } catch (err) {
    return { code: 1, message: "windows-portable-packager init: failed to parse package.json: " + err.message };
  }

  if (!pkg.build) {
    pkg.build = {};
  }

  if (pkg.build.afterAllArtifactBuild === HOOK) {
    return { code: 0, message: "windows-portable-packager: already configured in package.json, nothing to do." };
  }

  if (pkg.build.afterAllArtifactBuild && pkg.build.afterAllArtifactBuild !== HOOK) {
    const answer = readLine ? readLine() : "";
    if (!answer.match(/^y(es)?$/i)) {
      return { code: 0, message: "Aborted." };
    }
  }

  pkg.build.afterAllArtifactBuild = HOOK;
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n", "utf8");

  return { code: 0, message: "windows-portable-packager: configured successfully." };
}

module.exports = { runInit, HOOK };
