// Reports availability only. A present file is not proof of UI integration.
import { existsSync, statSync } from 'node:fs';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export function availability(args = []) {
  const found = [];
  const missing = [];
  for (const name of ['editor', 'command', 'graph']) {
    const assignment = args.find(arg => arg.startsWith(`--${name}=`));
    const path = assignment?.slice(name.length + 3);
    if (!path) { missing.push(`${name}: path not specified`); continue; }
    const absolute = resolve(path);
    if (!existsSync(absolute) || !statSync(absolute).isFile()) {
      missing.push(`${name}: not provided (${path})`);
    } else {
      found.push(`${name}: candidate file present (${path}); exports/integration unverified`);
    }
  }
  return { found, missing, reusable: false };
}

if (process.argv[1] && resolve(process.argv[1]) === resolve(fileURLToPath(import.meta.url))) {
  const result = availability(process.argv.slice(2));
  for (const message of result.found) console.log(`CANDIDATE ${message}`);
  for (const message of result.missing) console.log(`BLOCKED ${message}`);
  console.log('NOT VERIFIED: no supplied Editor/Command/Graph code is imported into this consumer.');
  process.exitCode = 2; // explicit dependency-blocked, not a test success
}
