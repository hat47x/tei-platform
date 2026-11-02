#!/usr/bin/env python3
"""Read-only same-input comparison of real TEI CLI operations/artifacts output.

Evidence comparator only: not a source acceptance, provider read, authorization,
publication, Consumer receipt, or business-result verification service.
"""
import argparse
import hashlib
import json
import subprocess
import sys
from collections import Counter
from pathlib import Path


class WitnessError(ValueError):
    pass


def need(ok, why):
    if not ok:
        raise WitnessError(why)


def fingerprint(blob):
    return 'sha256:' + hashlib.sha256(blob.encode('utf-8')).hexdigest()


def artifact_digest(artifact):
    rule = artifact['rule']
    text = ('tei.artifact/v0\n' + artifact['kind'] + '\n'
            + artifact['inputRevision'] + '\n' + rule['id'] + '\n'
            + rule['version'] + '\n' + rule['fingerprint'] + '\n'
            + artifact['generator'] + '\n' + artifact['content']
            + '\n--unresolved--\n')
    for x in sorted(artifact.get('unresolved', []),
                    key=lambda a: (a['subject'], a['reason'], a['detail'])):
        text += x['subject'] + '\t' + x['reason'] + '\t' + x['detail'] + '\n'
    return fingerprint(text)


def manifest_digest(manifest):
    text = 'tei.artifact.manifest/v0\n' + manifest['inputRevision'] + '\n'
    entries = manifest['entries']
    for e in entries:
        text += (e['kind'] + '\t' + e['rule'] + '\t' + e['version']
                 + '\t' + e['fingerprint'] + '\t' + e['digest'] + '\t'
                 + str(e['unresolved']) + '\n')
    text += 'complete=' + str(manifest['complete']).lower() + '\n'
    return fingerprint(text)


def compare(operations, artifacts, application):
    """Return limited evidence; fail on missing/mismatched observations.

    Main's SurfaceWorkspace and RenderWorkspace are separate workspace.Load calls.
    Equal modelRevision therefore does not guarantee one atomic acquisition.
    """
    need(isinstance(operations, dict) and isinstance(artifacts, dict), 'not_object')
    need(operations.get('schemaVersion') == 'tei.operations.operation-surface/v0', 'surface_schema_unknown')
    need(operations.get('effect') == 'read-only', 'surface_effect_unexpected')
    need(operations.get('application') == application, 'surface_application_mismatch')
    revision = operations.get('modelRevision')
    need(isinstance(revision, str) and revision, 'surface_revision_unknown')
    need(artifacts.get('inputRevision') == revision, 'cross_load_revision_mismatch')
    manifest = artifacts.get('manifest')
    need(isinstance(manifest, dict), 'manifest_absent')
    need(manifest.get('inputRevision') == revision, 'manifest_revision_mismatch')
    need(isinstance(manifest.get('entries'), list), 'manifest_entries_absent')
    need(isinstance(manifest.get('complete'), bool), 'manifest_completeness_unknown')
    need(manifest.get('digest') == manifest_digest(manifest), 'manifest_digest_mismatch')
    items = artifacts.get('artifacts')
    need(isinstance(items, list) and items, 'artifact_bodies_absent')
    expected_entries = []
    for item in items:
        need(item.get('inputRevision') == revision, 'artifact_revision_mismatch')
        need(item.get('digest') == artifact_digest(item), 'artifact_envelope_digest_mismatch')
        expected_entries.append((item['kind'], item['rule']['id'], item['rule']['version'],
                                 item['rule']['fingerprint'], item['digest'],
                                 len(item.get('unresolved', []))))
    actual_entries = [(e['kind'], e['rule'], e['version'], e['fingerprint'],
                       e['digest'], e['unresolved']) for e in manifest['entries']]
    need(Counter(actual_entries) == Counter(expected_entries), 'manifest_member_mismatch')
    need(not artifacts.get('findings'), 'artifact_consistency_findings')
    need(manifest['complete'] == all(not i.get('unresolved') for i in items),
         'manifest_completeness_mismatch')
    surface_artifacts = [i for i in items if i.get('kind') == 'operation-surface']
    need(len(surface_artifacts) == 1, 'operation_surface_artifact_missing_or_duplicate')
    try:
        embedded = json.loads(surface_artifacts[0]['content'])
    except (TypeError, ValueError) as e:
        raise WitnessError('operation_surface_content_not_json') from e
    need(embedded.get('schemaVersion') == operations['schemaVersion'], 'embedded_schema_mismatch')
    for field in ('application', 'rule', 'operations', 'gaps', 'summary'):
        need(embedded.get(field) == operations.get(field), 'surface_' + field + '_mismatch')
    need(isinstance(operations.get('worldFingerprint'), str)
         and operations['worldFingerprint'], 'world_fingerprint_absent')
    need(isinstance(operations.get('rule'), dict)
         and operations['rule'].get('fingerprint'), 'operation_rule_missing')
    gaps = operations.get('gaps')
    need(isinstance(gaps, list), 'gaps_not_observed')
    return {
        'status': 'read_only_cross_projection_consistent',
        'meaningAccepted': False, 'sameAtomicReadProven': False,
        'providerOrHostProven': False, 'consumerReadProven': False,
        'revision': revision, 'artifactCount': len(items),
        'unresolvedGapCount': len(gaps), 'actionReady': False,
    }


def run_cli(binary, *args):
    p = subprocess.run([str(binary), *args], capture_output=True, text=True,
                       timeout=120, check=False, encoding='utf-8')
    need(p.returncode == 0, 'cli_failed_' + args[0])
    need(not p.stderr, 'cli_stderr_' + args[0])
    try:
        return json.loads(p.stdout)
    except ValueError as e:
        raise WitnessError('cli_nonjson_' + args[0]) from e


def main(argv=None):
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--workspace', type=Path, required=True)
    parser.add_argument('--application', required=True)
    parser.add_argument('--expected-head', required=True)
    args = parser.parse_args(argv)
    need(args.binary.is_file(), 'binary_missing')
    need((args.workspace / 'tei.workspace.yaml').is_file(), 'workspace_missing')
    need(len(args.expected_head) == 40 and
         all(c in '0123456789abcdef' for c in args.expected_head), 'head_not_exact_sha')
    version = run_cli(args.binary, 'version', '-json')
    need(version.get('vcsRevision') == args.expected_head, 'binary_head_mismatch')
    need(version.get('vcsModified', False) is False, 'binary_modified')
    root = str(args.workspace.resolve())
    surface = run_cli(args.binary, 'operations', '-workspace', root,
                      '-application', args.application)
    artifact_set = run_cli(args.binary, 'artifacts', '-workspace', root,
                           '-application', args.application)
    print(json.dumps(compare(surface, artifact_set, args.application), sort_keys=True))


if __name__ == '__main__':
    try:
        main()
    except (WitnessError, OSError, subprocess.TimeoutExpired, KeyError, TypeError) as exc:
        print('read_only_join_unqualified: ' + type(exc).__name__ + ': ' + str(exc), file=sys.stderr)
        sys.exit(1)
