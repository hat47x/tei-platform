// R0 read-only bridge: TEI CLI inspect/v0 -> operations inspect/v0 -> derived features.
// Pure adapter; it does not execute CLI, read Workspace, authenticate, or accept meaning.
import {projectInspectionV0, InspectionProjectionError} from './project-inspection-v0.mjs';

export class InspectBridgeError extends Error {
  constructor(code) { super(code); this.code=code; }
}
const isObject=x=>x!==null&&typeof x==='object'&&!Array.isArray(x);
const value=x=>typeof x==='string'&&x.length>0;
const subjectKey=s=>s.kind+'\u0000'+s.id;
const kinds=new Set(['application','application-pattern','semantic-relation','semantic-data']);

function checkSubject(subject) {
  if(!isObject(subject)||!kinds.has(subject.kind)||!value(subject.id))
    throw new InspectBridgeError('INVALID_SUBJECT_SELECTOR');
  if(subject.id.trim()!==subject.id)throw new InspectBridgeError('INVALID_SUBJECT_SELECTOR');
}

export function projectCliInspectReport(report,expectedSubject) {
  checkSubject(expectedSubject);
  if(!isObject(report)||report.schemaVersion!=='tei.cli.inspect/v0'||report.ok!==true||
      report.error!=null||!isObject(report.result))
    throw new InspectBridgeError('INVALID_OR_FAILED_CLI_INSPECTION');
  const inspected=report.result;
  if(!isObject(inspected.subject)||inspected.subject.kind!==expectedSubject.kind||
     inspected.subject.id!==expectedSubject.id)
    throw new InspectBridgeError('WRONG_SUBJECT_RETURNED');
  try { return projectInspectionV0(inspected); }
  catch(e) {
    if(e instanceof InspectionProjectionError)
      throw new InspectBridgeError('UNUSABLE_INSPECTION_RESULT');
    throw e;
  }
}

// readOne is an injected async read of an already-authorized exact Subject.
// It MUST return parsed tei.cli.inspect/v0; we never manufacture or infer Canonical results.
export async function inspectSubjectBatch(subjects,readOne) {
  if(!Array.isArray(subjects)||subjects.length===0||typeof readOne!=='function')
    throw new InspectBridgeError('INVALID_INSPECTION_BATCH');
  const ids=new Set();
  for(const s of subjects) {
    checkSubject(s);
    const key=subjectKey(s);
    if(ids.has(key))throw new InspectBridgeError('DUPLICATE_SUBJECT');
    ids.add(key);
  }
  const observations=[];
  let world=null,sourceIndex=null;
  for(const selector of subjects) {
    // Fail without returning partial successful output.
    const report=await readOne(selector);
    const row=projectCliInspectReport(report,selector);
    if(world===null) {
      world=row.provenance.worldFingerprint;
      sourceIndex=row.provenance.sourceIndexFingerprint;
    } else if(world!==row.provenance.worldFingerprint||
              sourceIndex!==row.provenance.sourceIndexFingerprint) {
      throw new InspectBridgeError('SNAPSHOT_DRIFT');
    }
    observations.push(row);
  }
  return {
    schemaVersion:'tei.research.inspection-batch/v0',
    effect:'read-only',authority:'derived_observation_only',
    acceptance:'not_asserted',
    worldFingerprint:world,sourceIndexFingerprint:sourceIndex,
    observations
  };
}
