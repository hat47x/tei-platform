// One exact scratch Workspace: real Chrome edit -> existing Studio proposal ->
// Shared Acceptance Review -> explicit test-harness Accept -> actual CLI Inspect.
// An automated Accept is not a human decision or Owner meaning acceptance.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { execFileSync, spawnSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

if (process.argv.length !== 4)
  throw new Error("usage: node required-source-loop.mjs PLAYWRIGHT_ROOT OUTPUT_DIR");
const repo = resolve(dirname(fileURLToPath(import.meta.url)), "../..");
const playwright = resolve(process.argv[2]), output = resolve(process.argv[3]);
const { chromium } = await import(pathToFileURL(
  join(playwright, "node_modules/playwright-core/index.mjs")).href);
const studioCLI = join(repo, "reference/studio-authoring/cli.mjs");
const scratch = mkdtempSync(join(tmpdir(), "tei-studio-required-source-"));
const workspace = join(scratch, "workspace");
const sourceRel = "semantic/studio/inspection-response.tei.md";
const sourcePath = join(workspace, sourceRel);
const subject = "studio.inspection_response";
const attributeID = subject + ".snapshot_fingerprint";
const focus = "attribute:" + attributeID;
const sha = bytes => "sha256:" + createHash("sha256").update(bytes).digest("hex");

function invoke(binary, args) {
  return execFileSync(binary, args, {
    cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    timeout: 180000
  });
}
const tei = (command, ...args) => JSON.parse(invoke("go", [
  "run", "./cmd/tei", command, "-workspace", workspace, ...args
]));
const acceptance = (command, ...args) => JSON.parse(invoke("go", [
  "run", "./tools/reference-shared-acceptance", command,
  "-workspace", workspace, ...args
]));
const studio = (command, ...args) => JSON.parse(invoke(process.execPath, [
  studioCLI, command, "-workspace", workspace, ...args
]));
const modelRevision = () => acceptance("revision").modelRevision;
function requiredFromWorld() {
  const w = tei("world");
  const data = (w.semantics || []).flatMap(ns => ns.data || []).find(d => d.id === subject);
  assert.ok(data, "actual Workspace World lacks the target Data");
  const attr = (data.attributes || []).find(a => a.id === attributeID);
  assert.ok(attr, "actual Workspace World lacks the target Attribute");
  return attr.required === true;
}
function requiredFromInspect() {
  const result = tei("inspect", "-subject-kind", "semantic-data", "-subject-id", subject);
  assert.equal(result.ok, true, "actual built-in CLI Inspector must accept the Data");
  assert.equal(result.schemaVersion, "tei.cli.inspect/v0");
  const raw = result.result?.canonical;
  const canonical = typeof raw === "string" ? JSON.parse(raw) : raw;
  assert.equal(canonical?.id, subject);
  const attribute = canonical.attributes?.find(a => a.id === attributeID);
  assert.ok(attribute, "actual InspectSubject Canonical lacks the target attribute");
  return attribute.required === true;
}
function rejectedCLI(command, ...args) {
  const result = spawnSync("go", ["run", "./tools/reference-shared-acceptance",
    command, "-workspace", workspace, ...args], {
    cwd: repo, encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
    timeout: 180000
  });
  if (result.error) throw result.error;
  assert.notEqual(result.status, 0, command + " must refuse the unqualified change");
  return result;
}

