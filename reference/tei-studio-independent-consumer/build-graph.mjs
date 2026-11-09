// Browser adapter build: compile #9's actual TS source without copying its code
// into the consumer. Node >=22.16 is required for stripTypeScriptTypes.
import { readFileSync, mkdirSync, writeFileSync, realpathSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { stripTypeScriptTypes } from 'node:module';
import { createHash } from 'node:crypto';

const here = dirname(fileURLToPath(import.meta.url));
const source = resolve(here, '../tei-studio-graph/graph.ts');
let input;
try { input = readFileSync(realpathSync(source), 'utf8'); }
catch { console.error('BLOCKED: #9 graph.ts is not in this checkout.'); process.exit(2); }
if (!input.includes('export function mountReadonlyGraph(')) {
  console.error('BLOCKED: #9 Graph export is incompatible or unavailable.'); process.exit(2);
}
const compiled = stripTypeScriptTypes(input, { mode: 'strip' });
const digest = createHash('sha256').update(input).digest('hex');
const out = resolve(here, '.generated/graph.mjs');
mkdirSync(dirname(out), { recursive: true });
writeFileSync(out, `// Built from ../tei-studio-graph/graph.ts; source sha256: ${digest}\n${compiled}`);
console.log(`GENERATED .generated/graph.mjs from #9 graph.ts (sha256 ${digest})`);
