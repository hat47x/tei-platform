#!/usr/bin/env python3
"""T1-A *candidate-only* projection of saved REA JSON; not TEI admission.

This code never loads a TEI workspace, issues a canonical Subject, grants access,
or makes a business or runtime claim. The REA archive is untrusted input.
"""
import argparse
import hashlib
import json
from pathlib import Path, PurePosixPath

import audit

MAX_RECORDS = 4096


def _file(root, name, cap=4096):
    raw = audit._read(root, name, cap)
    if raw is None:
        raise ValueError('missing saved report member: ' + name)
    return raw


def _path(value):
    if not isinstance(value, str) or not value or '\\' in value or '\x00' in value:
        return None
    p = PurePosixPath(value)
    if p.is_absolute() or any(part == '..' for part in value.split('/')) or str(p) != value:
        return None
    return value


def _source_hashes(root, selected):
    prefix = _path(selected)
    if prefix is None:
        raise ValueError('invalid selected-target')
    raw = _file(root, 'fixture-source-sha256.txt')
    try:
        lines = raw.decode('ascii').splitlines()
    except UnicodeDecodeError as exc:
        raise ValueError('invalid manifest encoding') from exc
    hashes = {}
    for line in lines:
        parts = line.split('  ', 1)
        if len(parts) != 2 or len(parts[0]) != 64 or any(c not in '0123456789abcdef' for c in parts[0]):
            raise ValueError('invalid source manifest entry')
        file_path = _path(parts[1])
        if file_path is None or not file_path.startswith(prefix + '/'):
            raise ValueError('source manifest outside selected target')
        rel = file_path[len(prefix) + 1:]
        if rel in hashes:
            raise ValueError('duplicate manifest source')
        hashes[rel] = parts[0]
    if set(hashes) != {'index.js', 'package.json'}:
        raise ValueError('T0 requires exactly the two known public fixture files')
    return hashes


def _verify_saved_snapshots(report_dir, hashes):
    """Verify saved public fixture bytes on both sides of REA process execution.

    Same bytes at two instants are not atomic closure or A->B->A detection.
    This fixture explicitly supports only two top-level public files.
    """
    report_dir = Path(report_dir)
    before, after = report_dir / 'source-pre', report_dir / 'source-post'
    if (before.is_symlink() or after.is_symlink() or
            before.exists() != after.exists()):
        raise ValueError('inconsistent or symlinked source snapshot directories')
    if not before.exists():
        return False
    if not before.is_dir() or not after.is_dir():
        raise ValueError('source snapshots must be directories')
    for name, digest in hashes.items():
        # Top-level names have already been fixed by _source_hashes.
        left = _file(before, name, 1024 * 1024)
        right = _file(after, name, 1024 * 1024)
        if hashlib.sha256(left).hexdigest() != digest:
            raise ValueError('pre-source byte differs from saved manifest: ' + name)
        if hashlib.sha256(right).hexdigest() != digest:
            raise ValueError('post-source byte differs from saved manifest: ' + name)
    return True


def _location(evidence):
    descriptor = evidence.get('location', {}) if isinstance(evidence, dict) else {}
    if not isinstance(descriptor, dict) or descriptor.get('available') is not True:
        return None
    loc = descriptor.get('value')
    if not isinstance(loc, dict):
        return None
    kind = loc.get('kind')
    if kind == 'artifact-path':
        p = _path(loc.get('path'))
        return {'kind': 'artifact-path', 'path': p} if p else None
    if kind == 'source-range':
        p = _path(loc.get('source'))
        start, end = loc.get('start'), loc.get('end')
        if not p or not isinstance(start, dict) or not isinstance(end, dict):
            return None
        points = (start.get('line'), start.get('column'), end.get('line'), end.get('column'))
        if not all(type(v) is int for v in points) or points[0] < 1 or points[2] < 1 or min(points[1], points[3]) < 0:
            return None
        if points[2:] < points[:2]:
            return None
        return {'kind': kind, 'path': p, 'start': {'line': points[0], 'column': points[1]},
                'end': {'line': points[2], 'column': points[3]}}
    return None


def _record(identity, record_kind, evidence, hashes, snapshots_verified=False, **extra):
    if not isinstance(identity, str) or not identity:
        raise ValueError('missing external occurrence identity')
    if not isinstance(evidence, dict):
        raise ValueError('missing source-backed evidence on occurrence')
    loc = _location(evidence)
    artifact = evidence.get('artifact')
    digest = artifact.get('sha256') if isinstance(artifact, dict) else None
    path = loc['path'] if loc else None
    binding = 'not-independently-bound'
    if path in hashes and isinstance(digest, str):
        if digest != hashes[path]:
            raise ValueError('source digest conflict for ' + identity)
        binding = 'report-pre-post-byte-match' if snapshots_verified else 'saved-manifest-match-only'
    return {'kind': record_kind, 'externalOccurrenceID': identity,
            'observedState': evidence.get('state', 'unknown'),
            'authority': evidence.get('authority', 'unknown'),
            'source': loc, 'sourceBinding': binding,
            'evidenceCoverage': evidence.get('coverage'), **extra}


