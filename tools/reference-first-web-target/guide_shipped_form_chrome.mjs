// CI-only browser adapter for the first-change guide's shipped HTML form.
// Use the already pinned playwright-core from a runner-temporary directory.
// This does not replace the real Web process, its PostgreSQL read, or the
// independent Expected checked by the calling Go test.
import { access } from "node:fs/promises";
import path from "node:path";
import { pathToFileURL } from "node:url";

const [address, row, chromePath, playwrightRoot] = process.argv.slice(2);
if (!/^127\.0\.0\.1:[0-9]{1,5}$/.test(address ?? "") || !/^[01]$/.test(row ?? "") ||
    !path.isAbsolute(chromePath ?? "") || !path.isAbsolute(playwrightRoot ?? "")) {
  throw new Error("expected loopback address, selected row, absolute Chrome path and playwright root");
}

const playwrightEntrypoint = path.join(playwrightRoot, "node_modules", "playwright-core", "index.mjs");
await access(playwrightEntrypoint); // missing tooling must fail, not fall back to bootstrap HTML
const { chromium } = await import(pathToFileURL(playwrightEntrypoint).href);
const base = `http://${address}`;
const browser = await chromium.launch({
  executablePath: chromePath,
  headless: true,
  args: ["--no-sandbox", "--disable-dev-shm-usage"],
});

try {
  const page = await browser.newPage();
  const crossOrigin = new Set();
  const postRequests = [];
  const pageErrors = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));
  page.on("request", (request) => {
    if (new URL(request.url()).origin !== base) crossOrigin.add(new URL(request.url()).origin);
    if (request.method() === "POST") postRequests.push(request);
  });
  // Record any unexpected origin, but also block it before sending. This
  // page carries a credential fixture; detection after the request is too late.
  await page.route("**/*", (route) => {
    const origin = new URL(route.request().url()).origin;
    if (origin !== base) {
      crossOrigin.add(origin);
      return route.abort("blockedbyclient");
    }
    return route.continue();
  });

  const response = await page.goto(`${base}/custom-submit`, { waitUntil: "domcontentloaded" });
  if (!response || response.status() !== 200) throw new Error("the shipped GET form was not served");
  const form = page.locator('form[method="post"][action="/custom-submit"]');
  if (await form.count() !== 1) throw new Error("the shipped form is missing or ambiguous");

  // The test must exercise the actual user-visible fields, not a helper page
  // containing hidden inputs or a JavaScript-generated POST.
  for (const name of ["loginId", "credential", "from", "row"]) {
    const field = form.locator(`input[name="${name}"]`);
    if (await field.count() !== 1 || await field.getAttribute("required") === null) {
      throw new Error("the shipped visible input is missing, repeated or not required");
    }
  }
  if (await form.locator('input[name="credential"]').getAttribute("type") !== "password" ||
      await page.locator("#read-result").count() !== 0) {
    throw new Error("the unsubmitted GET form already has a result or lacks a password field");
  }
  // The published first-change guide starts browsing on September 1. Check
  // the value a real browser displays before the test explicitly fills it.
  if (await form.locator('input[name="from"]').inputValue() !== "2026-09-01T00:00:00Z") {
    throw new Error("the shipped form's initial Browse from differs from the first-change guide");
  }
  await form.locator('input[name="loginId"]').fill("legacy-order-editor");
  await form.locator('input[name="credential"]').fill("editor-secret");
  await form.locator('input[name="from"]').fill("2026-09-01T00:00:00Z");
  await form.locator('input[name="row"]').fill(row);
  const submit = form.locator('button[type="submit"]');
  if (await submit.count() !== 1) throw new Error("the shipped Submit button is missing");
  const [posted] = await Promise.all([
    page.waitForNavigation({ waitUntil: "domcontentloaded" }),
    submit.click(),
  ]);
  if (!posted || posted.status() !== 200) throw new Error("the submitted shipped form was rejected");
  await page.locator("#read-result").waitFor({ state: "attached" });
  if (postRequests.length !== 1 || postRequests[0].url() !== `${base}/custom-submit` ||
      crossOrigin.size !== 0 || pageErrors.length !== 0) {
    throw new Error("the shipped form made an unexpected request or failed in the browser");
  }
  const fields = new URLSearchParams(postRequests[0].postData() ?? "");
  const names = [...fields.keys()].sort();
  if (JSON.stringify(names) !== JSON.stringify(["credential", "from", "loginId", "row"]) ||
      fields.get("loginId") !== "legacy-order-editor" ||
      fields.get("credential") !== "editor-secret" ||
      fields.get("from") !== "2026-09-01T00:00:00Z" ||
      fields.get("row") !== row) {
    throw new Error("the shipped form did not POST the four selected visible fields");
  }
  const dom = await page.content();
  if (!dom.includes('id="order-lines"') || !dom.includes('id="read-identity"')) {
    throw new Error("the shipped form did not render the real order-line read");
  }
  process.stdout.write(dom);
} finally {
  await browser.close();
}
