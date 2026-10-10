import test from "node:test";
import assert from "node:assert/strict";
import {createHash} from "node:crypto";
import {mkdtempSync,readFileSync,writeFileSync,rmSync} from "node:fs";
import {tmpdir} from "node:os";
import {dirname,join} from "node:path";
import {fileURLToPath} from "node:url";
import {spawnSync} from "node:child_process";

const root=dirname(fileURLToPath(import.meta.url));
const cli=join(root,"dfd-edit-cli.mjs");
const read=name=>readFileSync(join(root,name));
const sha=bytes=>"sha256:"+createHash("sha256").update(bytes).digest("hex");
const candidate=(digest,patch={})=>({
  schema:"tei.reference.business-dfd-edit/v0",id:"candidate-role",
  origin:"studio",intent:"assign approver to submit step",
  procedureId:"order-entry.approval-and-shipment-prep",baseReviewSHA256:digest,
  operation:{kind:"assign-business-role",step:"APP-40",role:"approver"},
  ...patch
});

function scratch(fn) {
  const dir=mkdtempSync(join(tmpdir(),"tei-dfd-edit-"));
  try{
    const p=join(dir,"procedure.json"),r=join(dir,"review.json"),e=join(dir,"edit.json");
    const procedure=read("dfd-order-approval/order-approval-procedure.json");
    const review=read("dfd-order-approval/review-proposal.json");
    writeFileSync(p,procedure);writeFileSync(r,review);
    return fn({p,r,e,dir,procedure,review,reviewDigest:sha(review)});
  }finally{rmSync(dir,{recursive:true,force:true})}
}
function run(p,r,e) {
  return spawnSync(process.execPath,[cli,"-procedure",p,"-review",r,"-edit",e],
    {encoding:"utf8"});
}

test("CLI verifies review bytes and leaves all Source files untouched",()=>scratch(
  ({p,r,e,procedure,review,reviewDigest})=>{
    const edit=candidate(reviewDigest),input=JSON.stringify(edit);
    writeFileSync(e,input);
    const result=run(p,r,e);
    assert.equal(result.status,0,result.stderr);
    const projection=JSON.parse(result.stdout);
    assert.equal(projection.inputSnapshotsVerified,true);
    assert.equal(projection.inputSnapshots.procedure,sha(procedure));
    assert.equal(projection.inputSnapshots.review,reviewDigest);
    assert.equal(projection.inputSnapshots.edit,sha(Buffer.from(input)));
    assert.equal(projection.readyForAcceptance,false);
    assert.deepEqual(projection.sourceWrites,[]);
    assert.equal(projection.proposedReview.responsibilities.length,7);
    assert.equal(readFileSync(p).toString(),procedure.toString());
    assert.equal(readFileSync(r).toString(),review.toString());
    assert.equal(readFileSync(e).toString(),input);
  }
));

test("CLI refuses stale fingerprint before returning a usable candidate",()=>scratch(
  ({p,r,e})=>{
    writeFileSync(e,JSON.stringify(candidate("sha256:"+"0".repeat(64))));
    const result=run(p,r,e);
    assert.notEqual(result.status,0);
    assert.match(result.stderr,/INVALID_EDIT_OR_STALE_REVIEW/);
    assert.equal(result.stdout,"");
  }
));

test("CLI refuses direct external-party to Store write even with correct fingerprint",()=>scratch(
  ({p,r,e,reviewDigest})=>{
    writeFileSync(e,JSON.stringify(candidate(reviewDigest,{
      operation:{kind:"add-information-flow",id:"bad",
        from:{kind:"party",id:"supplier"},to:{kind:"store",id:"orders"},
        payload:"unapproved write"}
    })));
    const result=run(p,r,e);
    assert.notEqual(result.status,0);
    assert.match(result.stderr,/INFORMATION_PROCESS_BOUNDARY_REQUIRED/);
    assert.equal(result.stdout,"");
  }
));
