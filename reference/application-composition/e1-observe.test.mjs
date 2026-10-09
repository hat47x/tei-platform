import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtemp, mkdir, writeFile, symlink, rm} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {observeSource, probes} from './e1-observe.mjs';

async function workspace(fn) {
  const root = await mkdtemp(join(tmpdir(), 'tei-composition-e1-'));
  try { await fn(root); } finally { await rm(root, {recursive:true, force:true}); }
}
const check = {id:'fixture', group:'case', path:'source/file.txt', contains:'known claim'};

test('the E1 probe catalog identifies two distinct consumers', () => {
  assert.ok(probes.some(p=>p.group === 'order_entry'));
  assert.ok(probes.some(p=>p.group === 'sui_stage0'));
  assert.equal(new Set(probes.map(p=>p.id)).size, probes.length);
});

test('observes only an actual literal in an existing source', async () => workspace(async root => {
  await mkdir(join(root,'source'));
  await writeFile(join(root,'source/file.txt'), 'known claim\n');
  const result = await observeSource(root,[check]);
  assert.equal(result.observations[0].status, 'observed');
  assert.equal(result.observations[0].reason, 'literal-present');
  assert.match(result.observations[0].sha256, /^[0-9a-f]{64}$/);
}));

test('missing or changed source cannot be silently accepted', async () => workspace(async root => {
  let r = await observeSource(root,[check]);
  assert.equal(r.observations[0].reason,'missing-source');
  await mkdir(join(root,'source'));
  await writeFile(join(root,'source/file.txt'),'a different meaning');
  r = await observeSource(root,[check]);
  assert.equal(r.observations[0].reason,'marker-missing');
  assert.equal(r.observations[0].status,'unresolved');
}));

test('refuses traversal and a symlink pointing outside the root', async () => workspace(async root => {
  await assert.rejects(() => observeSource(root,[{...check,path:'../outside.txt'}]), /invalid_probe/);
  const external = await mkdtemp(join(tmpdir(),'tei-outside-'));
  try {
    await writeFile(join(external,'outside.txt'),'known claim');
    await mkdir(join(root,'source'));
    await symlink(join(external,'outside.txt'), join(root,'source/file.txt'));
    const r = await observeSource(root,[check]);
    assert.equal(r.observations[0].reason,'outside-root');
  } finally { await rm(external,{recursive:true,force:true}); }
}));
