// E7: transport E5 report and Source Index as a same-invocation pair to E6.
// Read-only; an envelope tag is not trusted provenance, acceptance or authority.
import {readFile} from 'node:fs/promises';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';
import {assessParsedApplicationSourceDrift} from './e6-typed-index-bridge.mjs';

const format='tei.e5-paired-source-observation.v1';
const qualification='read-only-paired-input-not-an-authority';
const object=x=>x !== null && typeof x === 'object' && !Array.isArray(x);
const limit=32*1024*1024;

export function assessPairedEvidence(baseline,current) {
  const context={fullClosure:false,authority:'none'};
  if (![baseline,current].every(x=>object(x) && x.format === format &&
      x.qualification === qualification && object(x.report) && object(x.index))) {
    return {...context,status:'unresolved',reason:'invalid_or_unqualified_evidence_envelope'};
  }
  // Exact source-reference consistency and declared dependency drift remain
  // E6/E4 responsibilities. This wrapper grants no additional trust to JSON.
  return assessParsedApplicationSourceDrift({
    baselineReport:baseline.report,currentReport:current.report,
    baselineIndex:baseline.index,currentIndex:current.index,
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  const paths=process.argv.slice(2);
  if (paths.length !== 2 || paths.some(p=>!p || p.startsWith('-'))) {
    process.stderr.write('Usage: node e7-paired-evidence.mjs <baseline.json> <current.json>\n');
    process.exitCode=1;
  } else {
    try {
      const contents=await Promise.all(paths.map(async p=>{
        const text=await readFile(p,'utf8');
        if (Buffer.byteLength(text,'utf8') > limit) throw new Error('snapshot_too_large');
        try { return JSON.parse(text); }
        catch { throw new Error('invalid_snapshot_json'); }
      }));
      const result=assessPairedEvidence(contents[0],contents[1]);
      process.stdout.write(JSON.stringify(result,null,2)+'\n');
      if (result.status !== 'limited-unchanged') process.exitCode=2;
    } catch (error) {
      process.stderr.write('Invalid evidence input: '+(error?.message || 'unknown')+'\n');
      process.exitCode=1;
    }
  }
}
