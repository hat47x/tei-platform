// Actual Chrome + actual Go processes: independent project advisory and
// read-only accepted order_entry reference. Not a production acceptance test.
import assert from "node:assert/strict";
import {spawn, execFileSync} from "node:child_process";
import {createHash} from "node:crypto";
import {readFile, mkdir, writeFile} from "node:fs/promises";
import {createServer} from "node:net";
import {join, resolve} from "node:path";
import {pathToFileURL} from "node:url";

if (process.argv.length !== 6) {
  throw Error("usage: node real_browser_replay.mjs ORDER_API_BINARY ADVICE_HOST_BINARY PLAYWRIGHT_ROOT SCREENSHOTS");
}
const [apiBinary, hostBinary, playwrightRoot, screenshots] = process.argv.slice(2).map(arg => resolve(arg));
const {chromium} = await import(pathToFileURL(join(playwrightRoot, "node_modules/playwright-core/index.mjs")).href);
const repo = resolve(".");
const revision = execFileSync("git", ["rev-parse", "HEAD"], {encoding:"utf8"}).trim();
const origin = "http://127.0.0.1:8787";
const digest = bytes => createHash("sha256").update(bytes).digest("hex");

const reservePort = async () => {
  const sock = createServer();
  await new Promise((resolve, reject) => sock.once("error", reject).listen(0, "127.0.0.1", resolve));
  const port = sock.address().port;
  await new Promise((resolve, reject) => sock.close(err => err ? reject(err) : resolve()));
  return port;
};
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
const stop = async proc => {
  if (!proc || proc.exitCode !== null || proc.signalCode !== null) return;
  proc.kill("SIGTERM");
  await Promise.race([
    new Promise(resolve => proc.once("exit", resolve)),
    sleep(3000).then(() => { if (proc.exitCode === null) proc.kill("SIGKILL"); }),
  ]);
};
const waitReady = async (url, proc) => {
  for (let attempt=0; attempt<100; attempt++) {
    if (proc.exitCode !== null || proc.signalCode !== null) {
      throw Error("Go reference process exited before ready");
    }
    try {
      const res=await fetch(url,{signal:AbortSignal.timeout(1500)});
      if(res.ok)return;
    }catch{ /* retry only a harmless read, never a Command */ }
    await sleep(100);
  }
  throw Error("Go reference process did not become ready");
};
const statusSnippet = async (page,index,text) => {
  await page.waitForFunction(({index,text}) => {
    const host = document.querySelector("tei-project-customer-advice");
    const node = host?.shadowRoot?.querySelectorAll('p[role="status"]')[index];
    return !!node && node.textContent.includes(text);
  },{index,text},{timeout:10000});
};

