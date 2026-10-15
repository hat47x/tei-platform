#!/usr/bin/env python3
"""Independent file-output probe for PoC B #1833. No TEI schema or resolver."""
import argparse
import copy
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

PR_HEAD = '1af4bf08c9afcc650ac74def3b0aad6728a5f58d'
NEW_HEAD = 'd560bddf289df68d896a651ecc6bfad61088f18e'
CONNECTED_HEAD = '059260521770e6b69eed0294287483dd60753587'
NEW_PIN_DIFF = {
    'main.go': '8d6eca235264f00a44e43484b757bd60fc00c270',
    'main_test.go': '7c8b9d651f545250b5091d79fb35a4212426dc49',
    'compose-ui.test.mjs': '77a168349cd004bc8b871f5ae406e189b11d9f71',
}
CONNECTED_PIN_DIFF = {
    'README.md': '391d8c0770cb1d88f511e30fbe72c8292cfed816',
    'compose-ui.mjs': '6e20b723ff7f0a3abf2ddce9267c879feeff3599',
    'compose-ui.test.mjs': '8d83bdff2c373f28b47eaf5129c0e76cdf600b2d',
    'connect-workbench.mjs': '29249c44d02af687103eff5124db95e5bea88c00',
    'connect-workbench.test.mjs': 'f587d7f3bde970bf16a6f6eb5c93bd948aec6714',
    'main_test.go': '074f34cd64168a2ec915ffb797e5d8c9b3a571cf',
}
BASE = 'tools/reference-studio-static-composition'
PINNED = {
    'README.md': '78fcc2210adb2f92f8fd690474c68bd29c2f754b',
    'fixture.json': '48f1167a957dfe07ba36c51d2c7b15e54d38c13a',
    'main.go': '891503713393cf17fde2bbbdf5b2f7b49672bf8b',
    'compose-ui.mjs': '2d8fbb83c5d930e1af93ec41dbbab2983bc44812',
    'main_test.go': '4bf9a962a76268d5763901d038cac3f2521b2ade',
    'compose-ui.test.mjs': 'c81824444286a3ee3a38fefda5cf33914222e75c',
}
GO_FILES = ('services.generated.go', 'services.provenance.json')
UI_FILES = ('ui.generated.mjs', 'ui.provenance.json')
CONNECTED_UI_FILES = ('ui.generated.mjs', 'ui.generated.d.mts', 'ui.provenance.json')
def pins_for(head):
    return {**PINNED, **(NEW_PIN_DIFF if head in (NEW_HEAD, CONNECTED_HEAD) else {}),
            **(CONNECTED_PIN_DIFF if head == CONNECTED_HEAD else {})}
def ui_files_for(head):
    return CONNECTED_UI_FILES if head == CONNECTED_HEAD else UI_FILES


class Blocked(Exception):
    pass


class Failed(Exception):
    pass


def sha(b):
    return hashlib.sha256(b).hexdigest()


def git_blob(b):
    return hashlib.sha1(b'blob ' + str(len(b)).encode() + b'\0' + b).hexdigest()


def require(cond, message):
    if not cond:
        raise Failed(message)


def prepare(root, lanes, head):
    base = root / BASE
    pins = pins_for(head)
    selected = pins if set(lanes) == {'go', 'ui'} else {
        k: v for k, v in pins.items() if k == 'fixture.json' or
        (k in ('main.go', 'main_test.go') if 'go' in lanes else
         k in ('compose-ui.mjs', 'compose-ui.test.mjs', 'connect-workbench.mjs', 'connect-workbench.test.mjs'))
    }
    for name, expected in selected.items():
        path = base / name
        if not path.is_file():
            raise Blocked(f'missing #1833 exact source: {path}')
        require(git_blob(path.read_bytes()) == expected, f'#1833 source differs from {head}: {name}')
    if 'ui' in lanes:
        resolved = root / 'reference/ui-customization/composition/resolve-ui-kit.mjs'
        if not resolved.is_file():
            raise Blocked('missing existing #997 UI Kit resolver')
        require(git_blob(resolved.read_bytes()) == 'a852de9751e596ac0460215aa389a1d3f404c267',
                'UI Kit resolver differs from main 1284294 reference')
    raw = (base / 'fixture.json').read_bytes()
    return base, raw, json.loads(raw)


