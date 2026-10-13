// Reference-only WASI Preview 1 host for the standalone read-only quote guest.
// The existing process carrier remains an outer test harness, not the Wasm ABI.
// No guest filesystem preopens, inherited environment, or custom host imports.
import { readFile } from "node:fs/promises";
import { WASI } from "node:wasi";

if (process.argv.length !== 3) {
  process.stderr.write("expected one Wasm artifact path\n");
  process.exit(2);
}
try {
  const wasi = new WASI({
    version: "preview1",
    args: [],
    env: {},
    preopens: {},
    returnOnExit: true,
  });
  const module = await WebAssembly.compile(await readFile(process.argv[2]));
  const instance = await WebAssembly.instantiate(module, wasi.getImportObject());
  const status = wasi.start(instance);
  if (status !== 0) process.exitCode = status;
} catch {
  // Do not propagate guest request contents or runtime-internal diagnostics.
  process.stderr.write("WASI quote reference execution failed\n");
  process.exitCode = 2;
}