let api, host, browser;
const events = [];
const errors = [];
try {
  await mkdir(screenshots, {recursive:true});
  const port = await reservePort();
  const apiOrigin = "http://127.0.0.1:"+port;
  api=spawn(apiBinary,[
    "-workspace",join(repo,"reference/workspace"),
    "-legacy-users",join(repo,"reference/legacy-auth/users.json"),
    "-legacy-authorization",join(repo,"reference/legacy-auth/authorization.json"),
    "-application","order_entry",
    "-baseline-id","B",
    "-baseline-revision","e5231839a5dcc62cce1181a2731fcb7c70d16736",
    "-target-revision",revision,
    "-listen","127.0.0.1:"+port,
  ],{stdio:"ignore"});
  await waitReady(apiOrigin+"/manifest.json",api);
  const sourceReadResponse=await fetch(apiOrigin+"/scenario/read");
  assert.equal(sourceReadResponse.status,200);
  const sourceRead=await sourceReadResponse.json();
  assert.equal(sourceRead.result.applicationId,"order_entry");

  host=spawn(hostBinary,["-order-entry-api",apiOrigin],{stdio:"ignore"});
  await waitReady(origin+"/",host);
  const htmlResponse=await fetch(origin+"/");
  assert.match(htmlResponse.headers.get("content-security-policy")??"",/script-src 'self'/);
  assert.match(htmlResponse.headers.get("content-security-policy")??"",/connect-src 'self'/);
  const bundleResponse=await fetch(origin+"/lit-core.min.js");
  assert.equal(bundleResponse.status,200);
  const bundle=Buffer.from(await bundleResponse.arrayBuffer());
  const localBundle=await readFile("tools/reference-project-advice-lit/web/lit-core.min.js");
  assert.equal(digest(bundle),digest(localBundle),"Browser-served Lit differs from pinned local bundle");
  const licenseResponse=await fetch(origin+"/lit-license.txt");
  assert.equal(licenseResponse.status,200);
  assert.match(await licenseResponse.text(),/BSD 3-Clause License/);

  browser=await chromium.launch({channel:"chrome",headless:true,args:["--no-sandbox"]});
  for(const viewport of [{name:"desktop",width:1280,height:800},
                          {name:"mobile",width:375,height:812}]){
    const context=await browser.newContext({viewport:{width:viewport.width,height:viewport.height}});
    const page=await context.newPage();
    const requests=[];
    page.on("pageerror",e=>errors.push(viewport.name+": "+e.message));
    page.on("request",req=>requests.push({url:req.url(),method:req.method()}));
    await page.goto(origin+"/",{waitUntil:"networkidle"});
    await page.waitForFunction(() =>
      customElements.get("tei-project-customer-advice") &&
      document.querySelector("tei-project-customer-advice")?.shadowRoot?.querySelector("form"),
      null,{timeout:10000});
    const ui=page.locator("tei-project-customer-advice");
    assert.equal(await ui.locator('input[name="customerCode"]').count(),1);
    assert.equal(await ui.locator('textarea[name="note"]').count(),1);
    assert.equal(await ui.locator('p[role="status"]').count(),2);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth),true);

    await ui.locator('input[name="customerCode"]').fill("VIP-123");
    await ui.locator('textarea[name="note"]').fill("");
    // Keyboard activation exercises semantic HTML rather than a mouse-only UI.
    await ui.locator('button[type="submit"]').focus();
    await page.keyboard.press("Enter");
    await statusSnippet(page,0,"メモの記入を推奨");
    events.push({viewport:viewport.name,result:"review_suggested"});

    await ui.locator('textarea[name="note"]').fill("確認済み");
    await ui.locator('button[type="submit"]').click();
    await statusSnippet(page,0,"追加の提案はありません");
    events.push({viewport:viewport.name,result:"no_suggestion"});

    const readRequestsBefore=requests.filter(x=>new URL(x.url).pathname==="/project/order-entry-context").length;
    await ui.getByRole("button",{name:"受注参照を取得"}).click();
    await statusSnippet(page,1,"受注参照: order_entry");
    const referenceStatus=await ui.locator('p[role="status"]').nth(1).innerText();
    assert.ok(referenceStatus.includes(sourceRead.result.scenarioId));
    assert.ok(referenceStatus.includes("選択行 "+sourceRead.result.selectedRowIndex));
    assert.equal(requests.filter(x=>new URL(x.url).pathname==="/project/order-entry-context").length,
      readRequestsBefore+1);
    assert.ok(requests.every(x=>new URL(x.url).origin===origin),"unexpected external Browser request");
    assert.equal(requests.filter(x=>new URL(x.url).pathname==="/scenario/submit").length,0);
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth<=innerWidth),true);

    const png=join(screenshots,viewport.name+"-lit-order-entry-read.png");
    await page.screenshot({path:png,fullPage:true,animations:"disabled"});
    events.push({viewport:viewport.name,result:"order_entry_read",
      screenshot:viewport.name+"-lit-order-entry-read.png",sha256:digest(await readFile(png))});

    // Advisory transport uncertainty must not turn into a success/denial.
    await page.route("**/project/advice",route=>route.abort("failed"));
    await ui.locator('textarea[name="note"]').fill("");
    await ui.locator('button[type="submit"]').click();
    await statusSnippet(page,0,"結果を確認できません");
    await page.unroute("**/project/advice");
    events.push({viewport:viewport.name,result:"advisory_network_unknown"});

    // Killing a read-only upstream must not fabricate a fresh read result.
    if(viewport.name==="mobile"){
      await stop(api);
      api=null;
      await ui.getByRole("button",{name:"受注参照を取得"}).click();
      await statusSnippet(page,1,"接続が必要");
      events.push({viewport:viewport.name,result:"order_entry_read_unavailable"});
    }
    await context.close();
  }
  assert.deepEqual(errors,[]);
  const receipt={
    kind:"tei-reference-project-advice-lit-real-chrome/v0",
    sourceRevision:revision,
    browserEngine:"Google Chrome (GitHub runner)",
    testedViewports:["1280x800","375x812"],
    actualOrderEntryReadOnlyGoAPI:true,
    pinnedLocalLitBundleLoaded:true,
    projectAdvisoryGoHTTPVerified:true,
    browserKeyboardActivationVerified:true,
    browserShadowDOMAndStatusVerified:true,
    noBusinessCommandInvoked:true,
    browserNetworkFailureIsNotBusinessSuccess:true,
    screenshotDigests:events.filter(x=>x.screenshot),
    productionServiceContractAuthorityVerified:false,
    acceptedBusinessValidationExtended:false,
    visualFidelityHumanAccepted:false,
    events,
  };
  await writeFile(join(screenshots,"project-lit-browser-receipt.json"),JSON.stringify(receipt,null,2)+"\n");
  console.log(JSON.stringify(receipt));
}finally{
  if(browser)await browser.close();
  await stop(host);
  await stop(api);
}
