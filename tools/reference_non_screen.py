"""Real process/SQL checks for the bounded non-screen reference host."""
import csv
import hashlib
import json
import subprocess


def verify_non_screen(root, output, binary, head, digest, dsn_file, sql):
    checks = []
    common = [str(binary), '-composition-scenarios', '-postgres-dsn-file', str(dsn_file),
              '-postgres-fencing-token', '1', '-workspace', 'reference/workspace',
              '-baseline-id', 'B', '-baseline-revision', 'e5231839a5dcc62cce1181a2731fcb7c70d16736',
              '-target-revision', head, '-from', '2026-09-18T00:00:00Z']

    def invoke(name, args, success=True):
        p = subprocess.run(common + args, cwd=root, capture_output=True, text=True, timeout=30)
        (output / (name + '.stdout.json')).write_text(p.stdout)
        (output / (name + '.stderr.log')).write_text(p.stderr)
        assert 'editor-secret' not in p.stdout + p.stderr
        assert (p.returncode == 0) == success, name
        return json.loads(p.stdout) if p.stdout.strip() else None

    sql("UPDATE public.orders SET status='draft'; UPDATE public.tei_reference_execution_grants SET model_revision='', revoked=false;")
    csv_path = output / 'orders.csv'
    receipt = invoke('report-csv', ['-non-screen', 'report', '-rows', '0,1,2', '-artifact-out', str(csv_path)])
    assert receipt['rows'] == 4 and receipt['composition']['artifactDigest'] == digest
    assert receipt['renderer']['providerId'] == 'report-csv'
    assert receipt['digest'] == 'sha256:' + hashlib.sha256(csv_path.read_bytes()).hexdigest()
    with csv_path.open(newline='') as f:
        rows = list(csv.reader(f))
    assert len(rows) == 5 and rows[1][:3] == ['1001', '1', 'ITEM-A']
    json_path = output / 'orders.json'
    receipt = invoke('report-json', ['-non-screen', 'report', '-rows', '0,1,2', '-report-format', 'json', '-artifact-out', str(json_path)])
    assert receipt['renderer']['providerId'] == 'report-json'
    assert len(json.loads(json_path.read_text())) == 4
    checks.append('real CLI CSV/JSON renderer selection; actual document digest and independent parsed rows')
    invoke('report-overwrite', ['-non-screen', 'report', '-artifact-out', str(csv_path)], False)
    sql("UPDATE public.tei_reference_execution_grants SET revoked=true WHERE provider_id='order-report';")
    denied = output / 'denied.csv'
    invoke('report-revoked', ['-non-screen', 'report', '-artifact-out', str(denied)], False)
    assert not denied.exists()
    checks.append('exclusive artifact creation and separate report execution revocation')
    private = output / 'editor.private'
    private.write_text('editor-secret\n')
    private.chmod(0o600)
    try:
        batch = ['-non-screen', 'batch', '-credential-file', str(private)]
        receipt = invoke('batch-partial', batch + ['-rows', '0,1,2'], False)
        assert receipt['stage'] == 'business-stopped' and receipt['planned'] == [1001, 1002, 1003]
        assert len(receipt['items']) == 2 and receipt['items'][0]['result']['observation']['nature'] == 'external-observed'
        assert sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'submitted'
        assert sql('SELECT status FROM public.orders WHERE order_id=1003;') == 'draft'
        receipt = invoke('batch-replay', batch + ['-rows', '0,1,2'], False)
        assert receipt['stage'] == 'execution-failed' and len(receipt['items']) == 1
        assert sql('SELECT status FROM public.orders WHERE order_id=1003;') == 'draft'
        receipt = invoke('batch-remainder', batch + ['-rows', '2'])
        assert receipt['stage'] == 'completed' and sql('SELECT status FROM public.orders WHERE order_id=1003;') == 'submitted'
        checks.append('real CLI partial success, business failure, replay rejection and explicit remainder; independent SQL')
    finally:
        private.unlink(missing_ok=True)
        sql("UPDATE public.orders SET status='draft'; UPDATE public.tei_reference_execution_grants SET revoked=false, model_revision='';")
    return {'passed': True, 'checks': checks, 'cancellationEvidence': 'deterministic real-database Go test cancels after first observed commit',
            'limits': ['no durable request-key deduplication or automatic resume', 'no PDF, browser or human-value validation']}
