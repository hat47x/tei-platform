// Diagnostic real-Chromium replay of a generated TEI Browser and a separate
// tei_order_only,tei_api_only Go process. This is *not* production security,
// visual acceptance, IdP, CSRF, or full Gate C equivalence evidence.
import assert from "node:assert/strict";
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtemp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { createServer } from "node:net";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
import { pathToFileURL } from "node:url";

if (process.argv.length !== 5) {
  throw Error("usage: node real_browser_replay.mjs ARTIFACT_DIR PLAYWRIGHT_INSTALL_ROOT SCREENSHOT_DIR");
}
const root = resolve(process.argv[2]);
const playwrightRoot = resolve(process.argv[3]);
const screenshots = resolve(process.argv[4]);
const { chromium } = await import(pathToFileURL(join(playwrightRoot, "node_modules/playwright-core/index.mjs")).href);
const { execFileSync } = await import("node:child_process");
const revision = execFileSync("git", ["rev-parse", "HEAD"], { encoding: "utf8" }).trim();
const measurement = JSON.parse(await readFile(join(root, "measurement.json"), "utf8"));
assert.equal(measurement.applicationId, "order_entry");
assert.equal(measurement.browserTransport, "same-origin-json");
assert.equal(measurement.browserAndApiSourceRevision, revision);
const frozenAPI = measurement.apiWorldMode === "digest-bound-frozen-world";
assert.deepEqual(measurement.apiBuildTags, frozenAPI
  ? ["tei_order_only", "tei_api_only", "tei_frozen_world"]
  : ["tei_order_only", "tei_api_only"]);

const digest = async path => createHash("sha256").update(await readFile(path)).digest("hex");
for (const [name, key] of [
  ["browser/index.html", "browserSHA256"],
  ["browser/browser-command.js", "browserScriptSHA256"],
  ["api/order-entry", "apiSHA256"],
]) {
  assert.equal(await digest(join(root, name)), measurement[key], name + " failed pinned digest");
}
await mkdir(screenshots, { recursive: true });

const reservePort = async () => {
  const server = createServer();
  await new Promise((yes, no) => server.once("error", no).listen(0, "127.0.0.1", yes));
  const port = server.address().port;
  await new Promise((yes, no) => server.close(err => err ? no(err) : yes()));
  return port;
};
const sleep = millis => new Promise(resolve => setTimeout(resolve, millis));
const wait = async (url, child, attempts = 100) => {
  for (let i = 0; i < attempts; i++) {
    if (child.exitCode !== null || child.signalCode !== null) throw Error("process exited before ready");
    try {
      const response = await fetch(url, { signal: AbortSignal.timeout(1800) });
      if (response.ok) return;
    } catch (_) { /* until deadline */ }
    await sleep(100);
  }
  throw Error("diagnostic process startup failed");
};
const stop = async child => {
  if (!child || child.exitCode !== null || child.signalCode !== null) return;
  child.kill("SIGTERM");
  await Promise.race([
    new Promise(resolve => child.once("exit", resolve)),
    sleep(3500).then(() => { if (child.exitCode === null) child.kill("SIGKILL"); }),
  ]);
};

