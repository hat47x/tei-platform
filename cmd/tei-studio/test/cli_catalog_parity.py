#!/usr/bin/env python3
"""Compare the CLI process and a live Wails CatalogWorkspace RPC."""

from __future__ import annotations

import argparse
import json
import socket
import subprocess
import tempfile
import time
from pathlib import Path
from urllib.error import URLError
from urllib.request import Request, urlopen


RPC_METHOD = "main.StudioBridge.CatalogWorkspace"


def run_cli(cli: Path, workspace: Path, namespace: str) -> tuple[int, dict]:
    command = [str(cli), "model", "catalog", "-workspace", str(workspace)]
    if namespace:
        command.extend(["-namespace", namespace])
    result = subprocess.run(command, capture_output=True, text=True, check=False)
    try:
        envelope = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise AssertionError(
            f"CLI did not return a JSON envelope: exit={result.returncode} "
            f"stdout={result.stdout!r} stderr={result.stderr!r}"
        ) from exc
    return result.returncode, envelope


def call_catalog(base_url: str, workspace_id: str, namespace: str, call_id: str) -> dict:
    authority = base_url.removeprefix("http://")
    request_body = json.dumps(
        {
            "object": 0,
            "method": 0,
            "args": {
                "call-id": call_id,
                "methodName": RPC_METHOD,
                "args": [workspace_id, namespace],
            },
        }
    ).encode("utf-8")
    request = Request(
        f"{base_url}/wails/runtime",
        data=request_body,
        headers={
            "Content-Type": "application/json",
            "Host": authority,
            "Origin": base_url,
        },
        method="POST",
    )
    with urlopen(request, timeout=5) as response:
        if response.status != 200:
            raise AssertionError(f"Catalog RPC returned HTTP {response.status}")
        try:
            return json.load(response)
        except json.JSONDecodeError as exc:
            raise AssertionError("Catalog RPC did not return a JSON body") from exc


def normalize_catalog(catalog: dict) -> dict:
    normalized = dict(catalog)
    normalized["entries"] = sorted(
        normalized.get("entries", []),
        key=lambda entry: json.dumps(entry, sort_keys=True, separators=(",", ":")),
    )
    normalized["authorityLimits"] = sorted(normalized.get("authorityLimits", []))
    return normalized


def compare_success(cli: Path, workspace: Path, base_url: str, namespace: str) -> None:
    exit_code, envelope = run_cli(cli, workspace, namespace)
    if exit_code != 0 or envelope.get("ok") is not True or not isinstance(envelope.get("result"), dict):
        raise AssertionError(f"CLI Catalog failed: exit={exit_code} envelope={envelope!r}")

    response = call_catalog(base_url, "workspace-1", namespace, f"cli-catalog-{namespace or 'all'}")
    if response.get("ok") is not True or not isinstance(response.get("value"), dict):
        raise AssertionError(f"Studio Catalog failed: response={response!r}")
    if response.get("failure") is not None:
        raise AssertionError(f"successful Studio Catalog carried a failure: {response!r}")

    cli_value = normalize_catalog(envelope["result"])
    studio_value = normalize_catalog(response["value"])
    if cli_value != studio_value:
        raise AssertionError(
            f"CLI process and Wails runtime Catalog results differ for namespace {namespace!r}:\n"
            f"CLI={json.dumps(cli_value, ensure_ascii=False, sort_keys=True)}\n"
            f"Studio={json.dumps(studio_value, ensure_ascii=False, sort_keys=True)}"
        )


def free_loopback_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


def wait_for_health(base_url: str, process: subprocess.Popen) -> None:
    for _ in range(30):
        if process.poll() is not None:
            raise AssertionError(f"failure-fixture Studio server exited with {process.returncode}")
        try:
            with urlopen(f"{base_url}/health", timeout=1) as response:
                if response.status == 200:
                    return
        except (OSError, URLError):
            time.sleep(0.5)
    raise AssertionError("failure-fixture Studio server did not become healthy")


def compare_load_failure(cli: Path, server: Path) -> None:
    with tempfile.TemporaryDirectory(prefix="tei-studio-empty-workspace-") as empty_workspace:
        workspace = Path(empty_workspace).resolve()
        port = free_loopback_port()
        base_url = f"http://127.0.0.1:{port}"
        with tempfile.TemporaryFile(mode="w+t") as server_log:
            process = subprocess.Popen(
                [str(server), "-workspace", str(workspace), "-port", str(port)],
                stdout=server_log,
                stderr=subprocess.STDOUT,
            )
            try:
                wait_for_health(base_url, process)
                exit_code, envelope = run_cli(cli, workspace, "")
                if exit_code != 2 or envelope.get("ok") is not False:
                    raise AssertionError(f"CLI did not report a load failure with exit 2: {envelope!r}")
                if envelope.get("result") is not None or envelope.get("error", {}).get("code") != "WORKSPACE_LOAD_FAILED":
                    raise AssertionError(f"CLI load failure envelope differs: {envelope!r}")

                response = call_catalog(base_url, "workspace-1", "", "cli-catalog-load-failure")
                failure = response.get("failure")
                if response.get("ok") is not False or response.get("value") is not None:
                    raise AssertionError(f"Studio load failure returned a success or partial value: {response!r}")
                if not isinstance(failure, dict) or failure.get("code") != envelope["error"]["code"]:
                    raise AssertionError(f"CLI and Studio load failure codes differ: CLI={envelope!r} Studio={response!r}")
                if failure.get("stage") != "load":
                    raise AssertionError(f"Studio load failure stage differs: {response!r}")
            except AssertionError as exc:
                server_log.seek(0)
                raise AssertionError(f"{exc}; failure server log: {server_log.read()}") from exc
            finally:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cli", type=Path, required=True)
    parser.add_argument("--server", type=Path, required=True)
    parser.add_argument("--workspace", type=Path, required=True)
    parser.add_argument("--base-url", required=True)
    args = parser.parse_args()

    cli = args.cli.resolve()
    server = args.server.resolve()
    workspace = args.workspace.resolve()
    compare_success(cli, workspace, args.base_url, "sales")
    compare_success(cli, workspace, args.base_url, "")
    compare_load_failure(cli, server)
    print("CLI process and live Wails CatalogWorkspace results match for sales, all namespaces, and load failure.")


if __name__ == "__main__":
    main()
