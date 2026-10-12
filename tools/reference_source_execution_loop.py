"""Scratch-only Shared Acceptance -> exact host model -> PostgreSQL execution."""
import json
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request


def verify_source_loop(root, output, binary, head, digest, dsn_file, sql, run):
    scratch = Path(tempfile.mkdtemp(prefix='source-loop-', dir=output)) / 'workspace'
    shutil.copytree(root / 'reference/workspace', scratch)
    acceptance = output / 'shared-acceptance'
    run(['go', 'build', '-o', str(acceptance), './tools/reference-shared-acceptance'], cwd=root)
    evidence = {'passed': False, 'checks': [], 'scratchWorkspace': str(scratch),
                'decisionAuthorityStatus': 'unresolved-reference-only', 'activation': 'explicit-host-restart',
                'limits': ['no change to reference/workspace', 'scratch acceptance is not human approval',
                           'no atomic coordination with arbitrary filesystem writers', 'no live model replacement']}
    processes = []
    logs = []

    def cli(command, *args, expected=0):
        result = subprocess.run([str(acceptance), command, '-workspace', str(scratch), *args], cwd=root,
                                capture_output=True, text=True, timeout=120)
        assert result.returncode == expected, result.stderr or result.stdout
        return json.loads(result.stdout)

    def revision():
        return cli('revision')['modelRevision']

    def request(url, path, payload=None):
        data = None if payload is None else json.dumps(payload).encode()
        req = urllib.request.Request(url + path, data, headers={'Content-Type': 'application/json'})
        try:
            with urllib.request.urlopen(req, timeout=20) as response:
                value = json.load(response)
                return response.status, value.get('result', value)
        except urllib.error.HTTPError as error:
            return error.code, error.read().decode()

    def host_args(model_revision, address):
        return [str(binary), '-workspace', str(scratch), '-composition-scenarios', '-model-revision', model_revision,
                '-postgres-dsn-file', str(dsn_file), '-postgres-fencing-token', '1', '-baseline-id', 'B',
                '-baseline-revision', 'e5231839a5dcc62cce1181a2731fcb7c70d16736', '-target-revision', head, '-listen', address]

    def start(model_revision, name):
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        log = (output / f'source-loop-{name}.log').open('w'); logs.append(log)
        process = subprocess.Popen(host_args(model_revision, f'127.0.0.1:{port}'), cwd=root, stdout=log, stderr=log)
        processes.append(process)
        url = f'http://127.0.0.1:{port}'
        for _ in range(100):
            try:
                urllib.request.urlopen(url + '/healthz', timeout=1).close()
                return url
            except OSError:
                if process.poll() is not None:
                    raise RuntimeError('Source-bound host exited; inspect source-loop log')
                time.sleep(.1)
        raise RuntimeError('Source-bound host did not become ready')

    def admit(model_revision):
        # Only after explicit fixture selection or a successful Shared Acceptance
        # decision. The grants are runtime admission, not acceptance authority.
        assert model_revision.startswith('sha256:') and len(model_revision) == 71
        sql(f"UPDATE public.tei_reference_execution_grants SET model_revision='{model_revision}', revoked=false, expires_at=clock_timestamp()+interval '10 minutes';")

    payload = {'loginId': 'legacy-order-editor', 'credential': 'editor-secret', 'from': '2026-09-18T08:30:00Z', 'row': 0}
    read_path = '/scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0'
    try:
        r0 = revision(); evidence['baseModelRevision'] = r0
        source_before = {str(p.relative_to(scratch)): p.read_bytes() for p in scratch.rglob('*') if p.is_file()}
        candidate = json.loads((root / 'reference/source-execution-loop/candidate.json').read_text())
        candidate['baseModelRevision'] = r0
        candidate_path = output / 'source-loop-candidate.json'
        candidate_path.write_text(json.dumps(candidate, ensure_ascii=False, indent=2) + '\n')
        preview = cli('preview', '-candidate', str(candidate_path))
        assert preview['acceptable'] and preview['modelRevisionAfter'] != r0 and revision() == r0
        assert source_before == {str(p.relative_to(scratch)): p.read_bytes() for p in scratch.rglob('*') if p.is_file()}
        (output / 'source-loop-preview.json').write_text(json.dumps(preview, ensure_ascii=False, indent=2) + '\n')
        admit(r0)
        sql("UPDATE public.orders SET status='draft'; UPDATE public.order_lines SET unit_price=0 WHERE order_line_id=5001;")
        old_url = start(r0, 'r0')
        status, before = request(old_url, '/scenario/submit', payload)
        assert status == 200 and before['stage'] == 'completed' and len(before['validations']) == 3
        assert before['composition']['modelRevision'] == r0 and before['read']['composition']['modelRevision'] == r0
        evidence['checks'].append('preview is read-only; unaccepted unit-price rule does not affect R0 execution')
        sql("UPDATE public.orders SET status='draft' WHERE order_id=1001;")
        unaccepted = subprocess.run(host_args(preview['modelRevisionAfter'], '127.0.0.1:0'), cwd=root,
                                    capture_output=True, text=True, timeout=20)
        assert unaccepted.returncode != 0 and 'revision' in unaccepted.stderr
        evidence['checks'].append('host cannot start a proposed revision against unchanged source')
        review = cli('review', '-candidate', str(candidate_path), '-format', 'json')
        (output / 'source-loop-review.json').write_text(json.dumps(review, ensure_ascii=False, indent=2) + '\n')
        refused = cli('accept', '-candidate', str(candidate_path), '-expected-review-fingerprint', 'wrong-review', expected=1)
        assert refused['decision'] == 'refused' and not refused['sourceChanged'] and revision() == r0
        accepted = cli('accept', '-candidate', str(candidate_path), '-expected-review-fingerprint', review['reviewFingerprint'])
        assert accepted['decision'] == 'accepted' and accepted['sourceChanged']
        r1 = accepted['modelRevisionAfter']; assert revision() == r1 == preview['modelRevisionAfter']
        assert accepted['modelRevisionBefore'] == r0 and accepted['reviewBinding']['fingerprint'] == review['reviewFingerprint']
        evidence['acceptedModelRevision'] = r1
        evidence['candidateFingerprint'] = accepted['candidateFingerprint']
        evidence['reviewFingerprint'] = review['reviewFingerprint']
        (output / 'source-loop-accepted.json').write_text(json.dumps(accepted, ensure_ascii=False, indent=2) + '\n')
        evidence['checks'].append('wrong review preserves R0; exact review accepts rule and Application selection together as R1')
        stale = cli('accept', '-candidate', str(candidate_path), '-expected-review-fingerprint', review['reviewFingerprint'], expected=1)
        assert stale['decision'] == 'refused' and not stale['sourceChanged'] and revision() == r1
        status, _ = request(old_url, '/scenario/submit', payload)
        assert status >= 400 and sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'draft'
        evidence['checks'].append('stale proposal preserves R1 and old R0 host refuses new effects')
        # Reloaded source alone does not grant runtime permission for R1.
        new_url = start(r1, 'r1')
        status, _ = request(new_url, read_path)
        assert status >= 400
        admit(r1)
        status, manifest = request(new_url, '/manifest.json')
        assert status == 200 and manifest['modelRevision'] == r1 and len(manifest['validations']) == 4
        evidence['checks'].append('R1 reload needs separate database host admission; projection and execution use R1')
        status, blocked = request(new_url, '/scenario/submit', payload)
        assert status == 200 and blocked['stage'] == 'validation-not-satisfied' and len(blocked['validations']) == 4
        assert blocked['validations'][-1]['validation'] == 'sales.order_line.unit_price_positive'
        assert blocked['validations'][-1]['result'] != 'satisfied' and 'command' not in blocked and 'observation' not in blocked
        assert blocked['composition']['modelRevision'] == r1 and sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'draft'
        sql('UPDATE public.order_lines SET unit_price=4000.00 WHERE order_line_id=5001;')
        status, after = request(new_url, '/scenario/submit', payload)
        assert status == 200 and after['stage'] == 'completed' and len(after['validations']) == 4
        assert after['composition']['modelRevision'] == r1 and after['read']['composition']['modelRevision'] == r1
        assert after['observation']['nature'] == 'external-observed' and after['observation']['result'] == 'submitted'
        assert sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'submitted'
        assert 'editor-secret' not in json.dumps(after)
        (output / 'source-loop-execution.json').write_text(json.dumps({'before': before, 'blocked': blocked, 'after': after}, indent=2) + '\n')
        evidence['checks'].append('accepted rule blocks zero price without an effect; valid price commits and is separately observed under R1')
        sql("UPDATE public.orders SET status='draft' WHERE order_id=1001;")
        source = scratch / 'semantic/sales/validation.tei'
        source.write_text(source.read_text() + '\n// unaccepted edit after R1 selection\n')
        status, _ = request(new_url, '/scenario/submit', payload)
        assert status >= 400 and sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'draft'
        evidence['checks'].append('direct unaccepted source edits invalidate R1 host without changing persistent state')
        evidence['passed'] = True
        return evidence
    finally:
        for process in processes:
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill(); process.wait()
        for log in logs:
            log.close()
        # Restore the ordinary fixture admission before its DB-loss check.
        sql("UPDATE public.tei_reference_execution_grants SET model_revision='', revoked=false, expires_at=clock_timestamp()+interval '10 minutes';")
        (output / 'source-loop-evidence.json').write_text(json.dumps(evidence, indent=2) + '\n')
