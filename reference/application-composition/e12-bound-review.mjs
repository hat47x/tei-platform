// #1016 E12: read-only CLI for E7 paired evidence and #1120 Go-shaped output.
// E11 is the sole owner of the combined review decision; no parallel resolver.
import {readFile,stat} from 'node:fs/promises';
import {resolve} from 'node:path';
import {pathToFileURL} from 'node:url';
import {qualifyPairedCompositionReview} from './e11-review-gate.mjs';

const MAX_BYTES = 16*1024*1024;
const validId = s => typeof s === 'string' && !!s.trim() && s === s.trim();

// Inputs are external JSON and NOT proof of common source revision, provenance,
// permission, or correctness of Go Validate/Impact. No secret data is printed.
export async function inspectEvidenceFiles({baselinePath,currentPath,modelrefPath,application,scope}={}) {
  if (![baselinePath,currentPath,modelrefPath,application,scope].every(validId)) {
    throw new TypeError('invalid_arguments');
  }
  const read = async path => {
    const info=await stat(path);
    if (!info.isFile() || info.size > MAX_BYTES) throw new TypeError('invalid_evidence_file');
    const body=await readFile(path);
    if (body.byteLength > MAX_BYTES) throw new TypeError('invalid_evidence_file');
    return JSON.parse(body.toString('utf8'));
  };
  const [baseline,current,typed] = await Promise.all([
    read(baselinePath),read(currentPath),read(modelrefPath),
  ]);
  // The Go JSON carrier is a caller-generated grouping, not a new Lock/DSL.
  return qualifyPairedCompositionReview({
    baseline,current,projection:typed?.projection,
    diagnostics:typed?.diagnostics,affected:typed?.affected,application,scope,
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  if (process.argv.length !== 7) {
    process.stderr.write('Usage: node e12-bound-review.mjs <baseline.json> <current.json> <modelref.json> <application> <scope>\n');
    process.exitCode=1;
  } else {
    try {
      const result=await inspectEvidenceFiles({
        baselinePath:process.argv[2], currentPath:process.argv[3],
        modelrefPath:process.argv[4],application:process.argv[5],scope:process.argv[6],
      });
      process.stdout.write(JSON.stringify(result,null,2)+'\n');
      // Until shared Go snapshot provenance is proven, review never authorizes
      // a successful build/deployment (no current result may exit 0).
      process.exitCode=2;
    } catch {
      // Never echo raw JSON, source paths or private values from errors.
      process.stderr.write('Invalid review evidence input\n');
      process.exitCode=1;
    }
  }
}