def run_builder(root, base, raw, kind, timeout, head=PR_HEAD):
    with tempfile.TemporaryDirectory(prefix='.poc-b-7-', dir=base) as tmp:
        target = Path(tmp)
        fixture = target / 'candidate.json'
        fixture.write_bytes(raw)
        # One-level output dir preserves #1 generated UI's relative import resolution.
        output = Path(tempfile.mkdtemp(prefix='.poc-b-7-out-', dir=base))
        try:
            if kind == 'go':
                argv = ['go', 'run', './' + BASE, '-fixture', str(fixture), '-out', str(output)]
            else:
                argv = ['node', BASE + '/compose-ui.mjs', str(fixture), str(output)]
            try:
                p = subprocess.run(argv, cwd=root, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   timeout=timeout, check=False)
            except (FileNotFoundError, PermissionError, subprocess.TimeoutExpired) as e:
                raise Blocked(f'{kind} process unavailable: {type(e).__name__}') from e
            files = sorted(x.name for x in output.iterdir())
            expected = GO_FILES if kind == 'go' else ui_files_for(head)
            data = {name: (output / name).read_bytes() for name in expected if (output / name).is_file()}
            refs = []
            if kind == 'ui' and p.returncode == 0 and UI_FILES[0] in data:
                src = data[UI_FILES[0]].decode('utf-8')
                refs = re.findall(r'^import \{ [^\n]+ \} from "([^"]+)";', src, re.MULTILINE)
                refs = [(path, (output / path).resolve().is_file()) for path in refs]
            return {'code': p.returncode, 'stdout': p.stdout, 'stderr': p.stderr,
                    'files': files, 'data': data, 'imports': refs}
        finally:
            shutil.rmtree(output)


def assert_success(result, kind, raw, head=PR_HEAD):
    expected = GO_FILES if kind == 'go' else ui_files_for(head)
    require(result['code'] == 0, f'{kind}: unexpectedly rejected valid fixture, stderr={result["stderr"][:300]!r}')
    require(result['stdout'] == b'', f'{kind}: unexpected stdout; expected actual generated files')
    require(set(result['files']) == set(expected), f'{kind}: file outputs mismatch: {result["files"]}')
    source = result['data'][expected[0]]
    meta = json.loads(result['data'][expected[-1]])
    require(meta.get('fixtureSha256') == sha(raw), f'{kind}: fixture byte hash mismatch')
    require(meta.get('generatedSha256') == sha(source), f'{kind}: generated byte hash mismatch')
    require(meta.get('reference') == 'tei-studio-poc-b-reference-v0', f'{kind}: wrong reference')
    require(meta.get('authorityLimitations'), f'{kind}: authority limitations omitted')
    text = source.decode('utf-8')
    if kind == 'go':
        require(meta.get('serviceOrder') == ['studio.catalog', 'studio.validate'], 'go: wrong resolved order')
        for symbol in ('type CatalogService', 'type ValidateService', 'operations.CatalogWorkspace',
                       'operations.ValidateWorkspace', 'func Services(root string) []any'):
            require(symbol in text, f'go: generated service symbol missing: {symbol}')
        require('os/exec' not in text and 'plugin.Open' not in text, 'go: unexpected dynamic execution')
    else:
        refs = meta.get('uiContributions', [])
        require([v['slot'] for v in refs] == ['commands', 'graph'], 'ui: slot order mismatch')
        require([v['implementation'] for v in refs] == ['studio.commands', 'studio.graph'], 'ui: wrong implementation')
        require(all(v.get('kit') == 'studio.readonly' and v.get('version') == '0.1.0' for v in refs),
                'ui: incorrect resolved Kit trace')
        for v in refs:
            require(v['sourcePath'] in text, f'ui: generated import not linked to trace {v["slot"]}')
        if head == CONNECTED_HEAD:
            declarations = result['data'].get('ui.generated.d.mts')
            require(declarations is not None, 'ui: generated type declarations missing')
            require(meta.get('declarationsSha256') == sha(declarations),
                    'ui: declaration provenance fingerprint differs')
            for symbol in ('mountReadonlyGraph', 'mountCommandPanel'):
                require('export const ' + symbol + ' = ' in text,
                        'ui: required read-only mount alias absent: ' + symbol)
                require(('export { ' + symbol + ' } from ').encode() in declarations,
                        'ui: expected declaration export absent: ' + symbol)
        require('studioUIContributions' in text and 'eval(' not in text, 'ui: generated exports unsafe/missing')
    return {'sourceSha256': sha(source), 'provenanceSha256': sha(result['data'][expected[-1]]),
            'fixtureSha256': sha(raw), 'details': meta, 'imports': result['imports']}


