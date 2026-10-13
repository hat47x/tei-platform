#!/usr/bin/env python3
"""Identify and run statically linked reference targets at one clean revision."""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess
import time
import urllib.error
import urllib.request


def main():
    if not __debug__:
        raise RuntimeError('Evidence requires enabled assertions')
    parser = argparse.ArgumentParser()
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parent.parent
    output = Path(args.output).resolve()
    if output.exists() or output == root or root in output.parents:
        raise RuntimeError('Use a new evidence directory outside the repository')
    env = dict(os.environ, GOENV='off', GOWORK='off', GOFLAGS='', CGO_ENABLED='0')

    def run(args):
        p = subprocess.run(args, cwd=root, env=env, capture_output=True, text=True, timeout=600)
        if p.returncode:
            raise RuntimeError(f'{args[0]} failed: {p.stderr}')
        return p.stdout.strip()

    assert not run(['git', 'status', '--porcelain'])
    head = run(['git', 'rev-parse', 'HEAD'])
    output.mkdir(parents=True)
    evidence = {'candidateHead': head, 'goProfile': json.loads(run(['go', 'env', '-json', 'GOVERSION', 'GOOS', 'GOARCH', 'GOAMD64', 'CGO_ENABLED'])),
                'passed': False, 'cleanBefore': True, 'targets': {},
                'limits': ['synthetic Oracle, authentication and order state', 'no production authority or real database evidence',
                           'static builds and process startup, not process hot replacement', 'no browser or user-value validation',
                           'combined CLI uses explicit diagnostic Source loading, without an authoritative Host Reader']}
    try:
        for name, tags in [('combined', ''), ('order', 'tei_order_only'), ('repair', 'tei_repair_only')]:
            binary = output / name
            tag_args = ['-tags=' + tags] if tags else []
            run(['go', 'build', '-trimpath', '-ldflags=-buildid=', *tag_args, '-o', str(binary), './tools/reference-first-web-target'])
            deps = run(['go', 'list', '-deps', *tag_args, './tools/reference-first-web-target'])
            nm = run(['go', 'tool', 'nm', str(binary)])
            (output / (name + '.deps.txt')).write_text(deps + '\n')
            (output / (name + '.nm.txt')).write_text(nm + '\n')
            evidence['targets'][name] = {'digest': 'sha256:' + hashlib.sha256(binary.read_bytes()).hexdigest(), 'bytes': binary.stat().st_size}
            if name == 'repair':
                for forbidden in ['tei/plugins/report-renderer', 'tei/tools/internal/postgresdb', 'tei/internal/driver/queryruntime/postgres', 'github.com/jackc/pgx/v5', 'tei/reference/web-composition/go-host']:
                    assert forbidden not in deps.splitlines(), forbidden
                for forbidden in ['newReferenceReadScenarioRunner', 'newReferenceCommandScenarioRunner', 'postgresScenarioState', 'wireNonScreen', 'handleProtectedSessionCommand']:
                    assert forbidden not in nm, forbidden
                assert 'composedCustomerLookup' in nm
            if name == 'order':
                assert 'composedCustomerLookup' not in nm and 'newReferenceCustomerLookupScenarioRunner' not in nm

        def host(name, app, composed):
            with socket.socket() as sock:
                sock.bind(('127.0.0.1', 0))
                port = sock.getsockname()[1]
            args = [str(output / name), '-application', app, '-workspace', 'reference/workspace', '-baseline-id', 'B',
                    '-baseline-revision', 'e5231839a5dcc62cce1181a2731fcb7c70d16736', '-target-revision', head,
                    '-listen', f'127.0.0.1:{port}']
            if composed:
                args += ['-composition-scenarios']
            if name == 'combined':
                args += ['-diagnostic-unaccepted-source']
            log = (output / f'{name}-{app}-{composed}.log').open('w')
            proc = subprocess.Popen(args, cwd=root, env=env, stdout=log, stderr=log)
            base = f'http://127.0.0.1:{port}'

            def request(path, payload=None):
                body = None if payload is None else json.dumps(payload).encode()
                req = urllib.request.Request(base + path, body, headers={'Content-Type': 'application/json'})
                try:
                    with urllib.request.urlopen(req, timeout=15) as response:
                        return response.status, json.load(response).get('result')
                except urllib.error.HTTPError as error:
                    return error.code, None

            try:
                for _ in range(100):
                    try:
                        urllib.request.urlopen(base + '/healthz', timeout=1).close()
                        break
                    except OSError:
                        if proc.poll() is not None:
                            raise RuntimeError('target exited during startup')
                        time.sleep(.1)
                else:
                    raise RuntimeError('target startup timeout')
                if app == 'repair_intake':
                    code, value = request('/scenario/customer-lookup?customerId=11')
                    assert code == 200 and len(value['customerRows']) == 1
                    assert request('/scenario/customer-lookup?customerId=invalid')[0] == 400
                    if name == 'repair':
                        assert request('/scenario/read')[0] == 404
                        assert request('/scenario/submit', {})[0] == 404
                    values = [value]
                    traces = [('repair-customer', 'tei.reference.repair-intake.customer-lookup')]
                else:
                    code, value = request('/scenario/read?from=2026-09-18T08:30:00Z&row=0')
                    assert code == 200
                    payload = {'loginId': 'legacy-order-editor', 'credential': 'editor-secret', 'from': '2026-09-18T08:30:00Z', 'row': 0}
                    code, command = request('/scenario/submit', payload)
                    assert code == 200 and command['stage'] == 'completed'
                    assert command['observation']['nature'] == 'synthetic'
                    assert 'editor-secret' not in json.dumps(command)
                    if name == 'order':
                        assert request('/scenario/customer-lookup?customerId=11')[0] == 404
                    values = [value, command]
                    traces = [('order-read', 'tei.reference.order-entry.read'), ('order-submit', 'tei.reference.order-entry.submit')]
                if composed:
                    for value, (provider, contract) in zip(values, traces):
                        trace = value['composition']
                        assert trace['providerId'] == provider and trace['contractId'] == contract
                        assert trace['generation'] == 1 and trace['artifactDigest'] == evidence['targets'][name]['digest']
                (output / f'{name}-{app}-{composed}.json').write_text(json.dumps(values, indent=2) + '\n')
                return values
            finally:
                proc.terminate()
                try:
                    proc.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    proc.kill()
                    proc.wait()
                log.close()

        def business(values):
            values = copy.deepcopy(values)
            for value in values:
                value.pop('composition', None)
                if 'read' in value:
                    value['read'].pop('composition', None)
            return values

        for app, tagged in [('order_entry', 'order'), ('repair_intake', 'repair')]:
            baseline = host('combined', app, False)
            combined = host('combined', app, True)
            dedicated = host(tagged, app, True)
            assert business(baseline) == business(combined) == business(dedicated)
        evidence['checks'] = ['three actual artifacts, digests, dependency closures and symbol exclusion',
                              'uncomposed/composed/dedicated business-result parity for two Applications',
                              'composed HTTP input errors and excluded routes', 'actual executable traces and credential absence']
        evidence['cleanAfter'] = not run(['git', 'status', '--porcelain'])
        assert evidence['cleanAfter'] and head == run(['git', 'rev-parse', 'HEAD'])
        evidence['passed'] = True
    finally:
        (output / 'evidence.json').write_text(json.dumps(evidence, indent=2) + '\n')
    print(json.dumps(evidence, indent=2))


if __name__ == '__main__':
    main()
