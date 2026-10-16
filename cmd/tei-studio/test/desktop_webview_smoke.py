"""Launch the Wails Desktop build through Edge WebDriver and record a small smoke result."""

from __future__ import annotations

import base64
import ctypes
import hashlib
import io
import json
import os
import re
import socket
import subprocess
import sys
import time
import traceback
import urllib.error
import urllib.parse
import urllib.request
import winreg
import zipfile
from pathlib import Path
from typing import Any


RUNTIME_CLIENT_ID = "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}"
VERSION_RE = re.compile(r"(\d+\.\d+\.\d+\.\d+)")
APP_IMAGE = "tei-studio-desktop-webview-smoke.exe"


def write_json(path: Path, value: Any) -> None:
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def query_runtime_versions() -> list[dict[str, str]]:
    key_path = rf"SOFTWARE\Microsoft\EdgeUpdate\Clients\{RUNTIME_CLIENT_ID}"
    hives = (("HKLM", winreg.HKEY_LOCAL_MACHINE), ("HKCU", winreg.HKEY_CURRENT_USER))
    views = (("64-bit", getattr(winreg, "KEY_WOW64_64KEY", 0)), ("32-bit", getattr(winreg, "KEY_WOW64_32KEY", 0)))
    found: list[dict[str, str]] = []
    seen: set[tuple[str, str]] = set()
    for hive_name, hive in hives:
        for view_name, view_flag in views:
            try:
                with winreg.OpenKey(hive, key_path, 0, winreg.KEY_READ | view_flag) as key:
                    version, _ = winreg.QueryValueEx(key, "pv")
                    if isinstance(version, str) and VERSION_RE.fullmatch(version):
                        item = (hive_name, version)
                        if item not in seen:
                            found.append({"registry_hive": hive_name, "registry_view": view_name, "version": version})
                            seen.add(item)
            except OSError:
                continue
    return found


def download_compatible_driver(runtime_version: str, evidence_dir: Path, facts: dict[str, Any]) -> Path:
    version = urllib.parse.quote(runtime_version, safe=".")
    url = f"https://msedgedriver.microsoft.com/{version}/edgedriver_win64.zip"
    facts["webdriver_download_url"] = url
    request = urllib.request.Request(url, headers={"User-Agent": "TEI-Studio-WebView2-Smoke/1.0"})
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            archive_bytes = response.read()
            facts["webdriver_archive_http_status"] = response.status
            facts["webdriver_archive_bytes"] = len(archive_bytes)
    except Exception as exc:
        raise RuntimeError(f"Could not download Microsoft Edge WebDriver for WebView2 Runtime {runtime_version}: {type(exc).__name__}: {exc}") from exc
    facts["webdriver_archive_sha256"] = hashlib.sha256(archive_bytes).hexdigest()
    target_dir = evidence_dir.parent / "tei-studio-webdriver"
    target_dir.mkdir(parents=True, exist_ok=True)
    try:
        with zipfile.ZipFile(io.BytesIO(archive_bytes)) as archive:
            driver_entries = [item for item in archive.infolist() if Path(item.filename).name.lower() == "msedgedriver.exe"]
            if len(driver_entries) != 1:
                raise RuntimeError(f"Expected one msedgedriver.exe in the Microsoft archive, found {len(driver_entries)}")
            driver_path = target_dir / "msedgedriver.exe"
            driver_path.write_bytes(archive.read(driver_entries[0]))
    except (zipfile.BadZipFile, OSError) as exc:
        raise RuntimeError(f"Could not extract the Microsoft Edge WebDriver archive: {type(exc).__name__}: {exc}") from exc
    return driver_path.resolve()


def version_prefix(version: str) -> tuple[str, str, str]:
    parts = version.split(".")
    if len(parts) != 4:
        raise RuntimeError(f"Expected a four-part version, got {version!r}")
    return parts[0], parts[1], parts[2]


def find_processes(image_name: str) -> list[dict[str, Any]]:
    script = (
        f"$name = '{image_name}'; "
        "$items = @(Get-CimInstance Win32_Process | Where-Object { $_.Name -eq $name } | "
        "Select-Object ProcessId,ParentProcessId,ExecutablePath,@{Name='AppMarker';Expression={$_.CommandLine -like '*tei-studio-desktop-webview-smoke.exe*'}}); "
        "ConvertTo-Json -InputObject $items -Compress -Depth 4"
    )
    result = subprocess.run(
        ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script],
        check=True,
        capture_output=True,
        text=True,
        encoding="utf-8",
        errors="replace",
        timeout=15,
    )
    decoded = json.loads(result.stdout.strip() or "[]")
    if decoded is None:
        return []
    if isinstance(decoded, dict):
        decoded = [decoded]
    return decoded