def cases(fixture, ui_version_error=b'invalid_manifest'):
    def change(edit):
        value = copy.deepcopy(fixture)
        edit(value)
        return json.dumps(value, ensure_ascii=False, separators=(',', ':')).encode()
    return {
        'go.unknown': (change(lambda p: p['services'][0].update(id='missing.service')), b'unknown service'),
        'go.duplicate': (change(lambda p: p['services'][1].update(id=p['services'][0]['id'])), b'duplicate service'),
        'go.version': (change(lambda p: p['services'][0].update(version='9.0.0')), b'incompatible version'),
        'go.scope': (change(lambda p: p.update(scope='other.tenant')), b'unsupported or incomplete PoC B fixture'),
        'ui.missing_parent': (change(lambda p: (p['uiKit'].update(extends={'id': 'unknown.kit', 'version': '0.1.0'}), p['uiKit'].pop('slots'))), b'missing_exact_parent'),
        'ui.cycle': (change(lambda p: (p['uiKit'].update(extends={'id': p['uiKit']['id'], 'version': p['uiKit']['version']}), p['uiKit'].pop('slots'))), b'inheritance_cycle'),
        'ui.unknown_implementation': (change(lambda p: p['uiKit']['changes'][0].update(implementation='unknown.ui')), b'implementation_missing'),
        'ui.duplicate_slot': (change(lambda p: p['uiKit']['changes'][1].update(slot='graph')), b'invalid_change'),
        'ui.unknown_slot': (change(lambda p: p['uiKit']['changes'][1].update(slot='unknown')), b'unknown_extension_point'),
        'ui.duplicate_component': (change(lambda p: p['uiKit']['changes'][1].update(implementation='studio.graph')), b'duplicate_ui_component'),
        'ui.framework': (change(lambda p: p['uiKit']['framework'].update(major=2)), b'framework_mismatch'),
        'ui.version': (change(lambda p: p['uiKit'].update(version='latest')), ui_version_error),
    }


def go_toolchain_ready(root):
    try:
        p = subprocess.run(['go', 'version'], cwd=root, capture_output=True, text=True, timeout=10)
    except (FileNotFoundError, subprocess.TimeoutExpired, PermissionError):
        return False
    m = re.search(r'\bgo(\d+)\.(\d+)\.(\d+)\b', p.stdout)
    return p.returncode == 0 and bool(m) and tuple(map(int, m.groups())) >= (1, 26, 9)


