#!/usr/bin/env node
// Read-only entrypoint for #1125. It computes exact byte SHA-256 before
// constructing a proposal. It does not call Shared Acceptance or write files.
import {readFileSync} from "node:fs";
import {createHash} from "node:crypto";
import {proposeDFDReviewEdit} from "./dfd-edit-proposal.mjs";

function argumentsFrom(argv) {
  const valid=new Set(["-procedure","-review","-edit"]);
  const out={};
  for (let i=0;i<argv.length;i+=2) {
    const flag=argv[i],value=argv[i+1];
    if (!valid.has(flag) || out[flag] || !value || value.startsWith("-")) {
      throw new Error("USAGE: -procedure <procedure.json> -review <review.json> -edit <edit.json>");
    }
    out[flag]=value;
  }
  if (Object.keys(out).length!==3 ||
      !out["-procedure"] || !out["-review"] || !out["-edit"]) {
    throw new Error("THREE_EXPLICIT_INPUT_PATHS_REQUIRED");
  }
  return out;
}
const digest=bytes=>"sha256:"+createHash("sha256").update(bytes).digest("hex");
const load=path=>{
  const bytes=readFileSync(path);
  return {path,digest:digest(bytes),json:JSON.parse(bytes.toString("utf8"))};
};
function run() {
  const paths=argumentsFrom(process.argv.slice(2));
  const files={
    procedure:load(paths["-procedure"]),
    review:load(paths["-review"]),
    edit:load(paths["-edit"])
  };
  const output=proposeDFDReviewEdit(
    files.procedure.json,files.review.json,files.review.digest,files.edit.json);
  for(const file of Object.values(files)) {
    if (digest(readFileSync(file.path))!==file.digest) {
      throw new Error("DFD_INPUT_CHANGED_DURING_REVIEW: "+file.path);
    }
  }
  output.inputSnapshots={
    procedure:files.procedure.digest,
    review:files.review.digest,
    edit:files.edit.digest
  };
  output.inputSnapshotsVerified=true;
  // No accepted World/revision or Source Index was loaded. Matching bytes
  // prove only stability of this input triple during a read-only request.
  process.stdout.write(JSON.stringify(output,null,2)+"\n");
}
try{run()}catch(err){process.stderr.write(String(err.message)+"\n");process.exitCode=1}