def free_loopback_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        return int(sock.getsockname()[1])


def webdriver_request(
    base_url: str,
    method: str,
    path: str,
    payload: dict[str, Any] | None = None,
    timeout_seconds: float = 10,
) -> dict[str, Any]:
    body = None if payload is None else json.dumps(payload, ensure_ascii=False).encode("utf-8")
    request = urllib.request.Request(
        f"{base_url}{path}",
        data=body,
        method=method,
        headers={} if body is None else {"Content-Type": "application/json; charset=utf-8"},
    )
    try:
        with urllib.request.urlopen(request, timeout=timeout_seconds) as response:
            text = response.read().decode("utf-8", errors="replace")
            return json.loads(text) if text else {}
    except urllib.error.HTTPError as exc:
        error_body = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"WebDriver {method} {path} returned HTTP {exc.code}: {error_body}") from exc


def open_process_handle(pid: int) -> int:
    kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel32.OpenProcess.argtypes = [ctypes.c_ulong, ctypes.c_int, ctypes.c_ulong]
    kernel32.OpenProcess.restype = ctypes.c_void_p
    handle = kernel32.OpenProcess(0x00100000 | 0x1000, False, pid)  # SYNCHRONIZE | QUERY_LIMITED_INFORMATION
    if not handle:
        raise ctypes.WinError(ctypes.get_last_error())
    return int(handle)


def wait_process(handle: int, timeout_seconds: float) -> int | None:
    kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel32.WaitForSingleObject.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
    kernel32.WaitForSingleObject.restype = ctypes.c_ulong
    kernel32.GetExitCodeProcess.argtypes = [ctypes.c_void_p, ctypes.POINTER(ctypes.c_ulong)]
    kernel32.GetExitCodeProcess.restype = ctypes.c_int
    result = kernel32.WaitForSingleObject(handle, int(timeout_seconds * 1000))
    if result == 0x00000102:  # WAIT_TIMEOUT
        return None
    if result != 0:
        raise ctypes.WinError(ctypes.get_last_error())
    exit_code = ctypes.c_ulong()
    if not kernel32.GetExitCodeProcess(handle, ctypes.byref(exit_code)):
        raise ctypes.WinError(ctypes.get_last_error())
    return int(exit_code.value)


def close_process_handle(handle: int) -> None:
    kernel32 = ctypes.WinDLL("kernel32", use_last_error=True)
    kernel32.CloseHandle.argtypes = [ctypes.c_void_p]
    kernel32.CloseHandle.restype = ctypes.c_int
    kernel32.CloseHandle(handle)


