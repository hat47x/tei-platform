// Opt-in local E2E entry point for a trusted, explicitly selected TEI CLI binary.
// No network, no shell, no Canonical writes, and no automatic discovery of other apps.
// WARNING: Source disclosure authorization is the caller's responsibility.
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {resolve} from 'node:path';
import {fileURLToPath} from 'node:url';
import {inspectSubjectBatch,InspectBridgeError} from './inspect-cli-bridge.mjs';
import {linkApplicationPattern,linkPatternRelation} from './project-inspection-v0.mjs';

const execFileAsync=promisify(execFile);
const REFERENCE_SUBJECTS=Object.freeze([
  {kind:'application',id:'order_entry'},
  {kind:'application-pattern',id:'order_entry.order_lines'},
  {kind:'semantic-relation',id:'sales.order.lines'},
  {kind:'semantic-data',id:'sales.order'},
  {kind:'application-pattern',id:'order_entry.customer_reference'}
]);

function cliArgs(args) {
  if(args.length!==4)throw new InspectBridgeError('EXPECTED_BINARY_AND_WORKSPACE');
  const opts={};
  for(let i=0;i<4;i+=2) {
    const flag=args[i];
    if(!['--tei-bin','--workspace'].includes(flag) || opts[flag]!==undefined ||
       typeof args[i+1]!=='string' || args[i+1].length===0)
      throw new InspectBridgeError('INVALID_CLI_ARGUMENTS');
    opts[flag]=resolve(args[i+1]);
  }
  if(!opts['--tei-bin']||!opts['--workspace'])
    throw new InspectBridgeError('EXPECTED_BINARY_AND_WORKSPACE');
  return {binary:opts['--tei-bin'],workspace:opts['--workspace']};
}

async function executeOne(binary,workspace,subject) {
  let output;
  try {
    output=await execFileAsync(binary,[
      'inspect','-workspace',workspace,
      '-subject-kind',subject.kind,'-subject-id',subject.id
    ],{encoding:'utf8',shell:false,maxBuffer:1024*1024,timeout:30000});
  }catch {
    // Suppress raw error/stdout/stderr: they might contain sensitive Source or paths.
    throw new InspectBridgeError('TEI_INSPECT_EXECUTION_FAILED');
  }
  if(output.stderr.trim()!=='')throw new InspectBridgeError('TEI_INSPECT_UNEXPECTED_STDERR');
  let report;
  try{report=JSON.parse(output.stdout);}
  catch{throw new InspectBridgeError('TEI_INSPECT_INVALID_JSON');}
  return report;
}

export async function runReferenceInspection(binary,workspace,readOne=executeOne) {
  const batch=await inspectSubjectBatch(REFERENCE_SUBJECTS,subject=>readOne(binary,workspace,subject));
  const get=(kind,id)=>batch.observations.find(o=>o.subject.kind===kind&&o.subject.id===id);
  const application=get('application','order_entry');
  const pattern=get('application-pattern','order_entry.order_lines');
  const relation=get('semantic-relation','sales.order.lines');
  return {
    ...batch,
    links:{
      applicationPattern:linkApplicationPattern(application,pattern),
      patternRelation:linkPatternRelation(pattern,relation)
    }
  };
}

if(process.argv[1] && resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  (async()=>{
    const {binary,workspace}=cliArgs(process.argv.slice(2));
    const result=await runReferenceInspection(binary,workspace);
    process.stdout.write(JSON.stringify(result,null,2)+'\n');
  })().catch(err=>{
    const code=err instanceof InspectBridgeError?err.code:'REFERENCE_INSPECTION_FAILED';
    process.stderr.write(code+'\n');
    process.exitCode=1;
  });
}
