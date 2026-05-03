const { runHook } = require("../lib/hook");

module.exports = async function afterAllArtifactBuild(buildResult) {
  const result = await runHook(buildResult);

  if (result.skipped) {
    console.warn("windows-portable-packager: skipped — " + result.reason);
    return [];
  }

  console.log("windows-portable-packager: launcher created → " + result.finalExe);
  return [result.finalExe];
};
