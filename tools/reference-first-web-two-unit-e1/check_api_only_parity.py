#!/usr/bin/env python3
"""Compare real Go HTTP processes, not merely an unlinked package graph.

This is diagnostic synthetic-fixture parity; it does not confer production
Authorization, Gate C accepted-Handover meaning, or Browser visual evidence.
"""
import json
import hashlib
import os
import pathlib
import socket
import subprocess
import sys
import time
import urllib.error
import urllib.request


def available_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def request(port, method, path, payload=None):
    url = f"http://127.0.0.1:{port}{path}"
    headers = {"Content-Type": "application/json"} if payload is not None else {}
    req = urllib.request.Request(url, data=payload, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=4) as reply:
            return reply.status, reply.read()
    except urllib.error.HTTPError as err:
        return err.code, err.read()


def boot(binary, repo, revision, workspace_path=None, requested_revision=None, extra_args=()):
    port = available_port()
    command = [
        binary,
        "-workspace", str(workspace_path or repo / "reference/workspace"),
        "-legacy-users", str(repo / "reference/legacy-auth/users.json"),
        "-legacy-authorization", str(repo / "reference/legacy-auth/authorization.json"),
        "-application", "order_entry",
        "-baseline-id", "B",
        "-baseline-revision", "e5231839a5dcc62cce1181a2731fcb7c70d16736",
        "-target-revision", requested_revision if requested_revision is not None else revision,
        "-listen", f"127.0.0.1:{port}",
    ]
    command.extend(extra_args)
    process = subprocess.Popen(command, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    try:
        for _ in range(80):
            if process.poll() is not None:
                raise RuntimeError(f"{binary}: process exited before readiness")
            try:
                response = request(port, "GET", "/manifest.json")
                if response[0] == 200:
                    return process, port
            except (urllib.error.URLError, TimeoutError):
                pass
            time.sleep(0.1)
        raise RuntimeError(f"{binary}: readiness timeout")
    except BaseException:
        process.terminate()
        process.wait(timeout=5)
        raise


def main():
    composed = sys.argv[-1:] == ["--composition"]
    if composed:
        sys.argv.pop()
    if len(sys.argv) not in (4, 5):
        raise SystemExit("usage: check_api_only_parity.py FULL_BINARY API_BINARY GIT_SHA [FROZEN_API_DIRECTORY] [--composition]")
    full, api, revision = sys.argv[1:4]
    api_world_dir = pathlib.Path(sys.argv[4]).resolve() if len(sys.argv) == 5 else None
    repo = pathlib.Path.cwd().resolve()
    processes = []
    try:
        full_proc, full_port = boot(full, repo, revision)
        processes.append(full_proc)
        api_proc, api_port = boot(api, repo, revision, api_world_dir, extra_args=["-composition-scenarios"] if composed else [])
        processes.append(api_proc)
        api_digest = "sha256:" + hashlib.sha256(pathlib.Path(api).read_bytes()).hexdigest() if composed else None
        def comparable(response):
            status, body = response
            if not composed:
                return response
            try:
                value = json.loads(body)
            except json.JSONDecodeError:
                return response
            result = value.get("result")
            if isinstance(result, dict) and "scenarioId" in result:
                def remove_trace(result):
                    trace = result.pop("composition", None)
                    if trace is None or trace.get("artifactDigest") != api_digest or trace.get("generation") != 1:
                        raise RuntimeError("selected service trace does not attest actual API executable")
                    expected = {"order-read": "tei.reference.order-entry.read", "order-submit": "tei.reference.order-entry.submit"}
                    if expected.get(trace.get("providerId")) != trace.get("contractId") or trace.get("contractVersion") != "1":
                        raise RuntimeError("unexpected selected service contract")
                    if isinstance(result.get("read"), dict):
                        remove_trace(result["read"])
                remove_trace(result)
            return status, json.dumps(value, sort_keys=True).encode()
        def expected_comparable(response):
            if composed:
                try:
                    return response[0], json.dumps(json.loads(response[1]), sort_keys=True).encode()
                except json.JSONDecodeError:
                    pass
            return response
        # Prove with the real compiled executable, not just a Go unit test,
        # that a caller cannot invent a Target build revision T that differs
        # from the Git revision embedded by the diagnostic builder.
        wrong_t = "0" * 40 if revision != "0" * 40 else "f" * 40
        try:
            unexpectedly_running, _ = boot(
                api, repo, revision, api_world_dir, requested_revision=wrong_t,
            )
        except RuntimeError as err:
            if "process exited before readiness" not in str(err):
                raise RuntimeError("misbound API start gave an unexpected failure") from err
        else:
            unexpectedly_running.terminate()
            unexpectedly_running.wait(timeout=5)
            raise RuntimeError("API-only executable accepted a different target build revision")
        paths = ["/manifest.json", "/healthz", "/scenario/read"]
        if composed:
            paths.append("/scenario/read?from=2026-09-18T08:30:00Z&row=0")
        for path in paths:
            expected = request(full_port, "GET", path)
            actual = request(api_port, "GET", path)
            if comparable(actual) != expected_comparable(expected):
                raise RuntimeError(f"HTTP GET {path}: API-only response differs")
        cases = {
            "completed": {
                "loginId": "legacy-order-editor", "credential": "editor-secret",
                "from": "2026-09-18T08:30:00Z", "row": 0,
            },
            "authentication-denied": {
                "loginId": "legacy-order-editor", "credential": "wrong-secret",
                "from": "2026-09-18T08:30:00Z", "row": 0,
            },
            "authorization-denied": {
                "loginId": "legacy-order-viewer", "credential": "viewer-secret",
                "from": "2026-09-18T08:30:00Z", "row": 0,
            },
            "validation-not-satisfied": {
                "loginId": "legacy-order-editor", "credential": "editor-secret",
                "from": "2026-09-18T08:00:00Z", "row": 1,
            },
            "execution-error": {
                "loginId": "legacy-order-editor", "credential": "editor-secret",
                "from": "2026-09-18T07:00:00Z", "row": 2,
            },
            "malformed": {"unexpected": True},
        }
        stages = []
        for name, values in cases.items():
            payload = json.dumps(values).encode("utf-8")
            expected = request(full_port, "POST", "/scenario/submit", payload)
            actual = request(api_port, "POST", "/scenario/submit", payload)
            if comparable(actual) != expected_comparable(expected):
                raise RuntimeError(f"HTTP POST /scenario/submit: {name} differs")
            # Do not log or persist request data, response data, or credentials.
            for secret in (b"editor-secret", b"viewer-secret", b"wrong-secret"):
                if secret in actual[1]:
                    raise RuntimeError("API response contains synthetic credentials")
            stages.append(name)
        if composed and api_world_dir is not None:
            try:
                wrongly_bound, _ = boot(api, repo, revision, api_world_dir, extra_args=["-composition-scenarios", "-model-revision", "sha256:live-source"])
            except RuntimeError as err:
                if "process exited before readiness" not in str(err):
                    raise
            else:
                wrongly_bound.terminate()
                wrongly_bound.wait(timeout=5)
                raise RuntimeError("frozen API accepted a live source revision")
        for method in ("GET", "POST"):
            status, _ = request(api_port, method, "/custom-submit")
            if status != 404:
                raise RuntimeError(f"API-only {method} /custom-submit returned {status}")
        status, _ = request(api_port, "GET", "/")
        if status != 404:
            raise RuntimeError(f"API-only index returned {status}")
        print(json.dumps({
            "kind": "diagnostic-api-only-http-parity",
            "revision": revision,
            "syntheticHttpParityVerified": True,
            "scenarioStages": stages,
            "htmlRoutesExcluded": True,
            "productionAuthorizationVerified": False,
            "fullE1BrowserAPIPackagingVerified": False,
            "artifactAuthority": False,
            "selectedServiceCompositionVerified": composed,
            "liveSourceRevisionRejectedForFrozenAPI": composed and api_world_dir is not None,
        }, sort_keys=True))
    finally:
        for process in reversed(processes):
            if process.poll() is None:
                process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)


if __name__ == "__main__":
    main()
