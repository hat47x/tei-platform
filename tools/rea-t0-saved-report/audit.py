#!/usr/bin/env python3
"""Read-only consumer audit of REA's *saved* CLI reports, not a REA runner.

Never concludes that a parsed report is complete or that Source/Provider/Host
was qualified. Intended only for the tiny, public T0 fixture.
"""
import argparse
import hashlib
import json
from pathlib import Path

MAX_REPORT = 16 * 1024 * 1024


def _read(root, name, limit):
    path = root / name
    if path.is_symlink():
        raise ValueError(f"symlink report rejected: {name}")
    if not path.is_file():
        return None
    if path.stat().st_size > limit:
        raise ValueError(f"report too large for T0 audit: {name}")
    return path.read_bytes()


def audit(root):
    root = Path(root)
    if root.is_symlink() or not root.is_dir():
        raise ValueError('report directory must be an actual directory, not a symlink')
    raw_exit = _read(root, 'exit-code.txt', 32)
    if raw_exit is None:
        raise ValueError('missing exit-code.txt; invocation status unknown')
    try:
        code = int(raw_exit.decode('ascii').strip())
        if not 0 <= code <= 255 or raw_exit.decode('ascii').strip() != str(code):
            raise ValueError('noncanonical exit code')
    except (UnicodeDecodeError, ValueError) as exc:
        raise ValueError('invalid exit-code.txt') from exc

    raw = _read(root, 'analysis.json', MAX_REPORT)
    stderr = _read(root, 'analysis.stderr', MAX_REPORT)
    result = {
        'auditVersion': 'rea-t0-saved-report/v1',
        'processExitCode': code,
        'processOutcome': 'exit-zero' if code == 0 else 'exit-nonzero',
        'stdoutBytes': len(raw) if raw is not None else None,
        'stdoutSHA256': hashlib.sha256(raw).hexdigest() if raw is not None else None,
        'stderrBytes': len(stderr) if stderr is not None else None,
        'stderrSHA256': hashlib.sha256(stderr).hexdigest() if stderr is not None else None,
        'reportShape': 'absent', 'reportedParseFailures': None,
        'reportedOK': None, 'processReportConsistency': 'not-observed',
        'analysisQualification': 'unknown',
        'externalSourceClosure': 'not-observed',
        'providerInvocation': 'not-observed',
        'consumerDelivery': 'local-file-read-only',
        'legacyRuntimeEffect': 'not-observed',
    }
    if raw is None or not raw.strip():
        result['reportShape'] = 'empty' if raw is not None else 'absent'
        return result
    try:
        obj = json.loads(raw)
    except (UnicodeDecodeError, json.JSONDecodeError):
        result['reportShape'] = 'invalid-json'
        return result
    result['reportShape'] = 'json-document' if isinstance(obj, dict) else 'other-json'
    if not isinstance(obj, dict):
        return result
    reported_ok = obj.get('ok')
    if type(reported_ok) is bool:
        result['reportedOK'] = reported_ok
        result['processReportConsistency'] = (
            'consistent' if reported_ok == (code == 0) else 'conflict'
        )
    if reported_ok is False:
        result['analysisQualification'] = 'reported-failure'
        return result
    stats = obj.get('data')
    stats = stats.get('normalized_result') if isinstance(stats, dict) else None
    stats = stats.get('statistics') if isinstance(stats, dict) else None
    count = stats.get('parse_failures') if isinstance(stats, dict) else None
    if isinstance(count, int) and not isinstance(count, bool) and count >= 0:
        result['reportedParseFailures'] = count
        if count > 0:
            result['analysisQualification'] = 'partial'
    return result


def jsonl_audit(path, *, object_collection):
    """Inspect derived JSONL framing; this never upgrades source coverage."""
    path = Path(path)
    if path.is_symlink() or path.stat().st_size > MAX_REPORT:
        raise ValueError('unsafe or oversized JSONL T0 view')
    try:
        lines = path.read_text(encoding='utf-8').splitlines()
    except UnicodeDecodeError:
        return {'framing': 'invalid-jsonl', 'rowsSeen': 0}
    records = []
    for line in lines:
        try:
            records.append(json.loads(line))
        except json.JSONDecodeError:
            return {'framing': 'invalid-jsonl', 'rowsSeen': len(records)}
    # REA error records are control records, never ordinary data rows.
    # A consumer must recognize them in either JSONL framing mode.
    controls = [
        (i, record.get('type'))
        for i, record in enumerate(records)
        if isinstance(record, dict) and record.get('$rea') == 'jsonl/v1'
    ]
    if any(kind == 'error' for _, kind in controls):
        return {'framing': 'reported-error', 'rowsSeen': len(records) - len(controls)}
    if not object_collection:
        return {'framing': 'bare-array-without-completeness-proof', 'rowsSeen': len(records)}
    if not records or not isinstance(records[-1], dict):
        return {'framing': 'missing-metadata', 'rowsSeen': len(records)}
    terminal = records[-1]
    if terminal.get('$rea') != 'jsonl/v1' or terminal.get('type') != 'metadata':
        return {'framing': 'missing-metadata', 'rowsSeen': len(records)}
    if len(controls) != 1 or controls[0] != (len(records) - 1, 'metadata'):
        return {'framing': 'extra-control-record', 'rowsSeen': len(records) - len(controls)}
    emitted = terminal.get('records', {}).get('emitted') if isinstance(terminal.get('records'), dict) else None
    if not isinstance(emitted, int) or isinstance(emitted, bool) or emitted != len(records) - 1:
        return {'framing': 'count-mismatch', 'rowsSeen': len(records) - 1}
    return {'framing': 'object-collection-framed', 'rowsSeen': len(records) - 1,
            'sourceCoverage': 'not-proven'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('report_dir')
    parser.add_argument('--jsonl-view', default=None)
    parser.add_argument('--jsonl-object-collection', action='store_true')
    args = parser.parse_args()
    out = audit(args.report_dir)
    if args.jsonl_view:
        out['derivedJsonl'] = jsonl_audit(args.jsonl_view, object_collection=args.jsonl_object_collection)
    print(json.dumps(out, sort_keys=True, ensure_ascii=False))


if __name__ == '__main__':
    main()
