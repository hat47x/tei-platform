import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { once } from "node:events";
import { createServer } from "node:http";
import { mkdtemp, mkdir, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { test } from "node:test";

const R = "a".repeat(40);
const T = "b".repeat(40);
const manifest = () => ({
  applicationId: "order_entry",
  handoverBaselineId: "B",
  baselineRepositoryRevision: R,
  targetBuildRevision: T,
});
const stop = async process => {
  if (!process || process.exitCode !== null || process.signalCode !== null) return;
  process.kill("SIGTERM");
  await Promise.race([
    once(process, "exit"),
    new Promise(resolve => setTimeout(resolve, 3000)),
  ]);
  if (process.exitCode === null && process.signalCode === null) process.kill("SIGKILL");
};
async function setup(t) {
  const root = await mkdtemp(join(tmpdir(), "tei-api-binding-"));
  const browser = join(root, "browser");
  const measurement = join(root, "measurement.json");
  await mkdir(browser);
  await writeFile(join(browser, "index.html"), '<form id="json-command" method="post" action="/scenario/submit"><script src="/browser-command.js"></script></form>');
  await writeFile(join(browser, "browser-command.js"), "void 0;");
  await writeFile(measurement, JSON.stringify({
    ...manifest(),
    browserAndApiSourceRevision: T,
    browserTransport: "same-origin-json",
  }));
  let exposed = manifest();
  let commandCount = 0;
  const server = createServer((req, res) => {
    if (req.url === "/manifest.json") {
      res.writeHead(200, { "content-type": "application/json" });
      res.end(JSON.stringify(exposed));
    } else if (req.url === "/scenario/submit") {
      commandCount++;
      res.writeHead(200, { "content-type": "application/json" });
      res.end(JSON.stringify({ result: { stage: "completed" } }));
    } else {
      res.writeHead(404);
      res.end();
    }
  });
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  const api = "http://127.0.0.1:" + server.address().port;
  let child;
  t.after(async () => {
    await stop(child);
    await new Promise(resolve => server.close(resolve));
    await rm(root, { recursive: true, force: true });
  });
  const start = async (includeMeasurement = true) => {
    child = spawn(process.execPath, [
      new URL("./browser-host.mjs", import.meta.url).pathname,
      browser, api, ...(includeMeasurement ? [measurement] : []),
    ], { stdio: ["ignore", "pipe", "pipe"] });
    let output = "";
    let error = "";
    child.stdout.on("data", chunk => { output += chunk; });
    child.stderr.on("data", chunk => { error += chunk; });
    for (let attempt = 0; attempt < 100; attempt++) {
      const found = /BROWSER_HOST=(http:\/\/127\.0\.0\.1:\d+)/.exec(output);
      if (found) return found[1];
      if (child.exitCode !== null || child.signalCode !== null) {
        throw Error("Browser host rejected: " + error.trim());
      }
      await new Promise(resolve => setTimeout(resolve, 30));
    }
    throw Error("diagnostic Host startup timeout: " + error.trim());
  };
  return {
    start,
    get count() { return commandCount; },
    changeManifest: next => { exposed = next; },
  };
}
async function submit(origin, extraHeaders = {}) {
  const response = await fetch(origin + "/scenario/submit", {
    method: "POST",
    headers: { Origin: origin, "Content-Type": "application/json", ...extraHeaders },
    body: '{"probe":true}',
  });
  return response.status;
}

test("matching B/R/T forwards only a valid-origin command", async t => {
  const f = await setup(t);
  const origin = await f.start();
  assert.equal(await submit(origin), 200);
  assert.equal(f.count, 1);
  assert.equal(await submit(origin, { Origin: "http://evil.invalid" }), 403);
  assert.equal(f.count, 1);
});

test("mismatched API manifest denies Host startup before Browser is exposed", async t => {
  const f = await setup(t);
  f.changeManifest({ ...manifest(), targetBuildRevision: "c".repeat(40) });
  await assert.rejects(f.start(), /API binding differs: targetBuildRevision/);
  assert.equal(f.count, 0);
});

test("API swap after startup denies POST without sending a command", async t => {
  const f = await setup(t);
  const origin = await f.start();
  f.changeManifest({ ...manifest(), baselineRepositoryRevision: "d".repeat(40) });
  assert.equal(await submit(origin), 502);
  assert.equal(f.count, 0);
  f.changeManifest(manifest());
  assert.equal(await submit(origin), 200);
  assert.equal(f.count, 1);
});

test("missing expected build measurement refuses JSON Browser startup", async t => {
  const f = await setup(t);
  await assert.rejects(f.start(false), /JSON Browser requires its existing build measurement/);
  assert.equal(f.count, 0);
});
