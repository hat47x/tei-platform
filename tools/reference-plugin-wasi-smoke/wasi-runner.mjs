// Execute the reference-only WASI build using Node's built-in preview1 host.
// This is an execution probe, not a browser runtime or a TEI production Host.
import { readFileSync } from "node:fs";
import { WASI } from "node:wasi";

if (process.argv.length < 3) {
  process.stderr.write("usage: node wasi-runner.mjs <wasm-file> [plugin arguments...]\n");
  process.exit(2);
}

const wasi = new WASI({
  version: "preview1",
  args: ["reference-process-executor", ...process.argv.slice(3)],
  env: {},
  preopens: {},
  returnOnExit: true,
});
const module = await WebAssembly.compile(readFileSync(process.argv[2]));
const instance = await WebAssembly.instantiate(module, {
  wasi_snapshot_preview1: wasi.wasiImport,
});
// Propagate the plugin\'s WASI proc_exit result to the test host itself.
process.exitCode = wasi.start(instance);
