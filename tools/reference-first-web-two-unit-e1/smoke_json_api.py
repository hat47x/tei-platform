#!/usr/bin/env python3
"""Real-process synthetic HTTP comparison of two JSON-API deployment units.

Does not certify production identity, CSRF policy, browser Visual Fidelity,
a signed Build Artifact Lock or all accepted Gate C scenarios.
"""
import hashlib
import json
import pathlib
import re
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
import socket


def free_port():
    with socket.socket() as sock:
        sock.bind(("127.0.0.1", 0))
        return sock.getsockname()[1]


def fetch(url, method="GET", body=None, headers=None):
    req = urllib.request.Request(url, method=method, data=body, headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=5) as response:
            return response.status, response.read(), dict(response.headers)
    except urllib.error.HTTPError as err:
        return err.code, err.read(), dict(err.headers)


def boot(command, log):
    proc = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT)
    return proc


def wait_for(url, proc):
    for _ in range(100):
        if proc.poll() is not None:
            raise RuntimeError("diagnostic process ended before startup")
        try:
            if fetch(url)[0] == 200:
                return
        except (urllib.error.URLError, TimeoutError):
            pass
        time.sleep(0.1)
    raise RuntimeError("diagnostic process startup timeout")


def must_same(a, b, label):
    if a[0] != b[0] or a[1] != b[1]:
        raise RuntimeError(f"API and Browser transport differ for {label}")


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: python3 smoke_json_api.py /absolute/artifact-directory")
    directory = pathlib.Path(sys.argv[1]).resolve(strict=True)
    repo = pathlib.Path.cwd().resolve()
    if repo == directory or repo in directory.parents:
        raise SystemExit("artifact directory must be outside checkout")
    measurement = json.loads((directory / "measurement.json").read_text("utf-8"))
    if measurement["browserTransport"] != "same-origin-json" or measurement["applicationId"] != "order_entry":
        raise SystemExit("unexpected source/transport identity")
    revision = subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip()
    if revision != measurement["browserAndApiSourceRevision"]:
        raise SystemExit("built revision does not match checkout")
    assets = (
        ("browser/index.html", "browserSHA256"),
        ("browser/browser-command.js", "browserScriptSHA256"),
        ("api/order-entry", "apiSHA256"),
    )
    for filename, key in assets:
        actual = hashlib.sha256((directory / filename).read_bytes()).hexdigest()
        if actual != measurement[key]:
            raise SystemExit(f"untrusted or changed build unit: {filename}")
    if b"<form id=\"json-command\" method=\"post\" action=\"/scenario/submit\"" not in (directory / "browser/index.html").read_bytes():
        raise SystemExit("JSON Browser transport is not selected")

    with tempfile.TemporaryDirectory(prefix="tei-json-e1-") as scratch:
        temp = pathlib.Path(scratch)
        api_port = free_port()
        api = f"http://127.0.0.1:{api_port}"
        browser_process = None
        api_process = None
        try:
            with (temp / "api.log").open("wb") as logfile:
                api_process = boot([
                    str(directory / "api/order-entry"),
                    "-workspace", str(repo / "reference/workspace"),
                    "-legacy-users", str(repo / "reference/legacy-auth/users.json"),
                    "-legacy-authorization", str(repo / "reference/legacy-auth/authorization.json"),
                    "-application", "order_entry", "-baseline-id", measurement["handoverBaselineId"],
                    "-baseline-revision", measurement["baselineRepositoryRevision"],
                    "-target-revision", revision, "-listen", f"127.0.0.1:{api_port}",
                ], logfile)
                wait_for(api + "/manifest.json", api_process)
            with (temp / "browser.log").open("wb") as logfile:
                browser_process = boot([
                    "node", "tools/reference-first-web-two-unit-e1/browser-host.mjs",
                    str(directory / "browser"), api,
                ], logfile)
                browser = None
                for _ in range(100):
                    if browser_process.poll() is not None:
                        raise RuntimeError("Browser test host exited")
                    match = re.search(rb"BROWSER_HOST=(http://127\.0\.0\.1:\d+)",
                                      (temp / "browser.log").read_bytes())
                    if match:
                        browser = match.group(1).decode()
                        break
                    time.sleep(0.1)
                if not browser:
                    raise RuntimeError("Browser host not ready")
                status, content, headers = fetch(browser + "/")
                if status != 200 or content != (directory / "browser/index.html").read_bytes():
                    raise RuntimeError("Browser static document was changed")
                if "script-src 'self'" not in headers.get("Content-Security-Policy", ""):
                    raise RuntimeError("generated script was not permitted by CSP")
                js = fetch(browser + "/browser-command.js")
                if js[0] != 200 or js[1] != (directory / "browser/browser-command.js").read_bytes():
                    raise RuntimeError("Browser JavaScript differs from build")
                must_same(fetch(api + "/manifest.json"), fetch(browser + "/manifest.json"), "manifest")
                must_same(fetch(api + "/scenario/read"), fetch(browser + "/scenario/read"), "read")
                examples = {
                    "completed": ("legacy-order-editor", "editor-secret", "2026-09-18T08:30:00Z", 0),
                    "authentication-denied": ("legacy-order-editor", "wrong-secret", "2026-09-18T08:30:00Z", 0),
                    "authorization-denied": ("legacy-order-viewer", "viewer-secret", "2026-09-18T08:30:00Z", 0),
                    "validation-not-satisfied": ("legacy-order-editor", "editor-secret", "2026-09-18T08:00:00Z", 1),
                    "execution-error": ("legacy-order-editor", "editor-secret", "2026-09-18T07:00:00Z", 2),
                }
                for label, (login, credential, from_value, row) in examples.items():
                    payload = json.dumps({"loginId": login, "credential": credential,
                                          "from": from_value, "row": row}).encode("utf-8")
                    headers = {"Content-Type": "application/json"}
                    direct = fetch(api + "/scenario/submit", "POST", payload, headers)
                    proxied = fetch(browser + "/scenario/submit", "POST", payload,
                                    {**headers, "Origin": browser})
                    must_same(direct, proxied, label)
                    if any(secret in proxied[1] for secret in (b"editor-secret", b"wrong-secret", b"viewer-secret")):
                        raise RuntimeError("secret echoed to browser")
                    doc = json.loads(proxied[1])
                    if label == "execution-error":
                        if proxied[0] != 422 or not doc.get("executionError"):
                            raise RuntimeError("execution failure was misclassified")
                    elif proxied[0] != 200 or doc.get("result", {}).get("stage") != label:
                        raise RuntimeError(f"{label}: business stage misclassified")
                if fetch(browser + "/custom-submit")[0] != 404 or fetch(api + "/custom-submit")[0] != 404:
                    raise RuntimeError("legacy HTML handler was not removed")
                payload = b'{"loginId":"x"}'
                if fetch(browser + "/scenario/submit", "POST", payload,
                         {"Content-Type": "application/json", "Origin": "http://evil.invalid"})[0] != 403:
                    raise RuntimeError("cross origin allowed")
                if fetch(browser + "/scenario/submit", "POST", payload,
                         {"Content-Type": "application/json"})[0] != 403:
                    raise RuntimeError("request without Origin allowed")
                if fetch(browser + "/scenario/submit", "POST", b"test",
                         {"Content-Type": "text/plain", "Origin": browser})[0] != 415:
                    raise RuntimeError("unsafe Content-Type allowed")
                if fetch(browser + "/admin")[0] != 404:
                    raise RuntimeError("unknown route mounted")
                api_process.terminate()
                api_process.wait(timeout=5)
                api_process = None
                if fetch(browser + "/")[1] != content:
                    raise RuntimeError("static Browser disappeared after API stop")
                if fetch(browser + "/scenario/submit", "POST", payload,
                         {"Content-Type": "application/json", "Origin": browser})[0] != 502:
                    raise RuntimeError("API failure treated as a success")
        finally:
            for proc in (browser_process, api_process):
                if proc is not None:
                    if proc.poll() is None:
                        proc.terminate()
                    try:
                        proc.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        proc.kill()
                        proc.wait(timeout=5)
    receipt = {
        "kind": "diagnostic-static-browser-to-api-json-http",
        "sourceRevision": revision,
        "browserStaticAndScriptMatchedDigests": True,
        "jsonApiBusinessStagesMatchedDirect": True,
        "originAndMediaTypeDenied": True,
        "unavailableApiNotPresentedAsSuccess": True,
        "realBrowserInteractionVerified": False,
        "productionAuthorizationVerified": False,
        "fullE1BrowserAPIPackagingVerified": False,
        "artifactAuthority": False,
    }
    (directory / "json-api-smoke.json").write_text(json.dumps(receipt, indent=2) + "\n", "utf-8")
    print(json.dumps(receipt, sort_keys=True))


if __name__ == "__main__":
    main()
