// Compile #5's actual TypeScript source for this browser consumer, no vendoring.
import { readFileSync, mkdirSync, writeFileSync, realpathSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { stripTypeScriptTypes } from 'node:module';
import { createHash } from 'node:crypto';

const here = dirname(fileURLToPath(import.meta.url));
const source = resolve(here, '../tei-studio-command/command.ts');
let input;
try { input = readFileSync(realpathSync(source), 'utf8'); }
catch { console.error('BLOCKED: #5 command.ts is not in this checkout.'); process.exit(2); }
if (!input.includes('export function mountCommandPanel(')) {
  console.error('BLOCKED: #5 Command export is incompatible or unavailable.'); process.exit(2);
}
const compiled = stripTypeScriptTypes(input, { mode: 'strip' });
const digest = createHash('sha256').update(input).digest('hex');
const out = resolve(here, '.generated/command.mjs');
mkdirSync(dirname(out), { recursive: true });
writeFileSync(out, `// Built from ../tei-studio-command/command.ts; source sha256: ${digest}\n${compiled}`);
console.log(`GENERATED .generated/command.mjs from #5 command.ts (sha256 ${digest})`);