def project(report_dir):
    report_dir = Path(report_dir)
    a = audit.audit(report_dir)
    result = {'projectionSchema': 'rea.external-observation.candidate-test/v0',
              'effect': 'read-only', 'TEICanonicalSubject': None, 'acceptedMeaning': False,
              'qualifiedTEISource': False, 'sourceClosure': 'not-qualified',
              'rawReportSHA256': a['stdoutSHA256'], 'processOutcome': a['processOutcome'],
              'analysisQualification': a['analysisQualification'],
              'admission': 'refused', 'records': [], 'unknowns': []}
    if (a['processExitCode'] != 0 or a['reportShape'] != 'json-document' or
            a['processReportConsistency'] != 'consistent'):
        return result
    raw = _file(report_dir, 'analysis.json', audit.MAX_REPORT)
    obj = json.loads(raw)
    data = obj.get('data') if isinstance(obj, dict) and obj.get('ok') is True else None
    if not isinstance(data, dict) or data.get('operation') != 'analyze_javascript_application':
        return result
    normalized = data.get('normalized_result')
    subject, provider = data.get('subject'), data.get('provider')
    if not all(isinstance(x, dict) for x in (normalized, subject, provider)):
        return result
    if (provider.get('id') != 'rea-javascript-application' or
            not isinstance(provider.get('version'), str) or
            not isinstance(data.get('evidence_id'), str) or not data['evidence_id']):
        return result
    subject_hash = (subject.get('digest') or {}).get('sha256')
    if not isinstance(subject_hash, str) or len(subject_hash) != 64 or subject_hash != normalized.get('root_artifact_sha256'):
        return result
    graph = normalized.get('graph')
    if not isinstance(graph, dict) or graph.get('schema') != 'JavaScriptApplicationGraph':
        return result
    nodes, edges = graph.get('nodes'), graph.get('edges')
    if not isinstance(nodes, list) or not isinstance(edges, list) or len(nodes) + len(edges) > MAX_RECORDS:
        return result
    target = _file(report_dir, 'selected-target.txt', 4096).decode('utf-8').strip()
    hashes = _source_hashes(report_dir, target)
    snapshots_verified = _verify_saved_snapshots(report_dir, hashes)
    identifiers = [n.get('node_id') for n in nodes if isinstance(n, dict)]
    if len(identifiers) != len(nodes) or len(set(identifiers)) != len(nodes) or any(not isinstance(x, str) for x in identifiers):
        return result
    records, seen = [], set()
    for n in nodes:
        observations = n.get('observations')
        if not isinstance(observations, list) or len(observations) > MAX_RECORDS:
            return result
        for obs in observations:
            if not isinstance(obs, dict):
                return result
            rec = _record(obs.get('observation_id'), 'graph-node-observation', obs.get('evidence'), hashes, snapshots_verified=snapshots_verified,
                          graphNodeID=n['node_id'], graphNodeKind=n.get('kind'))
            if rec['externalOccurrenceID'] in seen:
                return result
            seen.add(rec['externalOccurrenceID'])
            records.append(rec)
            if len(records) > MAX_RECORDS:
                return result
    for e in edges:
        if not isinstance(e, dict) or e.get('source_node_id') not in identifiers or e.get('target_node_id') not in identifiers:
            return result
        rec = _record(e.get('edge_id'), 'graph-relation-observation', e.get('evidence'), hashes, snapshots_verified=snapshots_verified,
                      sourceNodeID=e['source_node_id'], targetNodeID=e['target_node_id'], relation=e.get('relation'))
        if rec['externalOccurrenceID'] in seen:
            return result
        seen.add(rec['externalOccurrenceID'])
        records.append(rec)
        if len(records) > MAX_RECORDS:
            return result
    semantic = normalized.get('semantic_graph')
    if not isinstance(semantic, dict):
        return result
    unknowns = semantic.get('unknowns')
    if not isinstance(unknowns, list) or len(unknowns) > MAX_RECORDS:
        return result
    unknown_ids = [u.get('unknown_id') for u in unknowns if isinstance(u, dict)]
    if (len(unknown_ids) != len(unknowns) or
            any(not isinstance(i, str) or not i for i in unknown_ids) or
            len(unknown_ids) != len(set(unknown_ids))):
        return result
    result.update(admission='external-candidates-only', records=records,
                  unknowns=[{'externalUnknownID': u.get('unknown_id'), 'family': u.get('family'),
                             'reason': u.get('reason')} for u in unknowns if isinstance(u, dict)],
                  graphCoverage=graph.get('coverage'), semanticCoverage=semantic.get('coverage'),
                  limitationCount=len(data.get('limitations', [])) if isinstance(data.get('limitations'), list) else None,
                  REAEvidenceID=data.get('evidence_id'),
                  providerDescriptor={'id': provider.get('id'), 'version': provider.get('version')},
                  subjectDigest=subject_hash,
                  sourceManifestStatus='report-pre-post-byte-match' if snapshots_verified else 'saved-run-claimed-only',
                  selection=target)
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('report_dir')
    args = p.parse_args()
    print(json.dumps(project(args.report_dir), ensure_ascii=False, sort_keys=True))


if __name__ == '__main__':
    main()
