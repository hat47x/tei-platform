import { build } from "esbuild";
import { copyFile, mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
const here = dirname(fileURLToPath(import.meta.url));
await mkdir(resolve(here, "dist"), { recursive: true });
await build({
  absWorkingDir: here,
  entryPoints: [resolve(here, "main.ts")],
  outfile: resolve(here, "dist/main.js"),
  bundle: true,
  format: "esm",
  platform: "browser",
  target: "es2022",
  nodePaths: [resolve(here, "node_modules")],
  metafile: true,
}).then(async (result) => {
  await writeFile(
    resolve(here, "dist/dependencies.json"),
    JSON.stringify(result.metafile, null, 2) + "\n",
  );
  const packages = new Map();
  for (const input of Object.keys(result.metafile.inputs)) {
    if (!input.split(/[\\/]/).includes("node_modules")) continue;
    let folder = dirname(resolve(here, input));
    for (;;) {
      let manifest;
      try {
        manifest = JSON.parse(
          await readFile(resolve(folder, "package.json"), "utf8"),
        );
      } catch {}
      if (manifest?.name) {
        let notice;
        for (const name of [
          "LICENSE",
          "LICENSE.txt",
          "LICENSE.md",
          "license",
          "license.txt",
        ]) {
          try {
            notice = await readFile(resolve(folder, name), "utf8");
            break;
          } catch {}
        }
        if (!notice)
          throw Error(
            `License text missing for bundled package ${manifest.name}`,
          );
        packages.set(`${manifest.name}@${manifest.version}`, notice);
        break;
      }
      const parent = dirname(folder);
      if (parent === folder) throw Error(`Package not found for ${input}`);
      folder = parent;
    }
  }
  await writeFile(
    resolve(here, "dist/THIRD_PARTY_NOTICES.txt"),
    [...packages]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([name, notice]) => `${name}\n${"=".repeat(name.length)}\n${notice}`)
      .join("\n\n"),
  );
});
const mainPath = resolve(here, "dist/main.js");
const mainJS = await readFile(mainPath, "utf8");
const normalizedMainJS = mainJS.replace(/\/\*\*[\s\S]*?\*\//g, (comment) =>
  comment.replace(/[ \t]+$/gm, ""),
);
if (normalizedMainJS !== mainJS) await writeFile(mainPath, normalizedMainJS);
// Temporary diagnostic on this branch only; remove before integration.
const ix = normalizedMainJS.indexOf("function showRequiredPreview()");
const first = normalizedMainJS.indexOf("    output.textContent = JSON.stringify(draft.edit, null, 2);", ix);
const last = normalizedMainJS.indexOf("\n  } catch (error) {", first);
if (first < 0 || last < 0) throw Error("MISSING_PREVIEW_CODE");
console.log("TEI_WORKBENCH_EXACT_ENTRY=" + JSON.stringify(normalizedMainJS.slice(first, last)));
await copyFile(resolve(here, "index.html"), resolve(here, "dist/index.html"));
await copyFile(resolve(here, "style.css"), resolve(here, "dist/style.css"));
