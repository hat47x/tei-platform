#!/usr/bin/env python3
"""Compare ordinary and generated Studio Hosts through the same Wails RPCs."""

from __future__ import annotations

import argparse
import hashlib
import json
import socket
import subprocess
import time
from pathlib import Path
from urllib.error import URLError
from urllib.request import Request, urlopen


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for block in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def sha256_tree(root: Path) -> str:
    digest = hashlib.sha256()
    for path in sorted(item for item in root.rglob("*") if item.is_file()):
        relative = path.relative_to(root).as_posix().encode("utf-8")
        digest.update(len(relative).to_bytes(8, "big"))
        digest.update(relative)
        digest.update(bytes.fromhex(sha256_file(path)))
    return digest.hexdigest()


def free_loopback_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


def call_rpc(base_url: str, method_name: str, args: list[object], call_id: str) -> dict:
    request_body = json.dumps(
        {"object": 0, "method": 0, "args": {"call-id": call_id, "methodName": method_name, "args": args}}
    ).encode("utf-8")
    request = Request(
        f"{base_url}/wails/runtime",
        data=request_body,
        headers={"Content-Type": "application/json", "Host": base_url.removeprefix("http://"), "Origin": base_url},
        method="POST",
    )
    with urlopen(request, timeout=10) as response:
        if response.status != 200:
            raise AssertionError(f"{method_name} RPC returned HTTP {response.status}")
        return json.load(response)


def wait_for_health(base_url: str, process: subprocess.Popen, log: Path) -> None:
    for _ in range(40):
        if process.poll() is not None:
            raise AssertionError(f"Host exited with {process.returncode}: {log.read_text(errors='replace')}")
        try:
            with urlopen(f"{base_url}/health", timeout=1) as response:
                if response.status == 200:
                    return
        except (OSError, URLError):
            time.sleep(0.5)
    raise AssertionError(f"Host did not become healthy: {log.read_text(errors='replace')}")


def canonical(value: object) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True, separators=(",", ":"))


def normalize_catalog(value: dict) -> dict:
    normalized = dict(value)
    for field in ("entries", "authorityLimits"):
        if isinstance(normalized.get(field), list):
            normalized[field] = sorted(normalized[field], key=canonical)
    return normalized


def normalize_validation(value: dict) -> dict:
    normalized = dict(value)
    if isinstance(normalized.get("diagnostics"), list):
        normalized["diagnostics"] = sorted(normalized["diagnostics"], key=canonical)
    return normalized


def require_success(response: dict, method: str) -> dict:
    if response.get("ok") is not True or response.get("failure") is not None:
        raise AssertionError(f"{method} failed: {response!r}")
    value = response.get("value")
    if not isinstance(value, dict):
        raise AssertionError(f"{method} returned no object value: {response!r}")
    return value


