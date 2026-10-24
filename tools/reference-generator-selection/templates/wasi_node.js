"use strict";

const { WASI } = require("node:wasi");
const fs = require("node:fs");

if (process.argv.length < 3) {
  console.error("usage: node wasi_node.js <wasm>");
  process.exit(2);
}
const wasmPath = process.argv[2];
const wasi = new WASI({
  version: "preview1",
  args: [wasmPath],
  env: process.env,
  returnOnExit: true,
});
WebAssembly.instantiate(fs.readFileSync(wasmPath), wasi.getImportObject())
  .then(({ instance }) => {
    process.exit(wasi.start(instance));
  })
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
