"""Real process/SQL checks for the bounded non-screen reference host."""
import csv
import hashlib
import json
import subprocess
import time


def verify_non_screen(root, output, binary, head, digest, dsn_file, sql, workspace='reference/workspace', owned_container=None):
    checks = []
    common = [str(binary), '-composition-scenarios', '-postgres-dsn-file', str(dsn_file),
              '-postgres-fencing-token', '1', '-workspace', workspace,
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
    assert receipt['limits'] == {'maxRows': 10000, 'maxBytes': 8 << 20}
    for flag, value in [('-report-max-rows', '1'), ('-report-max-bytes', '1')]:
        bounded_path = output / ('bounded-' + flag.removeprefix('-') + '.csv')
        assert invoke('report-' + flag.removeprefix('-'), ['-non-screen', 'report', '-rows', '0,1,2',
                      flag, value, '-artifact-out', str(bounded_path)], False) is None
        assert not bounded_path.exists()
    checks.append('host report row and encoded-byte budgets refuse oversized artifacts before publication')
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
        journal_path = output / 'batch-progress.jsonl'
        receipt = invoke('batch-partial', batch + ['-rows', '0,1,2', '-batch-journal', str(journal_path)], False)
        assert receipt['stage'] == 'business-stopped' and receipt['planned'] == [1001, 1002, 1003]
        assert len(receipt['items']) == 2 and receipt['items'][0]['result']['observation']['nature'] == 'external-observed'
        assert sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'submitted'
        assert sql('SELECT status FROM public.orders WHERE order_id=1003;') == 'draft'
        journal_raw = journal_path.read_text()
        assert 'editor-secret' not in journal_raw
        events = [json.loads(line) for line in journal_raw.splitlines()]
        assert [e['kind'] for e in events] == ['planned', 'starting', 'result', 'starting', 'result', 'ended']
        assert [e['sequence'] for e in events] == list(range(1, 7))
        assert events[0]['planned'] == receipt['planned'] and events[0]['composition']['artifactDigest'] == digest
        assert events[-1]['receipt'] == receipt
        before_journal_refusal = sql('SELECT order_id,status,xmin::text FROM public.orders ORDER BY order_id;')
        assert invoke('batch-journal-existing', batch + ['-rows', '2', '-batch-journal', str(journal_path)], False) is None
        assert journal_path.read_text() == journal_raw
        assert sql('SELECT order_id,status,xmin::text FROM public.orders ORDER BY order_id;') == before_journal_refusal
        checks.append('synchronized progress journal matches receipt; no credentials; existing journal refusal before effects')
        review = invoke('batch-review-partial', ['-non-screen', 'batch-review', '-batch-journal-in', str(journal_path)])
        assert review['ended'] and not review['incompleteTail']
        assert [i['recorded'] for i in review['items']] == ['result-recorded', 'result-recorded', 'not-recorded-as-started']
        assert [i['observation']['result'] for i in review['current']['items']] == ['submitted', 'draft', 'draft']
        assert review['journalDigest'] == 'sha256:' + hashlib.sha256(journal_path.read_bytes()).hexdigest()
        bad_journal = output / 'invalid-journal.jsonl'
        bad_journal.write_text(journal_raw.replace('"sequence":2', '"sequence":9', 1))
        assert invoke('batch-review-invalid', ['-non-screen', 'batch-review', '-batch-journal-in', str(bad_journal)], False) is None
        assert sql('SELECT order_id,status,xmin::text FROM public.orders ORDER BY order_id;') == before_journal_refusal
        checks.append('journal-driven read-only review; invalid sequence refusal; recorded progress remains separate from current database state')
        before = sql('SELECT order_id, status, xmin::text FROM public.orders ORDER BY order_id;')
        observe_args = ['-non-screen', 'batch-observe', '-order-ids', '1003,1001,1002', '-rows', '999']
        sql("UPDATE public.tei_reference_execution_grants SET revoked=true WHERE provider_id IN ('order-submit','order-batch');")
        observed = invoke('batch-observe-partial', observe_args)
        assert observed['composition']['artifactDigest'] == digest
        assert observed['composition']['contractId'] == 'tei.reference.order-entry.batch-observe'
        assert [item['orderId'] for item in observed['items']] == [1003, 1001, 1002]
        assert [item['observation']['result'] for item in observed['items']] == ['draft', 'submitted', 'draft']
        assert all(item['observation']['nature'] == 'external-observed' for item in observed['items'])
        assert sql('SELECT order_id, status, xmin::text FROM public.orders ORDER BY order_id;') == before
        assert invoke('batch-observe-duplicate', ['-non-screen', 'batch-observe', '-order-ids', '1001,1001'], False) is None
        assert invoke('batch-observe-missing', ['-non-screen', 'batch-observe', '-order-ids', '1001,9999'], False) is None
        sql("UPDATE public.tei_reference_execution_grants SET revoked=true WHERE provider_id='order-batch-observe';")
        assert invoke('batch-observe-revoked', observe_args, False) is None
        sql("UPDATE public.tei_reference_execution_grants SET revoked=false WHERE provider_id IN ('order-submit','order-batch','order-batch-observe');")
        checks.append('read-only observation by exact IDs after partial success; independent SQL row versions; separate observation grant; no partial output on refusal')
        receipt = invoke('batch-replay', batch + ['-rows', '0,1,2'], False)
        assert receipt['stage'] == 'execution-failed' and len(receipt['items']) == 1
        assert sql('SELECT status FROM public.orders WHERE order_id=1003;') == 'draft'
        receipt = invoke('batch-remainder', batch + ['-rows', '2'])
        assert receipt['stage'] == 'completed' and sql('SELECT status FROM public.orders WHERE order_id=1003;') == 'submitted'
        checks.append('real CLI partial success, business failure, replay rejection and explicit remainder; independent SQL')
        crash = None
        if owned_container is not None:
            crash = verify_batch_crash(root, output, common, batch, sql, invoke, owned_container, digest)
            checks.append('real CLI force-killed during blocked second transaction; saved first commit and uncertain start; fresh external observation without retry')
    finally:
        private.unlink(missing_ok=True)
        sql("UPDATE public.orders SET status='draft'; UPDATE public.tei_reference_execution_grants SET revoked=false, model_revision='';")
    return {'passed': True, 'checks': checks, 'cancellationEvidence': 'deterministic real-database Go test cancels after first observed commit',
            'crashRecovery': crash,
            'limits': ['no durable request-key deduplication or automatic resume', 'no PDF, browser or human-value validation']}


def verify_batch_crash(root, output, common, batch, sql, invoke, container, digest):
    """The container, blocker session and CLI are all owned by this verifier."""
    sql("UPDATE public.orders SET status='draft';")
    journal = output / 'crashed-batch-progress.jsonl'
    blocker = host = None
    blocker_pid = None
    with (output / 'crash-blocker.log').open('w') as blocker_log, \
            (output / 'crash-host.stdout.log').open('w') as host_out, \
            (output / 'crash-host.stderr.log').open('w') as host_err:
        try:
            # Wait on server state, not psql output buffering. The owned session
            # has locked only 1003 when it enters pg_sleep; 1001 can commit.
            blocker = subprocess.Popen(['docker', 'exec', '-e', 'PGAPPNAME=tei_batch_crash_blocker', container,
                                        'psql', '-X', '-U', 'postgres', '-d', 'tei_reference', '-v', 'ON_ERROR_STOP=1',
                                        '-c', 'BEGIN; SELECT order_id FROM public.orders WHERE order_id=1003 FOR UPDATE; SELECT pg_sleep(30); ROLLBACK;'],
                                       stdout=blocker_log, stderr=blocker_log)
            deadline = time.monotonic() + 8
            while time.monotonic() < deadline:
                active = sql("SELECT pid FROM pg_stat_activity WHERE application_name='tei_batch_crash_blocker' AND wait_event='PgSleep';")
                if active:
                    blocker_pid = int(active)
                    break
                if blocker.poll() is not None:
                    raise RuntimeError('Owned database blocker exited before locking the row')
                time.sleep(.05)
            else:
                raise RuntimeError('Owned row blocker was not ready')
            host = subprocess.Popen(common + batch + ['-rows', '0,2', '-batch-journal', str(journal)],
                                    cwd=root, stdout=host_out, stderr=host_err)
            deadline = time.monotonic() + 8
            while time.monotonic() < deadline:
                if host.poll() is not None:
                    raise RuntimeError('Batch host exited before the intended crash boundary')
                events = []
                if journal.exists():
                    # A partially written final line is never a successful event.
                    for line in journal.read_bytes().splitlines(keepends=True):
                        if line.endswith(b'\n'):
                            events.append(json.loads(line))
                started = len(events) == 4 and events[-1].get('kind') == 'starting' and events[-1].get('orderId') == 1003
                blocked = sql(f'SELECT count(*) FROM pg_stat_activity WHERE {blocker_pid} = ANY(pg_blocking_pids(pid));')
                if started and int(blocked) > 0:
                    break
                time.sleep(.05)
            else:
                raise RuntimeError('Batch did not reach the blocked second transaction')
            host.kill()
            host.wait(timeout=5)
            assert host.returncode != 0
            # Only terminate the exact owned blocker in the disposable DB.
            sql(f"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid={blocker_pid} AND application_name='tei_batch_crash_blocker';")
            blocker.wait(timeout=5)
            events = [json.loads(line) for line in journal.read_text().splitlines()]
            assert [e['kind'] for e in events] == ['planned', 'starting', 'result', 'starting']
            assert [e['sequence'] for e in events] == [1, 2, 3, 4]
            assert events[0]['planned'] == [1001, 1003] and events[0]['composition']['artifactDigest'] == digest
            assert events[2]['item']['orderId'] == 1001 and events[2]['item']['result']['stage'] == 'completed'
            assert sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'submitted'
            assert sql('SELECT status FROM public.orders WHERE order_id=1003;') == 'draft'
            observed = invoke('batch-observe-after-crash', ['-non-screen', 'batch-observe', '-order-ids', '1001,1003'])
            assert [i['observation']['result'] for i in observed['items']] == ['submitted', 'draft']
            assert all(i['observation']['nature'] == 'external-observed' for i in observed['items'])
            reviewed = invoke('batch-review-after-crash', ['-non-screen', 'batch-review', '-batch-journal-in', str(journal)])
            assert not reviewed['ended'] and not reviewed['incompleteTail']
            assert [i['recorded'] for i in reviewed['items']] == ['result-recorded', 'started-without-result']
            assert [i['observation']['result'] for i in reviewed['current']['items']] == ['submitted', 'draft']
            assert 'editor-secret' not in journal.read_text() + (output / 'crash-host.stderr.log').read_text()
            return {'passed': True, 'boundary': 'first commit observed; second transaction blocked before mutation',
                    'journalKinds': [e['kind'] for e in events], 'observedStates': ['submitted', 'draft'],
                    'limits': ['starting alone does not prove rollback', 'no lost commit acknowledgement or power-loss proof', 'no automatic retry or resume']}
        finally:
            if host is not None and host.poll() is None:
                host.kill()
                host.wait(timeout=5)
            if blocker_pid is not None and blocker is not None and blocker.poll() is None:
                sql(f"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid={blocker_pid} AND application_name='tei_batch_crash_blocker';")
            if blocker is not None:
                try:
                    blocker.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    blocker.kill()
                    blocker.wait(timeout=5)
