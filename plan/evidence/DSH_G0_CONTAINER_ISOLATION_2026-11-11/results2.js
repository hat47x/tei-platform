// Summarize 4-slot preflight from reviewer-side request logs + slot DSH homes.
const fs = require('fs'), path = require('path'), crypto = require('crypto'), zlib = require('zlib');
const T = process.argv[2], verbose = process.argv[3];
const sha = b => crypto.createHash('sha256').update(b).digest('hex');
const direct = fs.readFileSync(path.join(T, 'log/catalog.direct.json'));
const walk = d => fs.readdirSync(d, { withFileTypes: true }).flatMap(e => e.isDirectory() ? walk(path.join(d, e.name)) : [path.join(d, e.name)]);
const canaries = (files) => { const found = {}; let frames = 0; for (const f of files) { let b = fs.readFileSync(f); if (/\.zstd?$/.test(f)) { try { b = zlib.zstdDecompressSync(b); frames++; } catch (e) { found['ZSTD_FAIL:' + e.message] = 1; } } for (const m of b.toString('latin1').match(/G0_CANARY_[A-Z0-9_]+/g) || []) found[m] = (found[m] || 0) + 1; } return { found, zstdFiles: frames }; };
const summary = {};
for (const s of ['h2-a', 'h3-a', 'h3-b', 'h2-b']) {
  const L = path.join(T, 'log', s);
  const reqs = fs.readdirSync(L).filter(f => /^req-\d+\.json$/.test(f)).sort();
  let last = null; const keys = new Set();
  for (const f of reqs) { const j = JSON.parse(fs.readFileSync(path.join(L, f))); Object.keys(j).forEach(k => keys.add(k)); if ((j.tools || []).length) last = j; }
  const uses = [], res = {};
  for (const m of last.messages) for (const c of (Array.isArray(m.content) ? m.content : [])) {
    if (c.type === 'tool_use') uses.push(c);
    if (c.type === 'tool_result') res[c.tool_use_id] = { err: !!c.is_error, text: typeof c.content === 'string' ? c.content : (c.content || []).map(x => x.text || '').join('') };
  }
  const out = { requests: reqs.length, tool_use: uses.length, tool_result: Object.keys(res).length, paired: uses.every(u => res[u.id]), topKeys: [...keys].join(',') };
  const R = i => (res[uses[i].id] || { text: 'MISSING' });
  const cat = R(2).text;
  out.g2_catalog = { direct_bytes: direct.length, direct_sha: sha(direct).slice(0, 16), result_bytes: Buffer.byteLength(cat), result_sha: sha(cat).slice(0, 16), identical: Buffer.from(cat).equals(direct), identical_trim: cat.trim() === direct.toString().trim(), parses: (() => { try { JSON.parse(cat); return true; } catch { return false; } })() };
  out.skill_tool = { err: R(3).err, bytes: Buffer.byteLength(R(3).text), sha: sha(R(3).text).slice(0, 16), head: R(3).text.slice(0, 70).replace(/\n/g, ' ') };
  const d = R(5).text;
  out.denied = { rc_lines: (d.match(/rc\[[^\]]+\]=\d+/g) || []).join(' '), permission_denied: (d.match(/Permission denied/g) || []).length };
  out.read_reviewer = R(6).text.slice(0, 90); out.read_other_skill = R(7).text.slice(0, 110); out.grep_others = R(8).text.slice(0, 120).replace(/\n/g, ' | ');
  out.log_canaries = canaries(walk(L).filter(f => !/steps\.json$/.test(f)));
  out.home_canaries = canaries(walk(path.join(T, 'slots', s, 'home')));
  out.home_files = walk(path.join(T, 'slots', s, 'home')).length;
  summary[s] = out;
  console.log(`\n##### ${s}`);
  for (const i of [0, 1, 4, 9]) console.log(`--- [${uses[i].input.description}] err=${R(i).err}\n${R(i).text.slice(0, 1100)}`);
  if (verbose) console.log('--- denied raw\n' + d.slice(0, 2500));
  console.log(JSON.stringify(out, null, 1));
}
console.log('SUMMARY_JSON ' + JSON.stringify(summary));