def probe(root, timeout=60, lanes=('go', 'ui'), head=PR_HEAD):
    base, raw, fixture = prepare(root, lanes, head)
    pins = pins_for(head)
    result = {'referenceHead': head, 'inputGitBlobs': pins,
              'generation': {}, 'rejection': {}, 'blocked': {}, 'limits': {}}
    for kind in lanes:
        if kind == 'go' and not go_toolchain_ready(root):
            result['generation'][kind] = {'status': 'BLOCKED', 'reason': 'Go 1.26.9 or later is not available locally'}
            continue
        try:
            first = run_builder(root, base, raw, kind, timeout, head)
            second = run_builder(root, base, raw, kind, timeout, head)
            evidence = assert_success(first, kind, raw, head)
            assert_success(second, kind, raw, head)
            require(first['data'] == second['data'], f'{kind}: nondeterministic generated files/provenance')
            result['generation'][kind] = {'status': 'PASS', 'stableReplays': 2,
                                          'sourceSha256': evidence['sourceSha256'],
                                          'provenanceSha256': evidence['provenanceSha256'],
                                          'fixtureSha256': evidence['fixtureSha256'],
                                          'serviceOrder' if kind == 'go' else 'uiContributions':
                                          evidence['details'].get('serviceOrder' if kind == 'go' else 'uiContributions')}
            if kind == 'ui':
                result['limits']['uiImports'] = [{'path': path, 'existsInSnapshot': ok}
                                                   for path, ok in evidence['imports']]
        except Blocked as err:
            result['generation'][kind] = {'status': 'BLOCKED', 'reason': str(err)}
        except (Failed, OSError, json.JSONDecodeError) as err:
            result['generation'][kind] = {'status': 'FAIL', 'reason': str(err)}
    for label, (mutated, error) in cases(fixture, b'invalid_request' if head != PR_HEAD else b'invalid_manifest').items():
        kind, _ = label.split('.', 1)
        if kind not in lanes:
            continue
        if result['generation'][kind]['status'] != 'PASS':
            result['rejection'][label] = {'status': 'BLOCKED', 'reason': 'valid builder generation did not PASS'}
            continue
        try:
            actual = run_builder(root, base, mutated, kind, timeout, head)
            require(actual['code'] != 0, f'{label}: rejected input was accepted')
            require(actual['files'] == [], f'{label}: rejected input created outputs: {actual["files"]}')
            require(error in actual['stderr'], f'{label}: expected diagnostic {error!r} missing from real stderr')
            result['rejection'][label] = {'status': 'PASS', 'exitCode': actual['code'],
                                          'diagnostic': error.decode()}
        except Blocked as err:
            result['rejection'][label] = {'status': 'BLOCKED', 'reason': str(err)}
        except (Failed, OSError) as err:
            result['rejection'][label] = {'status': 'FAIL', 'reason': str(err)}
    result['blocked'] = {
        'go.missing_dependency': 'PoC B service fixture has no dependency selection/edge field; UI parent is separately checked',
        'go.cycle': 'PoC B service fixture cannot declare a dependency cycle; UI inheritance is separately checked',
        'regularVersusComposed': 'Generated Services are not hosted/executed; no two-service normal baseline comparison',
        'pluginLock': 'No production/Plugin Lock output; fixtureSha256 is not a Lock',
        'assetDigest': 'Imported UI asset SHA-256 is not recorded as a composed artifact fingerprint',
        'buildArtifact': 'No built package/binary digest or Build Lock in PoC B output',
    }
    if 'ui' in result['generation'] and result['generation']['ui']['status'] == 'PASS':
        if any(not v['existsInSnapshot'] for v in result['limits'].get('uiImports', [])):
            result['blocked']['uiImportTargets'] = 'UI import targets absent from this fixed snapshot (separate Draft PR inputs)'
    result['status'] = ('FAIL' if any(v['status'] == 'FAIL' for v in
        list(result['generation'].values()) + list(result['rejection'].values())) else 'BLOCKED')
    # Full PoC B qualification is deliberately not asserted even if local cases pass.
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', required=True, type=Path, help='main + exact #1833 files checkout/snapshot')
    parser.add_argument('--timeout', type=int, default=60)
    parser.add_argument('--lane', choices=('all', 'go', 'ui'), default='all')
    parser.add_argument('--pocb-head', choices=(PR_HEAD, NEW_HEAD, CONNECTED_HEAD), default=PR_HEAD)
    args = parser.parse_args(argv)
    try:
        result = probe(args.repo.resolve(), args.timeout, ('go', 'ui') if args.lane == 'all' else (args.lane,), args.pocb_head)
        print(json.dumps(result, sort_keys=True, ensure_ascii=False, indent=2))
        return 1 if result['status'] == 'FAIL' else 2
    except (Failed, OSError, ValueError, json.JSONDecodeError) as err:
        print(json.dumps({'status': 'FAIL', 'reason': str(err)}, ensure_ascii=False))
        return 1
    except Blocked as err:
        print(json.dumps({'status': 'BLOCKED', 'reason': str(err)}, ensure_ascii=False))
        return 2


if __name__ == '__main__':
    sys.exit(main())
