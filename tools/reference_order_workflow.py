"""One business workflow: accepted rules, correction, submission, recovery review, reports."""
import csv
import hashlib
import json
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request

from reference_non_screen import verify_batch_crash


def verify_order_workflow(root, output, binary, head, digest, dsn_file, sql, run, container):
    workspace = Path(tempfile.mkdtemp(prefix='order-workflow-', dir=output)) / 'workspace'
    shutil.copytree(root / 'reference/workspace', workspace)
    acceptance = output / 'shared-acceptance'
    run(['go', 'build', '-o', str(acceptance), './tools/reference-shared-acceptance'], cwd=root)
    result = {'passed': False, 'workspace': str(workspace), 'stages': [], 'stateHistory': [], 'reports': [],
              'limits': ['scratch machine acceptance is not human or production authority',
                         'input corrections use diagnostic SQL, not a product editing API',
                         'no durable operation-key receipt or automatic resume',
                         'Oracle customer lookup and business identity remain synthetic',
                         'combined CLI uses explicit diagnostic Source loading; no authoritative Host Reader is connected']}
    processes, logs = [], []
    credential = output / 'workflow-editor.private'

    def save(name, value):
        text = json.dumps(value, ensure_ascii=False, indent=2) + '\n'
        assert 'editor-secret' not in text
        (output / (name + '.json')).write_text(text)

    def cli(action, *args, expected=0):
        proc = subprocess.run([str(acceptance), action, '-workspace', str(workspace), *args], cwd=root,
                              capture_output=True, text=True, timeout=60)
        assert proc.returncode == expected, action
        value = json.loads(proc.stdout)
        save('workflow-' + action + '-' + str(len(result['stages'])), value)
        return value

    def revision():
        return cli('revision')['modelRevision']

    def admit(model):
        assert model.startswith('sha256:') and len(model) == 71
        sql(f"UPDATE public.tei_reference_execution_grants SET model_revision='{model}', revoked=false, expires_at=clock_timestamp()+interval '10 minutes';")

    def common(model):
        return [str(binary), '-diagnostic-unaccepted-source', '-composition-scenarios', '-workspace', str(workspace), '-model-revision', model,
                '-postgres-dsn-file', str(dsn_file), '-postgres-fencing-token', '1', '-baseline-id', 'order-workflow',
                '-baseline-revision', head, '-target-revision', head]

    def start(model):
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        log = (output / ('workflow-host-' + str(len(processes)) + '.log')).open('w')
        logs.append(log)
        proc = subprocess.Popen(common(model) + ['-listen', f'127.0.0.1:{port}'], cwd=root, stdout=log, stderr=log)
        processes.append(proc)
        url = f'http://127.0.0.1:{port}'
        for _ in range(100):
            try:
                urllib.request.urlopen(url + '/healthz', timeout=1).close()
                return url
            except OSError:
                if proc.poll() is not None:
                    raise RuntimeError('Workflow host exited before readiness')
                time.sleep(.05)
        raise RuntimeError('Workflow host readiness timed out')

    def request(url, path):
        try:
            with urllib.request.urlopen(url + path, timeout=15) as response:
                value = json.load(response)
                return response.status, value.get('result', value)
        except urllib.error.HTTPError as error:
            return error.code, None

    def states():
        return sql('SELECT order_id,status FROM public.orders ORDER BY order_id;')

    def confirm_states(stage, expected):
        actual = states()
        assert actual == expected, stage
        result['stateHistory'].append({'stage': stage, 'states': actual.splitlines()})
        return actual

    try:
        # One initial data preparation, before the business workflow begins.
        # The three orders are never reset to draft between later stages.
        sql("UPDATE public.order_lines SET unit_price=CASE order_line_id WHEN 5001 THEN 4000 WHEN 5002 THEN 4500 WHEN 5003 THEN 8300 ELSE 0 END;")
        initial = confirm_states('initial', '1001|draft\n1002|draft\n1003|draft')
        r0 = revision()
        candidate = json.loads((root / 'reference/order-workflow/change-candidate.json').read_text())
        candidate['baseModelRevision'] = r0
        candidate_path = output / 'workflow-candidate.json'
        candidate_path.write_text(json.dumps(candidate, ensure_ascii=False, indent=2) + '\n')
        preview = cli('preview', '-candidate', str(candidate_path))
        assert preview['acceptable'] and revision() == r0 and states() == initial
        admit(r0)
        old = start(r0)
        read_path = '/scenario/read?from=2026-09-18T00%3A00%3A00Z&row=0'
        assert request(old, read_path)[0] == 200
        refused = subprocess.run(common(preview['modelRevisionAfter']) + ['-listen', '127.0.0.1:0'], cwd=root,
                                 capture_output=True, text=True, timeout=20)
        assert refused.returncode != 0
        review = cli('review', '-candidate', str(candidate_path), '-format', 'json')
        wrong = cli('accept', '-candidate', str(candidate_path), '-expected-review-fingerprint', 'wrong', expected=1)
        assert wrong['decision'] == 'refused' and revision() == r0
        accepted = cli('accept', '-candidate', str(candidate_path), '-expected-review-fingerprint', review['reviewFingerprint'])
        assert accepted['decision'] == 'accepted'
        r1 = accepted['modelRevisionAfter']
        assert r1 == preview['modelRevisionAfter'] == revision()
        confirm_states('accepted-rule', initial)
        result['acceptedModelRevision'] = r1
        result['stages'].append('review and accept positive unit-price rule without changing order state')
        assert request(old, read_path)[0] >= 400
        current = start(r1)
        assert request(current, read_path)[0] >= 400
        admit(r1)
        status, manifest = request(current, '/manifest.json')
        assert status == 200 and manifest['modelRevision'] == r1 and len(manifest['validations']) == 4
        result['stages'].append('reload accepted model and separately admit its database execution')
        credential.write_text('editor-secret\n')
        credential.chmod(0o600)
        host = common(r1) + ['-from', '2026-09-18T00:00:00Z']
        batch = ['-non-screen', 'batch', '-credential-file', str(credential)]

        def invoke(name, args, success=True):
            proc = subprocess.run(host + args, cwd=root, capture_output=True, text=True, timeout=30)
            assert 'editor-secret' not in proc.stdout + proc.stderr
            (output / (name + '.stderr.log')).write_text(proc.stderr)
            assert (proc.returncode == 0) == success, name
            value = json.loads(proc.stdout) if proc.stdout.strip() else None
            if value is not None:
                save(name, value)
                trace = value['current']['composition'] if 'current' in value else value['composition']
                assert trace['modelRevision'] == r1 and trace['artifactDigest'] == digest
            return value

        journal1 = output / 'workflow-batch-1.jsonl'
        first = invoke('workflow-first-batch', batch + ['-rows', '0,1,2', '-batch-journal', str(journal1)], False)
        assert first['stage'] == 'business-stopped' and first['planned'] == [1001, 1002, 1003]
        assert len(first['items']) == 2 and first['items'][1]['result']['stage'] == 'validation-not-satisfied'
        confirm_states('quantity-stop', '1001|submitted\n1002|draft\n1003|draft')
        diagnosis = invoke('workflow-first-review', ['-non-screen', 'batch-review', '-batch-journal-in', str(journal1)])
        assert diagnosis['modelRevisionMatches'] is True
        result['stages'].append('submit first order; stop on invalid quantity; diagnose the same model and remaining orders')
        sql('UPDATE public.order_lines SET quantity=1 WHERE order_line_id=5003;')
        journal2 = output / 'workflow-batch-2.jsonl'
        second = invoke('workflow-second-batch', batch + ['-rows', '1,2', '-batch-journal', str(journal2)], False)
        assert second['stage'] == 'business-stopped' and len(second['items']) == 2
        validations = second['items'][1]['result']['validations']
        assert any(v['validation'] == 'sales.order_line.unit_price_positive' and v['result'] != 'satisfied' for v in validations)
        confirm_states('price-stop', '1001|submitted\n1002|submitted\n1003|draft')
        result['stages'].append('correct quantity and explicitly submit remainder; accepted price rule stops the third order')
        sql('UPDATE public.order_lines SET unit_price=6200 WHERE order_line_id=5004;')
        result['crash'] = verify_batch_crash(root, output, host, batch, sql, invoke, container, digest,
                                           reset=False, rows='2', planned=(1003,))
        confirm_states('crash-reviewed', '1001|submitted\n1002|submitted\n1003|draft')
        result['stages'].append('correct price; crash before last mutation; review log without resetting prior commits')
        last = invoke('workflow-final-submit', batch + ['-rows', '2', '-batch-journal', str(output / 'workflow-batch-3.jsonl')])
        assert last['stage'] == 'completed'
        final_states = confirm_states('completed', '1001|submitted\n1002|submitted\n1003|submitted')
        repeated = invoke('workflow-repeat-refused', batch + ['-rows', '0,1,2'], False)
        assert repeated['stage'] == 'execution-failed'
        assert states() == final_states
        result['stages'].append('explicitly finish last order; reject replay of already committed orders')
        expected_rows = [['1001', '1', 'ITEM-A', '2', '4000'], ['1001', '2', 'ITEM-B', '1', '4500'],
                         ['1002', '1', 'ITEM-C', '1', '8300'], ['1003', '1', 'ITEM-D', '1', '6200']]
        for format in ('csv', 'json'):
            path = output / ('workflow-completed-orders.' + format)
            receipt = invoke('workflow-final-' + format, ['-non-screen', 'report', '-rows', '0,1,2',
                             '-report-format', format, '-artifact-out', str(path)])
            assert receipt['rows'] == 4 and receipt['digest'] == 'sha256:' + hashlib.sha256(path.read_bytes()).hexdigest()
            if format == 'csv':
                with path.open(newline='') as document:
                    parsed = list(csv.reader(document))
                assert parsed == [['order_id', 'line_number', 'item_code', 'quantity', 'unit_price'], *expected_rows]
            else:
                keys = ('orderId', 'lineNumber', 'itemCode', 'quantity', 'unitPrice')
                assert json.loads(path.read_text()) == [dict(zip(keys, row)) for row in expected_rows]
            result['reports'].append({'format': format, 'path': str(path), 'digest': receipt['digest'], 'rows': 4})
        stale_journal = output / 'workflow-old-model.jsonl'
        stale_journal.write_text(journal1.read_text().replace(r1, r0))
        mismatch = invoke('workflow-old-model-review', ['-non-screen', 'batch-review', '-batch-journal-in', str(stale_journal)])
        assert mismatch['modelRevisionMatches'] is False
        confirm_states('reports-and-mismatch-review', final_states)
        assert revision() == r1
        result['finalStates'] = final_states.splitlines()
        result['stages'].append('publish final CSV and JSON under the same accepted model; expose old-model log mismatch')
        result['passed'] = True
        return result
    finally:
        for process in processes:
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill(); process.wait(timeout=5)
        for log in logs:
            log.close()
        credential.unlink(missing_ok=True)
        save('order-workflow', result)