def compare_hosts(ordinary_url: str, generated_url: str, results: dict) -> None:
    for namespace in ("sales", ""):
        ordinary = require_success(
            call_rpc(ordinary_url, "main.StudioBridge.CatalogWorkspace", ["workspace-1", namespace], f"ordinary-catalog-{namespace or 'all'}"),
            "ordinary Catalog",
        )
        generated = require_success(
            call_rpc(generated_url, "main.StudioBridge.CatalogWorkspace", ["workspace-1", namespace], f"generated-catalog-{namespace or 'all'}"),
            "generated Catalog",
        )
        left, right = normalize_catalog(ordinary), normalize_catalog(generated)
        if left != right:
            raise AssertionError(f"Catalog differs for namespace {namespace!r}: ordinary={left!r} generated={right!r}")
        results[f"catalog:{namespace or 'all'}"] = left

    ordinary = require_success(
        call_rpc(ordinary_url, "main.StudioBridge.ValidateWorkspace", ["workspace-1"], "ordinary-validate"),
        "ordinary Validate",
    )
    generated = require_success(
        call_rpc(generated_url, "main.StudioBridge.ValidateWorkspace", ["workspace-1"], "generated-validate"),
        "generated Validate",
    )
    left, right = normalize_validation(ordinary), normalize_validation(generated)
    if left != right:
        raise AssertionError(f"Validate result or diagnostics differ: ordinary={left!r} generated={right!r}")
    results["validate"] = left

    for workspace_id in ("", "../reference/workspace", "workspace-2", "/tmp/arbitrary", " workspace-1"):
        for method, suffix in (("CatalogWorkspace", "catalog"), ("ValidateWorkspace", "validate")):
            responses = [
                call_rpc(url, f"main.StudioBridge.{method}", [workspace_id, ""] if method == "CatalogWorkspace" else [workspace_id], f"{label}-{suffix}-denied-{len(results)}")
                for url, label in ((ordinary_url, "ordinary"), (generated_url, "generated"))
            ]
            outcomes = []
            for response in responses:
                failure = response.get("failure")
                if response.get("ok") is not False or response.get("value") is not None or not isinstance(failure, dict):
                    raise AssertionError(f"{method} admitted or partially answered disallowed workspace {workspace_id!r}: {response!r}")
                if failure.get("code") != "WORKSPACE_NOT_ALLOWED":
                    raise AssertionError(f"{method} returned unexpected denial for {workspace_id!r}: {response!r}")
                outcomes.append({"code": failure.get("code"), "stage": failure.get("stage"), "value": response.get("value")})
            if outcomes[0] != outcomes[1]:
                raise AssertionError(f"{method} denial differs for {workspace_id!r}: {outcomes!r}")
            results[f"denied:{method}:{workspace_id!r}"] = outcomes[0]


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--ordinary", type=Path, required=True)
    parser.add_argument("--generated", type=Path, required=True)
    parser.add_argument("--workspace", type=Path, required=True)
    parser.add_argument("--generator", type=Path, required=True)
    parser.add_argument("--fixture", type=Path, required=True)
    parser.add_argument("--generated-source", type=Path, required=True)
    parser.add_argument("--source-sha", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    ordinary, generated = args.ordinary.resolve(), args.generated.resolve()
    workspace = args.workspace.resolve()
    args.output.mkdir(parents=True, exist_ok=True)
    binaries = {"ordinary": ordinary, "generated": generated}
    processes: list[subprocess.Popen] = []
    urls: dict[str, str] = {}
    try:
        for name, binary in binaries.items():
            port = free_loopback_port()
            base_url = f"http://127.0.0.1:{port}"
            log = args.output / f"{name}-host.log"
            stream = log.open("w", encoding="utf-8")
            process = subprocess.Popen(
                [str(binary), "-workspace", str(workspace), "-port", str(port)],
                stdout=stream,
                stderr=subprocess.STDOUT,
            )
            stream.close()
            processes.append(process)
            urls[name] = base_url
            wait_for_health(base_url, process, log)

        results: dict[str, object] = {}
        compare_hosts(urls["ordinary"], urls["generated"], results)
        comparison = {"sourceSha": args.source_sha, "comparison": results}
        comparison_path = args.output / "semantic-comparison.json"
        comparison_path.write_text(json.dumps(comparison, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

        manifest = {
            "sourceSha": args.source_sha,
            "buildModes": {
                "ordinary": ["server"],
                "generated": ["studio_generated", "server"],
            },
            "workspaceTreeSha256": sha256_tree(workspace),
            "generatorSourceSha256": sha256_file(args.generator.resolve()),
            "fixtureSha256": sha256_file(args.fixture.resolve()),
            "generatedGoSourceSha256": sha256_file(args.generated_source.resolve()),
            "ordinaryBinarySha256": sha256_file(ordinary),
            "generatedBinarySha256": sha256_file(generated),
            "resultFileSha256": sha256_file(comparison_path),
            "goVersion": subprocess.run(["go", "version"], check=True, capture_output=True, text=True).stdout.strip(),
        }
        (args.output / "provenance.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
        print("Ordinary and generated Hosts match for Catalog, Validate diagnostics, and disallowed Workspace IDs.")
    finally:
        for process in processes:
            if process.poll() is None:
                process.terminate()
        for process in processes:
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)


if __name__ == "__main__":
    main()