let go, host, browser;
const records = [];
const errors = [];
try {
  const port = await reservePort();
  const api = "http://127.0.0.1:" + port;
  go = spawn(join(root, "api/order-entry"), [
    "-workspace", frozenAPI ? join(root, "api") : resolve("reference/workspace"),
    "-legacy-users", resolve("reference/legacy-auth/users.json"),
    "-legacy-authorization", resolve("reference/legacy-auth/authorization.json"),
    "-application", "order_entry",
    "-baseline-id", measurement.handoverBaselineId,
    "-baseline-revision", measurement.baselineRepositoryRevision,
    "-target-revision", revision,
    "-listen", "127.0.0.1:" + port,
  ], { stdio: "ignore" });
  await wait(api + "/manifest.json", go);
  host = spawn("node", [
    "tools/reference-first-web-two-unit-e1/browser-host.mjs",
    join(root, "browser"), api,
  ], { stdio: ["ignore", "pipe", "ignore"] });
  let data = "";
  host.stdout.on("data", chunk => { data += String(chunk); });
  let origin;
  for (let i = 0; i < 100; i++) {
    const match = /^BROWSER_HOST=(http:\/\/127\.0\.0\.1:\d+)$/m.exec(data);
    if (match) { origin = match[1]; break; }
    if (host.exitCode !== null) throw Error("Browser host quit");
    await sleep(100);
  }
  assert.ok(origin, "Browser host failed to report bound address");
  await wait(origin + "/", host);

  // Use the system Chrome installed on the GitHub-hosted Ubuntu runner.
  // No browser download, no external site, no trace of user credentials.
  browser = await chromium.launch({ channel: "chrome", headless: true, args: ["--no-sandbox"] });
  const scenarios = [
    { name: "completed", login: "legacy-order-editor", credential: "editor-secret", from: "2026-09-18T08:30:00Z", row: "0" },
    { name: "authentication-denied", login: "legacy-order-editor", credential: "wrong-secret", from: "2026-09-18T08:30:00Z", row: "0" },
    { name: "authorization-denied", login: "legacy-order-viewer", credential: "viewer-secret", from: "2026-09-18T08:30:00Z", row: "0" },
    { name: "validation-not-satisfied", login: "legacy-order-editor", credential: "editor-secret", from: "2026-09-18T08:00:00Z", row: "1" },
    { name: "execution-error", login: "legacy-order-editor", credential: "editor-secret", from: "2026-09-18T07:00:00Z", row: "2" },
  ];
  for (const size of [
    { name: "desktop", width: 1280, height: 800 },
    { name: "mobile", width: 375, height: 812 },
  ]) {
    const context = await browser.newContext({ viewport: { width: size.width, height: size.height } });
    const page = await context.newPage();
    page.on("pageerror", err => errors.push(size.name + ": " + err.message));
    await page.goto(origin + "/", { waitUntil: "networkidle" });
    const title = await page.title();
    assert.match(title, /TEI Custom Submit UI/);
    assert.equal(await page.locator("#json-command").count(), 1);
    assert.equal(await page.locator("#command-feedback").count(), 1);
    assert.equal(await page.locator('input[name="credential"]').getAttribute("type"), "password");
    assert.equal(await page.locator("#command-feedback").getAttribute("aria-live"), "polite");
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true,
      "viewport must not have horizontal overflow");
    const blank = join(screenshots, size.name + "-blank.png");
    await page.screenshot({ path: blank, fullPage: true, animations: "disabled" });
    records.push({ screenshot: size.name + "-blank.png", sha256: await digest(blank) });

    for (const spec of scenarios) {
      const sent = [];
      const observe = request => {
        if (request.method() === "POST" && new URL(request.url()).pathname === "/scenario/submit") {
          sent.push({ url: request.url(), type: request.headers()["content-type"] });
        }
      };
      page.on("request", observe);
      await page.locator('input[name="loginId"]').fill(spec.login);
      await page.locator('input[name="credential"]').fill(spec.credential);
      await page.locator('input[name="from"]').fill(spec.from);
      await page.locator('input[name="row"]').fill(spec.row);
      await page.locator('#json-command button[type="submit"]').click();
      await page.locator("#command-feedback").waitFor({ state: "visible" });
      await page.waitForFunction(stage => document.querySelector("#command-feedback")?.dataset.state === stage,
        spec.name, { timeout: 10000 });
      assert.equal(sent.length, 1, "one user click must submit exactly one command");
      assert.equal(sent[0].url, origin + "/scenario/submit");
      assert.match(sent[0].type, /^application\/json/);
      assert.equal(await page.locator('input[name="credential"]').inputValue(), "",
        "credential must be cleared even on failed outcomes");
      const display = await page.locator("#command-feedback").innerText();
      assert.ok(display.length > 0);
      assert.ok(!display.includes(spec.credential), "credential reflected in UI");
      assert.ok(!display.includes("synthetic submit adapter unavailable"),
        "internal execution detail reflected in UI");
      assert.equal(await page.locator('#json-command button[type="submit"]').isEnabled(), true);
      page.off("request", observe);
      records.push({ viewport: size.name, stage: spec.name, visualState: "rendered" });
      if (spec.name === "completed") {
        const image = join(screenshots, size.name + "-completed.png");
        await page.screenshot({ path: image, fullPage: true, animations: "disabled" });
        records.push({ screenshot: size.name + "-completed.png", sha256: await digest(image) });
      }
    }
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth), true,
      "result display must not overflow horizontally");
    await context.close();
  }

  // Actual browser remains alive when API is unreachable. A POST must not
  // render an invented completion or automatically replay a command.
  await stop(go);
  go = null;
  const ctx = await browser.newContext({ viewport: { width: 1280, height: 800 } });
  const page = await ctx.newPage();
  page.on("pageerror", err => errors.push("disconnected: " + err.message));
  await page.goto(origin + "/", { waitUntil: "networkidle" });
  await page.locator('input[name="loginId"]').fill("legacy-order-editor");
  await page.locator('input[name="credential"]').fill("editor-secret");
  await page.locator('input[name="from"]').fill("2026-09-18T08:30:00Z");
  await page.locator('input[name="row"]').fill("0");
  await page.locator('#json-command button[type="submit"]').click();
  await page.waitForFunction(() => document.querySelector("#command-feedback")?.dataset.state === "unknown");
  assert.equal(await page.locator('input[name="credential"]').inputValue(), "");
  assert.equal(errors.length, 0, "unexpected uncaught Browser errors");
  await ctx.close();

  const receipt = {
    kind: "diagnostic-real-chrome-browser-json-api-replay",
    sourceRevision: revision,
    applicationId: "order_entry",
    handoverBaselineId: measurement.handoverBaselineId,
    baselineRepositoryRevision: measurement.baselineRepositoryRevision,
    browserEngine: "Google Chrome (GitHub runner)",
    viewportEvidence: ["desktop 1280x800", "mobile 375x812"],
    syntheticScenarioCount: scenarios.length * 2,
    browserDOMAndRealHTTPVerified: true,
    browserPasswordFieldCleared: true,
    browserUnknownOnApiFailure: true,
    visualScreenshotsCaptured: true,
    screenshotDigests: records.filter(x => x.screenshot),
    visualFidelityHumanAccepted: false,
    productionAuthorizationVerified: false,
    fullE1BrowserAPIPackagingVerified: false,
    artifactAuthority: false,
  };
  await writeFile(join(screenshots, "real-browser-receipt.json"), JSON.stringify(receipt, null, 2) + "\n");
  console.log(JSON.stringify(receipt));
} finally {
  if (browser) await browser.close();
  await stop(host);
  await stop(go);
}
