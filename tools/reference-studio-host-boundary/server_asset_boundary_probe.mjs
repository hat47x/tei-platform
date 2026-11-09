// A narrow negative HTTP probe for the fixed PoC A Wails Server build.
// Run explicitly AFTER starting a pinned server; this is not a test of
// authenticated Wails RPC, every possible URI, or all information leakage.
//
// TEI_STUDIO_BOUNDARY_URL=http://127.0.0.1:43123 \
//   node --test tools/reference-studio-host-boundary/server_asset_boundary_probe.mjs
import assert from "node:assert/strict";
import {randomBytes} from "node:crypto";
import {mkdtempSync, rmSync, writeFileSync} from "node:fs";
import http from "node:http";
import {tmpdir} from "node:os";
import {join} from "node:path";
import test from "node:test";

function loopbackPort() {
  const input = process.env.TEI_STUDIO_BOUNDARY_URL;
  assert.ok(input, "set TEI_STUDIO_BOUNDARY_URL to the running fixed Wails Server");
  const url = new URL(input);
  assert.equal(url.protocol, "http:", "only local plain-HTTP PoC Server is supported");
  assert.equal(url.hostname, "127.0.0.1", "refuse probes outside 127.0.0.1");
  assert.equal(url.pathname, "/", "base URL must be the Server root");
  assert.equal(url.search, "", "base URL must not contain a query");
  assert.equal(url.hash, "", "base URL must not contain a fragment");
  const port = Number(url.port);
  assert.ok(Number.isInteger(port) && port > 0 && port <= 65535, "explicit port required");
  return port;
}

function getRaw(port, path) {
  return new Promise((resolve, reject) => {
    const request = http.get({hostname: "127.0.0.1", port, path, timeout: 5000}, (response) => {
      let size = 0;
      const chunks = [];
      response.on("data", (chunk) => {
        size += chunk.length;
        if (size > 2 * 1024 * 1024) {
          response.destroy(Error("unexpectedly large HTTP response"));
          return;
        }
        chunks.push(chunk);
      });
      response.on("error", reject);
      response.on("end", () => resolve({
        status: response.statusCode,
        body: Buffer.concat(chunks).toString("utf8"),
      }));
    });
    request.on("timeout", () => request.destroy(Error("fixed Server did not respond")));
    request.on("error", reject);
  });
}

test("pinned Wails Server cannot serve an unrelated private file via asset-like GET requests", async (t) => {
  const port = loopbackPort();
  const health = await getRaw(port, "/health");
  assert.equal(health.status, 200, "fixed Server must be running and healthy");
  assert.match(health.body, /"status"\s*:\s*"ok"/);

  // This sentinel is OUTSIDE the configured Workspace and embedded assets.
  // The test itself creates and cleans it; no real secret file is opened.
  const directory = mkdtempSync(join(tmpdir(), "tei-studio-unserved-file-"));
  t.after(() => rmSync(directory, {recursive: true, force: true}));
  const filename = join(directory, "secret-" + randomBytes(8).toString("hex") + ".txt");
  const sentinel = "TEI_STUDIO_PRIVATE_BYTES_" + randomBytes(20).toString("hex");
  writeFileSync(filename, sentinel, {mode: 0o600});
  const nativePath = filename.replaceAll("\\", "/");
  const trimmed = nativePath.replace(/^\/+/, "");
  const encoded = encodeURIComponent(nativePath);
  const paths = [
    "/?file=" + encoded,
    "/?path=" + encoded,
    "/" + trimmed,
    "/../../" + trimmed,
    "/%2e%2e/%2e%2e/" + trimmed,
    "/" + encoded,
    "/wails/runtime.js?file=" + encoded,
  ];
  for (const [index, path] of paths.entries()) {
    await t.test("private-file-probe-" + (index + 1), async () => {
      const response = await getRaw(port, path);
      // A SPA may intentionally return the index for unknown paths.
      // Do not require 404; require that it never responds with private bytes.
      assert.equal(response.body.includes(sentinel), false,
        "HTTP static-asset route disclosed the unembedded private file");
    });
  }
});
