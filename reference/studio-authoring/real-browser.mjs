// #1124: Real Chrome interaction with the generated offline Studio form.
// Automated browser input is NOT human acceptance or source-write authority.
import assert from "node:assert/strict";
import { execFileSync, spawnSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

if (process.argv.length !== 4)
  throw new Error("usage: node real-browser.mjs PLAYWRIGHT_ROOT OUTPUT_DIR");
const root = fileURLToPath(new URL("../../", import.meta.url));
const installRoot = resolve(process.argv[2]), outputDir = resolve(process.argv[3]);
const { chromium } = await import(pathToFileURL(
  join(installRoot, "node_modules/playwright-core/index.mjs")).href);
const workspace = join(root, "reference/workspace");
const cli = join(root, "reference/studio-authoring/cli.mjs");
const temporary = mkdtempSync(join(tmpdir(), "tei-studio-chrome-"));
mkdirSync(outputDir, { recursive: true });

function run(bin, args) {
  return execFileSync(bin, args, {
    cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    timeout: 120_000
  });
}
const revision = () => JSON.parse(run("go",
  ["run", "./tools/reference-shared-acceptance", "revision",
    "-workspace", workspace])).modelRevision;
const baseline = revision();
const scenarios = [
  {
    name: "er", focus: "data:sales.order_line",
    fill: { symbol: "order", "target-data": "sales.order", cardinality: "one" },
    operation: "add-relation"
  },
  {
    name: "table", focus: "attribute:sales.order_line.unit_price",
    fill: { "validation-id": "sales.order_line.unit_price_positive" },
    operation: "add-validation"
  }
];
let browser;
const receipts = [];
try {
  // Ubuntu GitHub runner has Chrome. Playwright is installed outside checkout.
  browser = await chromium.launch({
    channel: "chrome", headless: true, args: ["--no-sandbox"]
  });
  for (const scenario of scenarios) {
    const html = run(process.execPath, [
      cli, "edit-form", "-workspace", workspace, "-focus", scenario.focus
    ]);
    assert.match(html, /Content-Security-Policy/);
    const htmlPath = join(temporary, scenario.name + ".html");
    writeFileSync(htmlPath, html, { mode: 0o600 });
    const context = await browser.newContext({
      viewport: { width: 1280, height: 900 }
    });
    const page = await context.newPage();
    const scriptErrors = [], networkRequests = [];
    page.on("pageerror", e => scriptErrors.push(e.message));
    page.on("request", request => {
      if (/^https?:/i.test(request.url())) networkRequests.push(request.url());
    });
    await page.goto(pathToFileURL(htmlPath).href, { waitUntil: "load" });
    assert.equal(await page.title(), "TEI Studio 編集候補の作成");
    assert.equal(await page.locator("#subject").inputValue(), scenario.focus);
    assert.equal(await page.locator("#model-revision").inputValue(), baseline);
    assert.equal(await page.locator("#subject").getAttribute("readonly"), "");
    assert.equal(await page.locator("#model-revision").getAttribute("readonly"), "");
    await page.locator("#candidate-id").fill("chrome-1124-" + scenario.name);
    await page.locator("#intent").fill("Chrome上の変更候補を共通審査に渡す");
    for (const [id, value] of Object.entries(scenario.fill)) {
      const field = page.locator("#" + id);
      if (id === "target-data" || id === "cardinality")
        await field.selectOption(value);
      else await field.fill(value);
    }
    await page.locator('button[type="submit"]').click();
    const output = await page.locator("#proposal-json").inputValue();
    const edit = JSON.parse(output);
    assert.equal(edit.schema, "tei.reference.studio-edit/v0");
    assert.equal(edit.origin, "studio");
    assert.equal(edit.expectedModelRevision, baseline);
    assert.equal(edit.subject, scenario.focus);
    assert.equal(edit.operation, scenario.operation);
    assert.equal(edit.id, "chrome-1124-" + scenario.name);
    assert.match(await page.locator("#notice").innerText(), /未審査/);
    assert.deepEqual(scriptErrors, [], "real Chrome did not reject the inline CSP-pinned script");
    assert.deepEqual(networkRequests, [], "offline form has no external HTTP requests");

    const editPath = join(temporary, scenario.name + ".json");
    writeFileSync(editPath, output, { mode: 0o600 });
    const proposed = JSON.parse(run(process.execPath, [
      cli, "propose", "-workspace", workspace, "-edit-file", editPath
    ]));
    assert.equal(proposed.baseModelRevision, baseline);
    assert.equal(proposed.schema, "tei.reference.change-candidate/v0");
    const reviewed = JSON.parse(run(process.execPath, [
      cli, "review-edit", "-workspace", workspace, "-edit-file", editPath
    ]));
    assert.equal(reviewed.schema, "tei.reference.studio-review-edit/v0");
    assert.equal(reviewed.modelRevision, baseline);
    assert.equal(reviewed.sourceRevisionAfterReview, baseline);
    assert.equal(reviewed.sourceWriteByThisCommand, false);
    assert.equal(reviewed.explicitAcceptance, "not-performed");
    assert.equal(reviewed.candidate.id, edit.id);
    assert.equal(revision(), baseline);
    await page.screenshot({
      path: join(outputDir, scenario.name + ".png"), fullPage: true
    });
    // Browser controls are presentation only. A caller can tamper with
    // <select> options in DevTools; the Go-backed source bridge must refuse
    // an unknown reference even when the real Chrome submit handler emits it.
    await page.locator("#candidate-id").fill("chrome-1124-tampered-" + scenario.name);
    const expectedError = scenario.name === "er"
      ? "INVALID_RELATION_PROPOSAL"
      : "APPLICATION_DOES_NOT_DECLARE_SEMANTIC_USE";
    if (scenario.name === "er") {
      await page.locator("#target-data").evaluate(el => {
        const option = document.createElement("option");
        option.value = "sales.nonexistent";
        option.textContent = "sales.nonexistent";
        el.add(option);
        el.value = option.value;
      });
    } else {
      await page.locator("#use-application").check();
      await page.locator("#application").evaluate(el => {
        const option = document.createElement("option");
        option.value = "nonexistent_application";
        option.textContent = "nonexistent_application";
        el.add(option);
        el.value = option.value;
      });
      await page.locator("#placement").fill("applications/order-entry/application.tei");
    }
    await page.locator('button[type="submit"]').click();
    const tamperedOutput = JSON.parse(await page.locator("#proposal-json").inputValue());
    assert.equal(tamperedOutput.expectedModelRevision, baseline);
    const tamperedPath = join(temporary, scenario.name + "-tampered.json");
    writeFileSync(tamperedPath, JSON.stringify(tamperedOutput) + "\\n", { mode: 0o600 });
    const rejected = spawnSync(process.execPath, [
      cli, "propose", "-workspace", workspace, "-edit-file", tamperedPath
    ], {
      cwd: root, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
      timeout: 120_000
    });
    assert.equal(rejected.status, 1, "tampered Chrome edit must fail before Review");
    assert.match(rejected.stderr, new RegExp(expectedError));
    assert.equal(rejected.stdout, "");
    assert.equal(revision(), baseline, "rejected browser input must not change Source revision");
    assert.deepEqual(scriptErrors, [], "tampered form must not cause an unhandled browser error");
    assert.deepEqual(networkRequests, [], "tampered form must not trigger network I/O");

    receipts.push({
      name: scenario.name, focus: scenario.focus,
      editSchema: edit.schema, candidateSchema: proposed.schema,
      reviewSchema: reviewed.schema,
      revisionBound: true, realChromeDOMEvent: true,
      cspScriptExecuted: true, offlineNetworkRequests: networkRequests.length,
      invalidBrowserReferenceRefused: true,
      accepted: false, sourceChanged: false
    });
    await context.close();
  }
  const receipt = {
    schema: "tei.reference.studio-real-browser-receipt/v0",
    head: process.env.GITHUB_SHA || null,
    browser: "real-chrome-headless",
    cases: receipts,
    revision: baseline,
    sourceRevisionAfter: revision(),
    authority: {
      humanAcceptance: "not-observed", productionBrowser: "not-observed",
      acceptedBusiness: "not-established", actualProvider: "not-observed"
    },
    qualification: "automated Chrome form submit to actual Go preview/review; no source accept or human review"
  };
  assert.equal(receipt.sourceRevisionAfter, baseline);
  writeFileSync(join(outputDir, "real-browser-receipt.json"),
    JSON.stringify(receipt, null, 2) + "\n", { mode: 0o600 });
  process.stdout.write(JSON.stringify(receipt, null, 2) + "\n");
} finally {
  if (browser) await browser.close();
  rmSync(temporary, { recursive: true, force: true });
}
