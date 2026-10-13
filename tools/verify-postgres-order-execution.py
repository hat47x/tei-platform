#!/usr/bin/env python3
"""Verify an exact clean revision using an owned disposable PostgreSQL container."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess
import time
import urllib.error
import urllib.request


def run(args, **kwargs):
    result = subprocess.run(args, text=True, capture_output=True, timeout=600, **kwargs)
    if result.returncode:
        raise RuntimeError(f'{args[0]} failed with exit {result.returncode}: {result.stderr.strip()}')
    return result.stdout.strip()


def main():
    if not __debug__:
        raise RuntimeError('Evidence checks require Python without assertion optimization')
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', required=True, help='Evidence directory outside the repository')
    parser.add_argument('--source-loop', action='store_true', help='Also verify scratch Shared Acceptance -> selected model -> PostgreSQL execution')
    parser.add_argument('--non-screen', action='store_true', help='Also verify CLI reports and bounded batch processing')
    parser.add_argument('--frozen-api', action='store_true', help='Use the digest-bound frozen World JSON API with the same actual PostgreSQL checks')
    args = parser.parse_args()
    if args.frozen_api and (args.source_loop or args.non_screen):
        parser.error('--frozen-api does not support live Source changes or non-screen CLI execution')
    root = Path(__file__).resolve().parent.parent
    output = Path(args.output).resolve()
    if output == root or root in output.parents:
        raise RuntimeError('Evidence directory must be outside the repository')
    if run(['git', 'status', '--porcelain'], cwd=root):
        raise RuntimeError('A clean exact revision is required before creating the fixture')
    head = run(['git', 'rev-parse', 'HEAD'], cwd=root)
    output.mkdir(parents=True, exist_ok=True)
    binary = output / 'reference-web'
    workspace = 'reference/workspace'
    if args.frozen_api:
        frozen = output / 'frozen'
        run(['bash', 'tools/reference-first-web-two-unit-e1/build_frozen_json_api.sh', str(frozen),
             'B', 'e5231839a5dcc62cce1181a2731fcb7c70d16736'], cwd=root)
        binary = frozen / 'api/order-entry'
        workspace = str(frozen / 'api')
    else:
        run(['go', 'build', '-o', str(binary), './tools/reference-first-web-target'], cwd=root)
    digest = 'sha256:' + hashlib.sha256(binary.read_bytes()).hexdigest()
    container = None
    proc = None
    log = None
    report = {'candidateHead': head, 'artifactDigest': digest, 'evidenceNature': 'real-postgres-disposable-reference-data',
              'worldMode': 'digest-bound-frozen-world' if args.frozen_api else 'live-workspace',
              'passed': False, 'checks': [], 'cleanBefore': True, 'cleanAfter': False, 'fixtureCleanupConfirmed': False,
              'limits': ['Oracle customer and authentication remain synthetic', 'no production or browser evidence',
                         'single PostgreSQL host; coarse line table lock', 'no process hot replacement or availability SLO']}
    if args.frozen_api:
        report['frozenWorldSHA256'] = hashlib.sha256((Path(workspace) / 'frozen-world.json').read_bytes()).hexdigest()
        report['startupChecks'] = []
        # Backend selection must fail before World/credentials/connection reads.
        for backend in [[], ['-composition-scenarios'], ['-composition-scenarios', '-postgres-fencing-token', '-1']]:
            refused = subprocess.run([str(binary), '-postgres-dsn-file', '/missing/connection', *backend],
                                     cwd=root, capture_output=True, text=True, timeout=15)
            assert refused.returncode != 0 and 'PostgreSQL requires' in refused.stderr
            report['startupChecks'].append('incomplete backend rejected before data loading')
    try:
        container = run(['docker', 'run', '-d', '--label', 'tei.reference=postgres-order-execution',
                         '-e', 'POSTGRES_HOST_AUTH_METHOD=trust', '-e', 'POSTGRES_DB=tei_reference',
                         '-p', '127.0.0.1::5432', 'postgres:16-alpine'])
        for _ in range(100):
            # The image starts a socket-only temporary server during init. Check
            # TCP so fixture creation waits for the final server, not that seed.
            ready = subprocess.run(['docker', 'exec', container, 'pg_isready', '-h', '127.0.0.1', '-U', 'postgres', '-d', 'tei_reference'],
                                   capture_output=True, timeout=10)
            if ready.returncode == 0:
                break
            time.sleep(.1)
        else:
            raise RuntimeError('Disposable PostgreSQL did not become ready')

        def sql(source):
            return run(['docker', 'exec', '-i', container, 'psql', '-X', '-A', '-t', '-U', 'postgres', '-d', 'tei_reference',
                        '-v', 'ON_ERROR_STOP=1', '-v', f'artifact_digest={digest}'], input=source)

        sql((root / 'reference/postgres-order-execution/fixture.sql').read_text())
        port = int(run(['docker', 'port', container, '5432/tcp']).split(':')[-1])
        dsn_file = output / 'connection.private'
        dsn_file.write_text(f'host=127.0.0.1 port={port} user=postgres dbname=tei_reference sslmode=disable connect_timeout=5\n')
        dsn_file.chmod(0o600)
        env = os.environ.copy()
        env['TEI_REFERENCE_POSTGRES_TEST_DSN_FILE'] = str(dsn_file)
        test = subprocess.run(['go', 'test', '-race', '-count=1', '-v', '-run', 'TestPostgresScenario',
                               './tools/reference-first-web-target'], cwd=root, env=env,
                              capture_output=True, text=True, timeout=600)
        (output / 'actual-postgres-tests.log').write_text(test.stdout + test.stderr)
        if test.returncode != 0:
            raise RuntimeError('Actual PostgreSQL tests failed; see actual-postgres-tests.log')
        if '--- SKIP:' in test.stdout:
            raise RuntimeError('Actual PostgreSQL test was skipped')
        report['checks'].append('actual PostgreSQL regression tests with Race Detector and no skips')
        # Tests attest their own executable. Restore grants to the HTTP artifact.
        sql("UPDATE public.orders SET status='draft'; UPDATE public.order_lines SET quantity=CASE order_line_id WHEN 5001 THEN 2 WHEN 5003 THEN 0 ELSE 1 END; "
            "UPDATE public.tei_reference_execution_grants SET artifact_digest=:'artifact_digest', generation=1, fencing_token=1, revoked=false, expires_at=clock_timestamp()+interval '10 minutes';")
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            http_port = sock.getsockname()[1]
        url = f'http://127.0.0.1:{http_port}'
        log = (output / 'http-host.log').open('w')
        proc = subprocess.Popen([str(binary), '-composition-scenarios', '-postgres-dsn-file', str(dsn_file),
                                 '-postgres-fencing-token', '1', '-workspace', workspace, '-baseline-id', 'B',
                                 '-baseline-revision', 'e5231839a5dcc62cce1181a2731fcb7c70d16736',
                                 '-target-revision', head, '-listen', f'127.0.0.1:{http_port}'], cwd=root, stdout=log, stderr=log)
        for _ in range(100):
            try:
                urllib.request.urlopen(url + '/healthz', timeout=1).close()
                break
            except OSError:
                if proc.poll() is not None:
                    raise RuntimeError('HTTP host exited; see http-host.log')
                time.sleep(.1)
        else:
            raise RuntimeError('HTTP host startup timed out')

        def request(path, payload=None):
            body = None if payload is None else json.dumps(payload).encode()
            req = urllib.request.Request(url + path, body, headers={'Content-Type': 'application/json'})
            try:
                with urllib.request.urlopen(req, timeout=15) as response:
                    value = json.load(response)
                    return response.status, value.get('result', value)
            except urllib.error.HTTPError as error:
                return error.code, error.read().decode()

        read_path = '/scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0'
        status, read = request(read_path)
        assert status == 200 and read['storageRevision'] and len(read['orderLineRows']) == 2
        assert read['composition']['artifactDigest'] == digest
        assert {x['part']: x['nature'] for x in read['evidenceSources']}['oracle-customer'] == 'synthetic'
        report['checks'].append('HTTP orders and lines use a real PostgreSQL snapshot; Oracle remains explicitly synthetic')
        payload = {'loginId': 'legacy-order-editor', 'credential': 'editor-secret', 'from': '2026-09-18T08:30:00Z', 'row': 0}
        status, submit = request('/scenario/submit', payload)
        assert status == 200 and submit['stage'] == 'completed'
        assert submit['composition']['contractId'] == 'tei.reference.order-entry.postgres-submit'
        assert submit['composition']['artifactDigest'] == digest and submit['effectAuthorization']['outcome'] == 'allowed'
        assert submit['observation']['nature'] == 'external-observed' and submit['observation']['result'] == 'submitted'
        assert sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'submitted'
        assert 'editor-secret' not in json.dumps(submit)
        report['checks'].append('HTTP command, effect authorization, separate external observation and independent SQL state agree')
        status, _ = request('/scenario/submit', payload)
        assert status >= 400 and sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'submitted'
        report['checks'].append('repeated submission is rejected without a second transition')
        sql("UPDATE public.orders SET status='draft' WHERE order_id=1001; UPDATE public.tei_reference_execution_grants SET revoked=true WHERE provider_id='order-submit';")
        status, _ = request('/scenario/submit', payload)
        assert status >= 400 and sql('SELECT status FROM public.orders WHERE order_id=1001;') == 'draft'
        report['checks'].append('database-side execution revocation prevents mutation')
        if args.source_loop:
            import sys
            sys.dont_write_bytecode = True
            from reference_source_execution_loop import verify_source_loop
            report['sourceLoop'] = verify_source_loop(root, output, binary, head, digest, dsn_file, sql, run)
            assert report['sourceLoop']['passed']
        if args.non_screen:
            import sys
            sys.dont_write_bytecode = True
            from reference_non_screen import verify_non_screen
            report['nonScreen'] = verify_non_screen(root, output, binary, head, digest, dsn_file, sql)
            assert report['nonScreen']['passed']
        sql("UPDATE public.tei_reference_execution_grants SET revoked=false;")
        assert request(read_path)[0] == 200
        run(['docker', 'stop', container])
        status, _ = request(read_path)
        assert status >= 400
        report['checks'].append('database loss fails closed; no synthetic fallback')
        report['cleanAfter'] = not run(['git', 'status', '--porcelain'], cwd=root)
        assert report['cleanAfter']
    finally:
        if proc is not None:
            proc.terminate()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                proc.kill()
                proc.wait()
        if log is not None:
            log.close()
        if container is not None:
            run(['docker', 'rm', '-f', container])
            remaining = run(['docker', 'ps', '-a', '-q', '--no-trunc', '--filter', f'id={container}'])
            report['fixtureCleanupConfirmed'] = not remaining
            if remaining:
                raise RuntimeError('Owned fixture container cleanup not confirmed')
        private = output / 'connection.private'
        if private.exists():
            private.unlink()
        report['passed'] = report['cleanAfter'] and report['fixtureCleanupConfirmed'] and len(report['checks']) == 6 and (not args.source_loop or report.get('sourceLoop', {}).get('passed', False)) and (not args.non_screen or report.get('nonScreen', {}).get('passed', False))
        (output / 'evidence.json').write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
