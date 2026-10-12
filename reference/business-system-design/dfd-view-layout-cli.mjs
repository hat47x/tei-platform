#!/usr/bin/env node
// Read-only layout preview. This entrypoint does not write Source or view files.
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {previewLayoutMove} from './dfd-view-layout.mjs';

const digest = bytes => 'sha256:' + createHash('sha256').update(bytes).digest('hex');
function read(path) {
  const bytes = readFileSync(path);
  return {path, hash:digest(bytes), value:JSON.parse(bytes.toString('utf8'))};
}
function cli(argv) {
  const fields = ['-projection','-layout','-edit'];
  if (argv.length !== 6) throw new Error('REQUIRED: -projection file -layout file -edit file');
  const files = {};
  for (let i=0; i<argv.length; i+=2) {
    const flag = argv[i], value = argv[i+1];
    if (!fields.includes(flag) || files[flag] || !value || value.startsWith('-')) {
      throw new Error('INVALID_LAYOUT_CLI_ARGS');
    }
    files[flag] = read(value);
  }
  if (Object.keys(files).length !== 3) throw new Error('REQUIRED_THREE_LAYOUT_FILES');
  const projection = files['-projection'];
  const layout = files['-layout'];
  const edit = files['-edit'];
  const result = previewLayoutMove(projection.value, projection.hash,
    layout.value, layout.hash, edit.value);
  for (const file of Object.values(files)) {
    if (digest(readFileSync(file.path)) !== file.hash) {
      throw new Error('VIEW_LAYOUT_INPUT_CHANGED_DURING_PREVIEW: ' + file.path);
    }
  }
  result.inputSnapshotsVerified = true;
  result.inputSnapshots = {
    projection:projection.hash, layout:layout.hash, edit:edit.hash
  };
  process.stdout.write(JSON.stringify(result,null,2) + '\n');
}
try { cli(process.argv.slice(2)); }
catch (error) { process.stderr.write(String(error.message) + '\n'); process.exitCode=1; }
