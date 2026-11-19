// Real shipped GET form -> user-like input -> button POST. This is a CI
// browser operation, not proof of a human completing the first-change guide.
import assert from "node:assert/strict";
import { join, resolve } from "node:path";
import { pathToFileURL } from "node:url";

if (process.argv.length !== 4) {
  throw Error("usage: node first_change_shipped_form_browser.mjs FORM_URL PLAYWRIGHT_INSTALL_ROOT");
}
const url = new URL(process.argv[2]);
assert.equal(url.protocol, "http:");
assert.equal(url.hostname, "127.0.0.1");
assert.equal(url.pathname, "/custom-submit");
const browserBinary = process.env.TEI_VIEW_BROWSER_BINARY;
assert.ok(browserBinary, "TEI_VIEW_BROWSER_BINARY is required for the real-form test");
const modulePath = join(resolve(process.argv[3]), "node_modules/playwright-core/index.mjs");
const { chromium } = await import(pathToFileURL(modulePath).href);
const browser = await chromium.launch({
  executablePath: browserBinary, headless: true, args: ["--no-sandbox"],
});
const cases = [];
try {
  for (const row of [0, 1]) {
    // A new page per order proves that the second answer did not come from a
    // previous result. The same running Web process and DB are used for both.
    const context = await browser.newContext();
    try {
      const page = await context.newPage();
      const requests = [];
      const pageErrors = [];
      page.on("pageerror", error => pageErrors.push(error.message));
      page.on("request", request => {
        if (request.method() === "POST") requests.push(request);
      });
      const initial = await page.goto(url.href, { waitUntil: "load" });
      assert.equal(initial?.status(), 200, "the product GET form must load");
      assert.equal(await page.locator('form[action="/custom-submit"][method="post"]').count(), 1);
      for (const name of ["loginId", "credential", "from", "row"]) {
        assert.equal(await page.locator('form input[name="' + name + '"]').count(), 1);
        assert.equal(await page.locator('form input[name="' + name + '"]').getAttribute("required"), "");
      }
      assert.equal(await page.locator('input[name="credential"]').getAttribute("type"), "password");
      assert.equal(await page.locator("#read-result").count(), 0,
        "an unsubmitted GET must not already contain a business read");
      await page.locator('input[name="loginId"]').fill("legacy-order-editor");
      await page.locator('input[name="credential"]').fill("editor-secret");
      await page.locator('input[name="from"]').fill("2026-09-01T00:00:00Z");
      await page.locator('input[name="row"]').fill(String(row));
      const [submitted] = await Promise.all([
        page.waitForResponse(response => response.url() === url.href &&
          response.request().method() === "POST", { timeout: 30000 }),
        page.locator('form[action="/custom-submit"] button[type="submit"]').click(),
      ]);
      await page.waitForLoadState("load");
      assert.equal(submitted.status(), 200, "product form POST must succeed");
      assert.equal(requests.length, 1, "one button click must issue exactly one POST");
      assert.equal(requests[0].url(), url.href, "the actual form must post to the product endpoint");
      const submittedFields = new URLSearchParams(requests[0].postData() ?? "");
      assert.deepEqual([...submittedFields.keys()].sort(), ["credential", "from", "loginId", "row"]);
      assert.equal(submittedFields.get("loginId"), "legacy-order-editor");
      assert.equal(submittedFields.get("credential"), "editor-secret");
      assert.equal(submittedFields.get("from"), "2026-09-01T00:00:00Z");
      assert.equal(submittedFields.get("row"), String(row));
      assert.deepEqual(pageErrors, [], "browser script errors during product form interaction");
      assert.equal(await page.locator("#read-result").count(), 1);
      assert.equal(await page.locator("#order-lines").count(), 1);
      assert.equal(await page.locator("#relation-answers").count(), 1);
      const html = await page.content();
      assert.ok(!html.includes("editor-secret"), "FORM_CREDENTIAL_LEAK");
      cases.push({ row, getStatus: initial.status(), postStatus: submitted.status(),
        posts: requests.length, html });
    } finally {
      await context.close();
    }
  }
} finally {
  await browser.close();
}
// stdout is only a machine-readable result for the Go independent Expected.
// No connection string or submitted credential is serialized.
process.stdout.write(JSON.stringify({ transport: "product-get-form-playwright", cases }));