let browser;
mkdirSync(output, {recursive:true});
try {
  mkdirSync(join(workspace, "semantic/studio"), {recursive:true});
  writeFileSync(join(workspace, "tei.workspace.yaml"),
    "workspace: studio_reader\nformatVersion: 1\nsources:\n" +
    "  - semantic\nlanguages:\n  - tei.authoring/v1\n");
  const original = "# Studio の定義\n\n" +
    "説明中の no は変えない。\n\n" +
    "<!-- tei: data InspectionResponse in studio -->\n" +
    "| attribute | type | required | display |\n" +
    "| --- | --- | --- | --- |\n" +
    "| schemaVersion | std.text | yes | Schema |\n" +
    "| snapshotFingerprint | std.text | no | Snapshot |\n" +
    "\n| required | unrelated |\n| --- | --- |\n| no | kept |\n";
  writeFileSync(sourcePath, original);
  const sourceDigestBefore = sha(readFileSync(sourcePath));
  const before = modelRevision();
  assert.equal(requiredFromWorld(), false);
  assert.equal(requiredFromInspect(), false);

  const html = invoke(process.execPath, [studioCLI, "edit-form",
    "-workspace", workspace, "-focus", focus, "-edit-mode", "required"]);
  assert.match(html, /Content-Security-Policy/);
  assert.match(html, /connect-src &#39;none&#39;/);
  assert.match(html, /現在の必須指定：no/);
  const htmlPath = join(scratch, "editor-before.html");
  writeFileSync(htmlPath, html);
  browser = await chromium.launch({channel:"chrome",headless:true,args:["--no-sandbox"]});
  const context = await browser.newContext({viewport:{width:1280,height:900}});
  try {
    const page = await context.newPage(), errors=[], requests=[];
    page.on("pageerror", e => errors.push(e.message));
    page.on("request", req => {
      if (/^https?:/i.test(req.url())) requests.push(req.url());
    });
    await page.goto(pathToFileURL(htmlPath).href, {waitUntil:"load"});
    assert.equal(await page.locator("#subject").inputValue(),focus);
    assert.equal(await page.locator("#model-revision").inputValue(),before);
    assert.equal(await page.locator("#subject").getAttribute("readonly"),"");
    assert.equal(await page.locator("#model-revision").getAttribute("readonly"),"");
    await page.locator("#candidate-id").fill("chrome-required-source");
    await page.locator("#intent").fill("Inspector の Snapshot 指紋を必須にする");
    await page.locator("#required-choice").selectOption("yes");
    await page.locator('button[type="submit"]').click();
    const edit = JSON.parse(await page.locator("#proposal-json").inputValue());
    assert.equal(edit.schema,"tei.reference.studio-edit/v0");
    assert.equal(edit.expectedModelRevision,before);
    assert.equal(edit.subject,focus);
    assert.equal(edit.required,true);
    assert.match(await page.locator("#notice").innerText(),/未審査/);
    assert.deepEqual(errors,[]);
    assert.deepEqual(requests,[]);
    await page.screenshot({path:join(output,"required-source-before.png"),fullPage:true});

    const editPath = join(scratch,"studio-edit.json");
    writeFileSync(editPath,JSON.stringify(edit,null,2)+"\n",{mode:0o600});
    const candidate = studio("propose","-edit-file",editPath);
    assert.equal(candidate.schema,"tei.reference.change-candidate/v0");
    assert.equal(candidate.baseModelRevision,before);
    assert.equal(candidate.origin?.kind,"studio");
    assert.deepEqual(candidate.delta,[{
      op:"set-attribute-required",id:attributeID,required:true
    }]);
    const candidatePath = join(scratch,"candidate.json");
    writeFileSync(candidatePath,JSON.stringify(candidate,null,2)+"\n",{mode:0o600});
    const reviewPacket = studio("review-edit","-edit-file",editPath);
    assert.equal(reviewPacket.schema,"tei.reference.studio-review-edit/v0");
    assert.equal(reviewPacket.modelRevision,before);
    assert.equal(reviewPacket.sourceRevisionAfterReview,before);
    assert.equal(reviewPacket.explicitAcceptance,"not-performed");
    assert.equal(reviewPacket.sourceWriteByThisCommand,false);
    assert.equal(reviewPacket.preview?.acceptable,true);
    const reviewFingerprint = reviewPacket.review?.reviewFingerprint;
    assert.match(reviewFingerprint,/^sha256:[0-9a-f]{64}$/);
    assert.equal(modelRevision(),before);
    assert.equal(sha(readFileSync(sourcePath)),sourceDigestBefore);
    assert.equal(requiredFromInspect(),false);

    rejectedCLI("accept","-candidate",candidatePath);
    rejectedCLI("accept","-candidate",candidatePath,
      "-expected-review-fingerprint","sha256:"+ "0".repeat(64));
    assert.equal(modelRevision(),before);
    assert.equal(sha(readFileSync(sourcePath)),sourceDigestBefore);
    const decision = acceptance("accept","-candidate",candidatePath,
      "-expected-review-fingerprint",reviewFingerprint);
    assert.equal(decision.decision,"accepted");
    assert.equal(decision.sourceChanged,true);
    const expected = original.replace(
      "| snapshotFingerprint | std.text | no | Snapshot |",
      "| snapshotFingerprint | std.text | yes | Snapshot |"
    );
    assert.equal(readFileSync(sourcePath,"utf8"),expected,
      "only the exact Data table required cell may change");
    const after = modelRevision();
    assert.notEqual(after,before);
    assert.equal(requiredFromWorld(),true);
    assert.equal(requiredFromInspect(),true);
    assert.notEqual(sha(readFileSync(sourcePath)),sourceDigestBefore);

    const htmlAfter = invoke(process.execPath, [studioCLI,"edit-form",
      "-workspace",workspace,"-focus",focus,"-edit-mode","required"]);
    assert.match(htmlAfter,/現在の必須指定：yes/);
    const htmlAfterPath=join(scratch,"editor-after.html");
    writeFileSync(htmlAfterPath,htmlAfter);
    await page.goto(pathToFileURL(htmlAfterPath).href,{waitUntil:"load"});
    assert.equal(await page.locator("#model-revision").inputValue(),after);
    assert.match(await page.locator("body").innerText(),/現在の必須指定：yes/);
    await page.screenshot({path:join(output,"required-source-after.png"),fullPage:true});
    assert.deepEqual(errors,[]);
    assert.deepEqual(requests,[]);
    rejectedCLI("accept","-candidate",candidatePath,
      "-expected-review-fingerprint",reviewFingerprint);
    const stale = spawnSync(process.execPath,[studioCLI,"review-edit",
      "-workspace",workspace,"-edit-file",editPath],{
        cwd:repo,encoding:"utf8",maxBuffer:16*1024*1024,timeout:180000
      });
    assert.notEqual(stale.status,0);
    assert.match(stale.stderr,/STUDIO_EDIT_STALE_MODEL_REVISION/);
    assert.equal(modelRevision(),after);

    const receipt={
      schema:"tei.reference.studio-required-source-loop/v0",
      head:process.env.GITHUB_SHA||null,
      source:sourceRel,subject,attributeID,
      browser:"real-chrome-headless",
      editSchema:edit.schema,candidateSchema:candidate.schema,
      reviewSchema:reviewPacket.schema,
      beforeRevision:before,afterRevision:after,
      beforeRequired:false,afterRequired:true,
      sourceBefore:sourceDigestBefore,sourceAfter:sha(readFileSync(sourcePath)),
      chromeSubmitted:true,reviewWithoutSourceWrite:true,
      unreviewedAcceptRejected:true,wrongFingerprintRejected:true,
      acceptedViaSharedCLI:true,exactSourceCellOnly:true,
      inspectorBeforeAndAfter:true,reopenedEditorShowsNewState:true,
      staleCandidateRejected:true,offlineNetworkRequests:requests.length,
      authority:{
        humanAcceptance:"not-observed",ownerMeaningAcceptance:"not-established",
        productWailsEditor:"not-observed",formalBuildIntent:"not-verified",
        testHarnessAccept:"automated"
      },
      qualification:"scratch-only Reference demonstration; not production authority or a Stage2 rebuild"
    };
    writeFileSync(join(output,"required-source-receipt.json"),JSON.stringify(receipt,null,2)+"\n");
    process.stdout.write(JSON.stringify(receipt,null,2)+"\n");
  } finally {
    await context.close();
  }
} finally {
  if(browser)await browser.close();
  rmSync(scratch,{recursive:true,force:true});
}
