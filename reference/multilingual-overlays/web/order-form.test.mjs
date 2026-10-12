import assert from 'node:assert/strict';
import {execFileSync, spawnSync} from 'node:child_process';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
import {test} from 'node:test';
import {createLocalizedReferenceOrderEntryUI, validateOrderTitleManifest} from './order-form.mjs';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../..');
function manifest(locale) {
  const output = execFileSync('go',
    ['run', './reference/multilingual-overlays/cmd/render-manifest', locale],
    {cwd: root, encoding:'utf8', timeout:90000, env:{...process.env, GOTOOLCHAIN:'local'}});
  return JSON.parse(output);
}
const fakeReact = {
  createElement(type, props, ...children) { return {type, props:props ?? {}, children}; },
  useState(initial) { return [initial, ()=>{}]; },
};
function findElements(node, type) {
  if (node === null || node === undefined) return [];
  if (Array.isArray(node)) return node.flatMap(n => findElements(n, type));
  if (typeof node !== 'object') return [];
  return [...(node.type === type ? [node] : []), ...findElements(node.children ?? [], type)];
}

test('O2: real Go Plugin Lock -> JSON -> unchanged React starter in English', () => {
  const en = manifest('en-US');
  assert.equal(en.text, 'Confirm order');
  assert.equal(en.locale, 'en-US');
  assert.match(en.lockFingerprint, /^[a-f0-9]{64}$/);
  const component = createLocalizedReferenceOrderEntryUI(fakeReact, {
    manifest:en, invoke:async () => ({kind:'completed',stage:'completed'}),
  });
  const tree = component();
  const headings = findElements(tree,'h1');
  assert.equal(headings.length,1);
  assert.equal(headings[0].children[0],'Confirm order');
  const form = findElements(tree,'form');
  assert.equal(form.length,1);
  const inputs = findElements(tree,'input');
  assert.deepEqual(inputs.map(x=>x.props.name),['loginId','credential','from','row']);
  assert.equal(inputs.find(x=>x.props.name === 'credential').props.type,'password');
  assert.equal(findElements(tree,'button').length,1);
  assert.equal(tree.props['data-tei-ui-plugin'],'tei.ui.web.react.action-form.reference');
});

test('O2: ordinary plugin works in original Japanese with same programmatic action', () => {
  const ja = manifest('ja-JP');
  const en = manifest('en-US');
  assert.equal(ja.text,'受注を確定');
  assert.equal(ja.fromSource,true);
  assert.equal(en.fromSource,false);
  assert.equal(ja.sourceRevision,en.sourceRevision);
  assert.equal(ja.lockFingerprint,en.lockFingerprint);
  const render=(display)=>createLocalizedReferenceOrderEntryUI(fakeReact,{
    manifest:display,invoke:async ()=>({kind:'completed'}),
  })();
  assert.equal(findElements(render(ja),'h1')[0].children[0],'受注を確定');
  assert.deepEqual(findElements(render(ja),'input').map(x=>x.props.name),
    findElements(render(en),'input').map(x=>x.props.name));
});

test('O2: fake/foreign/changed manifest is not accepted by UI adapter', () => {
  const good = manifest('en-US');
  for (const change of [
    {target:{id:'std.order-entry',version:'2.0.0'}},
    {resource:{...good.resource,field:'business-rule'}},
    {resource:{...good.resource,owner:'other-plugin'}},
    {lockFingerprint:'unlocked'},
    {locale:'fr-FR'},
    {text:''},
    {fromSource:true},
    {sourceRevision:'sha256:invalid'},
    {fallback:true},
  ]) {
    assert.throws(()=>validateOrderTitleManifest({...good,...change}),
      /invalid_locked_display_manifest/);
  }
});

test('O2: unsupported locale is rejected by locked Go projection before reaching UI', () => {
  const result=spawnSync('go',
    ['run','./reference/multilingual-overlays/cmd/render-manifest','fr-FR'],
    {cwd:root,encoding:'utf8',timeout:90000,env:{...process.env,GOTOOLCHAIN:'local'}});
  assert.notEqual(result.status,0);
  assert.equal(result.stdout,'');
  assert.match(result.stderr,/missing-translation/);
});

test('O2: no user content interpreted as HTML markup', () => {
  const doc=manifest('en-US');
  const withMarkup={...doc,text:'<img src=x onerror=alert(1)>'};
  const component=createLocalizedReferenceOrderEntryUI(fakeReact,{
    manifest:withMarkup,invoke:async()=>({kind:'completed'}),
  });
  const tree=component();
  assert.equal(findElements(tree,'h1')[0].children[0],withMarkup.text);
  assert.equal(findElements(tree,'img').length,0);
});