def main() -> int:
    if len(sys.argv) != 3:
        print("usage: desktop_webview_smoke.py <desktop-exe> <evidence-dir>", file=sys.stderr)
        return 2

    app_path = Path(sys.argv[1]).resolve()
    evidence_dir = Path(sys.argv[2]).resolve()
    evidence_dir.mkdir(parents=True, exist_ok=True)
    facts: dict[str, Any] = {
        "result": "FAIL_NOT_COMPLETED",
        "scope": "native Wails WebView2 startup and initial read-only ListWorkspaces rendering for one fixed workspace only",
        "app_path": str(app_path),
        "run_id": os.environ.get("GITHUB_RUN_ID"),
        "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT"),
        "workflow_event_sha": os.environ.get("GITHUB_SHA"),
        "candidate_head_sha": os.environ.get("CANDIDATE_HEAD_SHA"),
        "pr_base_sha": os.environ.get("PR_BASE_SHA"),
        "runner_os": os.environ.get("ImageOS"),
        "runner_image_version": os.environ.get("ImageVersion"),
        "events": [],
    }
    facts_path = evidence_dir / "summary.json"
    driver_stdout = (evidence_dir / "msedgedriver.stdout.log").open("w", encoding="utf-8", errors="replace")
    driver_stderr = (evidence_dir / "msedgedriver.stderr.log").open("w", encoding="utf-8", errors="replace")
    driver_process: subprocess.Popen[str] | None = None
    driver_session: str | None = None
    app_handle: int | None = None
    app_pid: int | None = None
    failure: str | None = None

    def record(event: str, **values: Any) -> None:
        facts["events"].append({"event": event, **values})
        write_json(facts_path, facts)
        print(f"{event}: {json.dumps(values, ensure_ascii=False, sort_keys=True)}", flush=True)

    try:
        facts["os_version"] = subprocess.run(
            ["powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[System.Environment]::OSVersion.VersionString"],
            check=True,
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            timeout=15,
        ).stdout.strip()
        facts["python_version"] = sys.version
        facts["go_version"] = subprocess.run(["go", "version"], check=True, capture_output=True, text=True, timeout=15).stdout.strip()
        facts["checkout_sha"] = subprocess.run(["git", "rev-parse", "HEAD"], check=True, capture_output=True, text=True, timeout=15).stdout.strip()
        if facts["candidate_head_sha"] and facts["checkout_sha"] != facts["candidate_head_sha"]:
            raise RuntimeError(f"Checkout SHA {facts['checkout_sha']} does not match candidate head {facts['candidate_head_sha']}")
        if not app_path.is_file():
            raise RuntimeError(f"Desktop executable does not exist: {app_path}")
        facts["exe_sha256"] = hashlib.sha256(app_path.read_bytes()).hexdigest()
        record("build-and-runner", os_version=facts["os_version"], runner_image=facts["runner_image_version"], exe_sha256=facts["exe_sha256"])

        runtime_versions = query_runtime_versions()
        facts["webview2_runtime_registry"] = runtime_versions
        if not runtime_versions:
            raise RuntimeError("WebView2 Evergreen Runtime version was not found in the EdgeUpdate Runtime registry keys")
        runtime_version = runtime_versions[0]["version"]
        facts["webview2_runtime_version"] = runtime_version
        record("webview2-runtime-detected", versions=runtime_versions)

        driver_path = download_compatible_driver(runtime_version, evidence_dir, facts)
        facts["webdriver_path"] = str(driver_path)
        version_result = subprocess.run([str(driver_path), "--version"], check=True, capture_output=True, text=True, timeout=15)
        version_output = (version_result.stdout + version_result.stderr).strip()
        match = VERSION_RE.search(version_output)
        if not match:
            facts["webdriver_version_output"] = version_output
            raise RuntimeError("Could not parse four-part version from msedgedriver --version")
        driver_version = match.group(1)
        facts["webdriver_version"] = driver_version
        facts["webdriver_version_output"] = version_output
        facts["webdriver_compatibility_rule"] = "WebView2 Runtime and Edge WebDriver first three version components must match"
        if version_prefix(runtime_version) != version_prefix(driver_version):
            raise RuntimeError(f"WebView2/Edge WebDriver version mismatch: Runtime={runtime_version}, Driver={driver_version}")
        record("webdriver-version-compatible", runtime_version=runtime_version, webdriver_version=driver_version)

        app_baseline = find_processes(APP_IMAGE)
        if app_baseline:
            facts["preexisting_app_processes"] = app_baseline
            raise RuntimeError("A process with the smoke executable name existed before WebDriver launch")
        workspace_root = Path(os.environ.get("GITHUB_WORKSPACE", Path.cwd())).resolve() / "reference" / "workspace"
        if not workspace_root.is_dir():
            raise RuntimeError(f"Fixed ListWorkspaces fixture is missing: {workspace_root}")
        facts["workspace_root"] = str(workspace_root)
        webview_baseline = find_processes("msedgewebview2.exe")
        baseline_pids = {int(item["ProcessId"]) for item in webview_baseline}
        facts["preexisting_webview2_count"] = len(baseline_pids)

        port = free_loopback_port()
        base_url = f"http://127.0.0.1:{port}"
        facts["webdriver_url"] = base_url
        driver_env = os.environ.copy()
        driver_env["MSEDGEDRIVER_TELEMETRY_OPTOUT"] = "1"
        inherited_server_host = driver_env.pop("WAILS_SERVER_HOST", None)
        facts["inherited_wails_server_host_cleared"] = inherited_server_host is not None
        driver_process = subprocess.Popen(
            [str(driver_path), "--verbose", f"--port={port}"],
            stdout=driver_stdout,
            stderr=driver_stderr,
            env=driver_env,
            text=True,
        )
        facts["webdriver_pid"] = driver_process.pid
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            if driver_process.poll() is not None:
                raise RuntimeError(f"msedgedriver exited before readiness with code {driver_process.returncode}")
            try:
                status = webdriver_request(base_url, "GET", "/status")
                if status.get("value", {}).get("ready") is True:
                    facts["webdriver_status"] = status
                    break
            except Exception:
                pass
            time.sleep(0.25)
        else:
            raise RuntimeError("msedgedriver did not become ready on its loopback port")
        record("webdriver-ready", pid=driver_process.pid, status=facts["webdriver_status"])

        capabilities = {
            "browserName": "webview2",
            "ms:edgeOptions": {
                "binary": str(app_path),
                "args": [f"--workspace={workspace_root}"],
            },
        }
        session_response = webdriver_request(
            base_url,
            "POST",
            "/session",
            {"capabilities": {"alwaysMatch": capabilities}},
            timeout_seconds=60,
        )
        value = session_response.get("value", {})
        driver_session = session_response.get("sessionId") or value.get("sessionId")
        if not driver_session:
            raise RuntimeError(f"WebDriver did not return a session ID: {session_response}")
        facts["webdriver_session_capabilities"] = value.get("capabilities", {})
        record("wails-webview-session-created", session_id=driver_session, capabilities=facts["webdriver_session_capabilities"])

        app_deadline = time.monotonic() + 20
        app_processes: list[dict[str, Any]] = []
        while time.monotonic() < app_deadline:
            app_processes = find_processes(APP_IMAGE)
            if app_processes:
                break
            time.sleep(0.25)
        if not app_processes:
            raise RuntimeError("Edge WebDriver created a session but the Wails executable process was not observable")
        app_pid = int(app_processes[0]["ProcessId"])
        app_handle = open_process_handle(app_pid)
        facts["app_process"] = app_processes[0]
        record("wails-process-observed", app_process=app_processes[0])

        webview_processes = find_processes("msedgewebview2.exe")
        facts["webview2_processes_during_session"] = webview_processes
        new_webview_pids = sorted(
            int(item["ProcessId"])
            for item in webview_processes
            if int(item["ProcessId"]) not in baseline_pids
            and (int(item["ParentProcessId"]) == app_pid or item.get("AppMarker") is True)
        )
        facts["new_webview2_pids"] = new_webview_pids
        if not new_webview_pids:
            raise RuntimeError("No new msedgewebview2.exe process was observed after Wails WebView launch")
        record("webview2-process-observed", new_pids=new_webview_pids)

        inspect_script = """
return {
  title: document.title,
  heading: document.querySelector('h1')?.innerText ?? '',
  state: document.querySelector('#state')?.innerText ?? '',
  output: document.querySelector('#output')?.innerText ?? '',
  workspaceOptions: Array.from(document.querySelectorAll('#workspace option')).map((item) => ({value:item.value,label:item.textContent})),
  readyState: document.readyState
};
"""
        page_state: dict[str, Any] = {}
        page_deadline = time.monotonic() + 25
        while time.monotonic() < page_deadline:
            response = webdriver_request(
                base_url,
                "POST",
                f"/session/{driver_session}/execute/sync",
                {"script": inspect_script, "args": []},
            )
            page_state = response.get("value", {})
            if page_state.get("output") and "Operation応答" in page_state.get("state", ""):
                break
            time.sleep(0.25)
        facts["page_state"] = page_state
        write_json(evidence_dir / "page-state.json", page_state)
        screenshot_response = webdriver_request(base_url, "GET", f"/session/{driver_session}/screenshot")
        screenshot = screenshot_response.get("value")
        if not screenshot:
            raise RuntimeError("WebDriver screenshot response did not contain image data")
        (evidence_dir / "webview2-initial-screen.png").write_bytes(base64.b64decode(screenshot))
        if not page_state.get("title", "").startswith("TEI Studio — read-only PoC A"):
            raise RuntimeError(f"Unexpected Desktop page title: {page_state.get('title')!r}")
        if "TEI Studio — PoC A" not in page_state.get("heading", ""):
            raise RuntimeError(f"Expected Studio heading was not visible: {page_state.get('heading')!r}")
        if "Operation応答" not in page_state.get("state", ""):
            raise RuntimeError(f"Initial ListWorkspaces did not render an operation response: {page_state.get('state')!r}")
        envelope = json.loads(page_state.get("output", ""))
        expected_envelope = {"ok": True, "value": [{"id": "workspace-1", "label": "workspace"}]}
        if envelope != expected_envelope:
            raise RuntimeError(f"Initial ListWorkspaces result differed from the fixed fixture response: {envelope!r}")
        expected_options = [{"value": "workspace-1", "label": "workspace (workspace-1)"}]
        if page_state.get("workspaceOptions") != expected_options:
            raise RuntimeError(f"Initial workspace option was not rendered: {page_state.get('workspaceOptions')!r}")
        record("initial-screen-and-listworkspaces-verified", title=page_state["title"], heading=page_state["heading"], state=page_state["state"], listworkspaces=envelope)

    except Exception as exc:  # Keep a failure artifact even when setup cannot reach the Wails app.
        failure = f"{type(exc).__name__}: {exc}"
        facts["failure"] = failure
        (evidence_dir / "failure.txt").write_text(traceback.format_exc(), encoding="utf-8")
        record("smoke-failed", reason=failure)
    finally:
        if driver_session and driver_process and driver_process.poll() is None:
            try:
                response = webdriver_request(f"http://127.0.0.1:{facts['webdriver_url'].rsplit(':', 1)[1]}", "DELETE", f"/session/{driver_session}")
                facts["webdriver_delete_session_response"] = response
                record("webdriver-session-closed", response=response)
            except Exception as exc:
                facts["webdriver_delete_session_error"] = f"{type(exc).__name__}: {exc}"
                failure = failure or facts["webdriver_delete_session_error"]

        if app_handle is not None:
            try:
                exit_code = wait_process(app_handle, 20)
                facts["app_exit_code"] = exit_code
                if exit_code is None:
                    facts["normal_exit"] = "FAIL_TIMEOUT"
                    failure = failure or "Wails Desktop process did not exit after WebDriver closed its session"
                    if app_pid is not None:
                        cleanup = subprocess.run(["taskkill.exe", "/F", "/T", "/PID", str(app_pid)], capture_output=True, text=True, timeout=15)
                        facts["forced_process_cleanup"] = {"returncode": cleanup.returncode, "stdout": cleanup.stdout, "stderr": cleanup.stderr}
                elif exit_code != 0:
                    facts["normal_exit"] = "FAIL_NONZERO"
                    failure = failure or f"Wails Desktop exited with code {exit_code}"
                else:
                    facts["normal_exit"] = "PASS"
                record("wails-process-exit", exit_code=exit_code, result=facts["normal_exit"])
            except Exception as exc:
                facts["normal_exit"] = "FAIL_OBSERVATION"
                failure = failure or f"{type(exc).__name__}: {exc}"
                record("wails-process-exit-observation-failed", reason=failure)
            finally:
                try:
                    close_process_handle(app_handle)
                except Exception as exc:
                    facts["process_handle_close_error"] = f"{type(exc).__name__}: {exc}"
                    failure = failure or facts["process_handle_close_error"]
        else:
            try:
                remaining = find_processes(APP_IMAGE)
                if remaining:
                    facts["unobserved_app_processes_after_cleanup"] = remaining
                    failure = failure or "A Wails Desktop process remained without an observed normal-exit handle"
                    for item in remaining:
                        pid = int(item["ProcessId"])
                        cleanup = subprocess.run(["taskkill.exe", "/F", "/T", "/PID", str(pid)], capture_output=True, text=True, timeout=15)
                        facts.setdefault("forced_process_cleanup", []).append(
                            {"pid": pid, "returncode": cleanup.returncode, "stdout": cleanup.stdout, "stderr": cleanup.stderr}
                        )
            except Exception as exc:
                facts["leftover_process_check_error"] = f"{type(exc).__name__}: {exc}"
                failure = failure or facts["leftover_process_check_error"]

        if driver_process is not None and driver_process.poll() is None:
            try:
                driver_process.terminate()
                try:
                    driver_process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    driver_process.kill()
                    driver_process.wait(timeout=5)
            except Exception as exc:
                facts["webdriver_process_cleanup_error"] = f"{type(exc).__name__}: {exc}"
                failure = failure or facts["webdriver_process_cleanup_error"]
        driver_stdout.close()
        driver_stderr.close()

        facts["result"] = "FAIL" if failure else "PASS_NATIVE_WEBVIEW_STARTUP_ONLY"
        if failure:
            facts["failure"] = failure
            if not (evidence_dir / "failure.txt").exists():
                (evidence_dir / "failure.txt").write_text(failure + "\n", encoding="utf-8")
        write_json(facts_path, facts)
        print(f"SMOKE_RESULT={facts['result']}", flush=True)
    return 1 if failure else 0


if __name__ == "__main__":
    raise SystemExit(main())
