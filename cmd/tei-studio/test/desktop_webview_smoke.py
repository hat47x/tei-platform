"""Launch the Wails Desktop build through Edge WebDriver and record a small smoke result."""

from __future__ import annotations

import base64
import ctypes
import hashlib
import io
import json
import os
import re
import shutil
import socket
import subprocess
import sys
import tempfile
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
        "Select-Object ProcessId,ParentProcessId,ExecutablePath,@{Name='AppMarker';Expression={$_.CommandLine -like '*tei-studio-desktop-webview-smoke.exe*'}},@{Name='RemoteDebugPort';Expression={if ($_.CommandLine -match '--remote-debugging-port=(\\d+)') {$Matches[1]} else {$null}}}); "
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


def query_listening_sockets(port: int) -> list[dict[str, Any]]:
    script = (
        f"$port = {port}; "
        "$items = @(Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue | "
        "Select-Object LocalAddress,LocalPort,OwningProcess); "
        "ConvertTo-Json -InputObject $items -Compress -Depth 3"
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


def webdriver_key_actions(base_url: str, session_id: str, events: list[dict[str, str]]) -> None:
    webdriver_request(
        base_url,
        "POST",
        f"/session/{session_id}/actions",
        {"actions": [{"type": "key", "id": "tei-studio-keyboard", "actions": events}]},
    )


def workspace_snapshot(root: Path) -> dict[str, Any]:
    digest = hashlib.sha256()
    files = 0
    items = sorted(root.rglob("*"), key=lambda item: item.relative_to(root).as_posix())
    for path in items:
        relative = path.relative_to(root).as_posix()
        if path.is_dir():
            item_type = "directory"
            content_hash = ""
        elif path.is_file():
            item_type = "file"
            content_hash = hashlib.sha256(path.read_bytes()).hexdigest()
            files += 1
        elif path.is_symlink():
            item_type = "symlink"
            content_hash = hashlib.sha256(os.readlink(path).encode("utf-8")).hexdigest()
        else:
            item_type = "other"
            content_hash = ""
        digest.update(item_type.encode("ascii"))
        digest.update(b"\0")
        digest.update(relative.encode("utf-8"))
        digest.update(b"\0")
        digest.update(content_hash.encode("ascii"))
        digest.update(b"\n")
    return {"entry_count": len(items), "file_count": files, "sha256": digest.hexdigest()}


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


def request_window_close(pid: int) -> list[str]:
    user32 = ctypes.WinDLL("user32", use_last_error=True)
    enum_windows = user32.EnumWindows
    enum_windows.argtypes = [ctypes.c_void_p, ctypes.c_void_p]
    enum_windows.restype = ctypes.c_int
    get_window_pid = user32.GetWindowThreadProcessId
    get_window_pid.argtypes = [ctypes.c_void_p, ctypes.POINTER(ctypes.c_ulong)]
    get_window_pid.restype = ctypes.c_ulong
    is_visible = user32.IsWindowVisible
    is_visible.argtypes = [ctypes.c_void_p]
    is_visible.restype = ctypes.c_int
    post_message = user32.PostMessageW
    post_message.argtypes = [ctypes.c_void_p, ctypes.c_uint, ctypes.c_size_t, ctypes.c_ssize_t]
    post_message.restype = ctypes.c_int
    windows: list[str] = []
    errors: list[int] = []
    callback_type = ctypes.WINFUNCTYPE(ctypes.c_int, ctypes.c_void_p, ctypes.c_void_p)

    def visit(hwnd: int, _parameter: int) -> int:
        window_pid = ctypes.c_ulong()
        get_window_pid(hwnd, ctypes.byref(window_pid))
        if window_pid.value == pid and is_visible(hwnd):
            if not post_message(hwnd, 0x0010, 0, 0):  # WM_CLOSE
                errors.append(ctypes.get_last_error())
            else:
                windows.append(hex(int(hwnd)))
        return 1

    callback = callback_type(visit)
    if not enum_windows(callback, 0):
        raise ctypes.WinError(ctypes.get_last_error())
    if errors:
        raise ctypes.WinError(errors[0])
    if not windows:
        raise RuntimeError(f"No visible top-level window was found for Wails process {pid}")
    return windows


def main() -> int:
    sys.stdout.reconfigure(encoding="utf-8", errors="strict")
    sys.stderr.reconfigure(encoding="utf-8", errors="strict")
    if len(sys.argv) != 3:
        print("usage: desktop_webview_smoke.py <desktop-exe> <evidence-dir>", file=sys.stderr)
        return 2

    app_path = Path(sys.argv[1]).resolve()
    evidence_dir = Path(sys.argv[2]).resolve()
    evidence_dir.mkdir(parents=True, exist_ok=True)
    facts: dict[str, Any] = {
        "result": "FAIL_NOT_COMPLETED",
        "scope": "native Wails WebView2 Workbench Catalog, read-only Inspect projection, Validate display, native read-only Editor keyboard flow, and fixed-fixture ObserveXead UI flow",
        "app_path": str(app_path),
        "run_id": os.environ.get("GITHUB_RUN_ID"),
        "run_attempt": os.environ.get("GITHUB_RUN_ATTEMPT"),
        "workflow_event_sha": os.environ.get("GITHUB_SHA"),
        "candidate_head_sha": os.environ.get("CANDIDATE_HEAD_SHA"),
        "pr_base_sha": os.environ.get("PR_BASE_SHA"),
        "runner_os": os.environ.get("ImageOS"),
        "runner_image_version": os.environ.get("ImageVersion"),
        "session_delete_attempted": False,
        "session_deleted": False,
        "events": [],
    }
    facts_path = evidence_dir / "summary.json"
    driver_stdout = (evidence_dir / "msedgedriver.stdout.log").open("w", encoding="utf-8", errors="replace")
    driver_stderr = (evidence_dir / "msedgedriver.stderr.log").open("w", encoding="utf-8", errors="replace")
    app_stdout = (evidence_dir / "wails-app.stdout.log").open("w", encoding="utf-8", errors="replace")
    app_stderr = (evidence_dir / "wails-app.stderr.log").open("w", encoding="utf-8", errors="replace")
    driver_process: subprocess.Popen[str] | None = None
    app_process: subprocess.Popen[str] | None = None
    driver_session: str | None = None
    app_handle: int | None = None
    app_pid: int | None = None
    scratch_workspace: tempfile.TemporaryDirectory[str] | None = None
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
        source_workspace = Path(os.environ.get("GITHUB_WORKSPACE", Path.cwd())).resolve() / "reference" / "workspace"
        fixture_source = Path(__file__).resolve().parent / "fixtures" / "observe-xead" / "model.xead"
        if not source_workspace.is_dir():
            raise RuntimeError(f"Fixed ListWorkspaces fixture is missing: {source_workspace}")
        if not fixture_source.is_file():
            raise RuntimeError(f"Fixed ObserveXead fixture is missing: {fixture_source}")
        scratch_workspace = tempfile.TemporaryDirectory(prefix="tei-studio-xead-smoke-")
        scratch_root = Path(scratch_workspace.name)
        workspace_root = scratch_root / "workspace"
        shutil.copytree(source_workspace, workspace_root)
        relative_xead_path = "tei-studio-smoke/observe.xead"
        xead_path = workspace_root / relative_xead_path
        xead_path.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(fixture_source, xead_path)
        fixture_sha256_before = hashlib.sha256(xead_path.read_bytes()).hexdigest()
        relative_empty_xead_path = "tei-studio-smoke/empty.xead"
        empty_xead_path = workspace_root / relative_empty_xead_path
        empty_xead_path.write_text("<System/>\n", encoding="utf-8")
        empty_fixture_sha256_before = hashlib.sha256(empty_xead_path.read_bytes()).hexdigest()
        relative_missing_xead_path = "tei-studio-smoke/missing.xead"
        missing_xead_path = workspace_root / relative_missing_xead_path
        if missing_xead_path.exists():
            raise RuntimeError("The synthetic missing-XEAD path unexpectedly exists")
        facts["source_workspace_root"] = str(source_workspace)
        facts["workspace_root"] = str(workspace_root)
        facts["observe_xead_fixture"] = {"relative_path": relative_xead_path, "sha256_before": fixture_sha256_before, "scratch_workspace_copy": True, "source_workspace_modified": False}
        facts["observe_xead_empty_fixture"] = {"relative_path": relative_empty_xead_path, "sha256_before": empty_fixture_sha256_before, "scratch_workspace_copy": True}
        facts["observe_xead_missing_fixture"] = {"relative_path": relative_missing_xead_path, "exists": False, "scratch_workspace_copy": True}
        webview_baseline = find_processes("msedgewebview2.exe")
        baseline_pids = {int(item["ProcessId"]) for item in webview_baseline}
        facts["preexisting_webview2_count"] = len(baseline_pids)

        webview_debug_port = free_loopback_port()
        facts["webview_debugger_address"] = f"127.0.0.1:{webview_debug_port}"
        app_env = os.environ.copy()
        inherited_server_host = app_env.pop("WAILS_SERVER_HOST", None)
        facts["inherited_wails_server_host_cleared"] = inherited_server_host is not None
        inherited_devserver_url = app_env.pop("FRONTEND_DEVSERVER_URL", None)
        facts["frontend_devserver_url_unset"] = True
        facts["inherited_frontend_devserver_url_cleared"] = inherited_devserver_url is not None
        facts["webview_debug_port_argument"] = f"--webview-debug-port={webview_debug_port}"
        app_process = subprocess.Popen(
            [str(app_path), f"--workspace={workspace_root}", f"--webview-debug-port={webview_debug_port}"],
            stdout=app_stdout,
            stderr=app_stderr,
            env=app_env,
            text=True,
        )
        app_pid = app_process.pid
        app_handle = open_process_handle(app_pid)
        facts["app_pid"] = app_pid
        record("wails-process-launched", pid=app_pid)

        webview_deadline = time.monotonic() + 30
        webview_processes: list[dict[str, Any]] = []
        while time.monotonic() < webview_deadline:
            if app_process.poll() is not None:
                raise RuntimeError(f"Wails Desktop exited during WebView2 startup with code {app_process.returncode}")
            webview_processes = find_processes("msedgewebview2.exe")
            new_webview_pids = sorted(
                int(item["ProcessId"])
                for item in webview_processes
                if int(item["ProcessId"]) not in baseline_pids
                and (int(item["ParentProcessId"]) == app_pid or item.get("AppMarker") is True)
            )
            if new_webview_pids:
                break
            time.sleep(0.25)
        else:
            raise RuntimeError("No new msedgewebview2.exe process was observed after Wails WebView launch")
        facts["webview2_processes_during_session"] = webview_processes
        facts["new_webview2_pids"] = new_webview_pids
        record("webview2-process-observed", new_pids=new_webview_pids)

        webview_debug_url = f"http://localhost:{webview_debug_port}"
        facts["webview_debugger_url"] = webview_debug_url
        debugger_last_error: str | None = None
        webview_debug_deadline = time.monotonic() + 30
        while time.monotonic() < webview_debug_deadline:
            try:
                debugger_version = webdriver_request(webview_debug_url, "GET", "/json/version")
                if debugger_version.get("webSocketDebuggerUrl"):
                    facts["webview_debugger_version"] = debugger_version
                    break
                debugger_last_error = f"No webSocketDebuggerUrl in response: {debugger_version!r}"
            except Exception as exc:
                debugger_last_error = f"{type(exc).__name__}: {exc}"
            time.sleep(0.25)
        else:
            facts["webview_debugger_last_error"] = debugger_last_error
            facts["webview2_processes_at_debugger_timeout"] = find_processes("msedgewebview2.exe")
            raise RuntimeError(f"WebView2 remote debugging endpoint did not become ready: {debugger_last_error}")
        webview_processes = find_processes("msedgewebview2.exe")
        new_webview_pids = sorted(
            int(item["ProcessId"])
            for item in webview_processes
            if int(item["ProcessId"]) not in baseline_pids
            and (int(item["ParentProcessId"]) == app_pid or item.get("AppMarker") is True)
        )
        facts["webview2_processes_during_session"] = webview_processes
        facts["new_webview2_pids"] = new_webview_pids
        debugger_bindings = query_listening_sockets(webview_debug_port)
        facts["webview_debugger_bindings"] = debugger_bindings
        loopback_addresses = {"127.0.0.1", "::1", "::ffff:127.0.0.1"}
        if not debugger_bindings:
            raise RuntimeError("WebView2 debugger endpoint responded but no listening TCP binding was observable")
        exposed_addresses = sorted(
            {str(item.get("LocalAddress")) for item in debugger_bindings if str(item.get("LocalAddress")) not in loopback_addresses}
        )
        if exposed_addresses:
            raise RuntimeError(f"WebView2 debugger endpoint is not loopback-only: {exposed_addresses}")
        webview_process_by_pid = {int(item["ProcessId"]): item for item in webview_processes}
        new_webview_pid_set = set(new_webview_pids)
        listener_owner_pids = {int(item["OwningProcess"]) for item in debugger_bindings}
        if not listener_owner_pids or not listener_owner_pids.issubset(new_webview_pid_set):
            raise RuntimeError(
                f"WebView2 debugger listener owner is outside the observed new WebView2 process set: "
                f"owners={sorted(listener_owner_pids)}, webview_pids={new_webview_pids}"
            )
        listener_owner_processes = [webview_process_by_pid.get(pid) for pid in sorted(listener_owner_pids)]
        if any(item is None or str(item.get("RemoteDebugPort")) != str(webview_debug_port) for item in listener_owner_processes):
            raise RuntimeError(
                f"WebView2 debugger listener owner command line does not contain the chosen port "
                f"{webview_debug_port}: {listener_owner_processes!r}"
            )
        facts["webview_debugger_listener_owner_pids"] = sorted(listener_owner_pids)
        facts["webview_debugger_listener_owner_processes"] = listener_owner_processes
        record("webview2-debugger-ready", version=debugger_version)
        record("webview2-debugger-loopback-only", bindings=debugger_bindings)
        record("webview2-debugger-owned-by-matching-process", owner_pids=sorted(listener_owner_pids), port=webview_debug_port)

        port = free_loopback_port()
        base_url = f"http://127.0.0.1:{port}"
        facts["webdriver_url"] = base_url
        driver_env = os.environ.copy()
        driver_env["MSEDGEDRIVER_TELEMETRY_OPTOUT"] = "1"
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
                "debuggerAddress": f"localhost:{webview_debug_port}",
            },
        }
        session_response = webdriver_request(
            base_url,
            "POST",
            "/session",
            {"capabilities": {"alwaysMatch": capabilities}},
            timeout_seconds=30,
        )
        value = session_response.get("value", {})
        driver_session = session_response.get("sessionId") or value.get("sessionId")
        if not driver_session:
            raise RuntimeError(f"WebDriver did not return a session ID: {session_response}")
        facts["webdriver_session_capabilities"] = value.get("capabilities", {})
        record("wails-webview-session-created", session_id=driver_session, capabilities=facts["webdriver_session_capabilities"])

        facts["app_process"] = {"ProcessId": app_pid, "ExecutablePath": str(app_path)}
        record("wails-webview-session-attached", session_id=driver_session, app_pid=app_pid)

        inspect_script = """
const editor = document.querySelector('#editor .cm-content');
const selection = window.getSelection();
const wailsRpcRequests = performance.getEntriesByType('resource')
  .filter((item) => new URL(item.name, location.href).pathname === '/wails/runtime')
  .map((item) => ({path:'/wails/runtime', startTime:item.startTime, initiatorType:item.initiatorType}));
return {
  title: document.title,
  pathname: location.pathname,
  connection: document.querySelector('#connection')?.textContent ?? '',
  subjectTitle: document.querySelector('#subject-title')?.textContent ?? '',
  sourceStatus: document.querySelector('#source-status')?.textContent ?? '',
  notice: document.querySelector('#notice')?.textContent ?? '',
  diagnosticSummary: document.querySelector('#diagnostic-summary')?.textContent ?? '',
  editor: editor?.innerText ?? '',
  editorEditable: editor?.getAttribute('contenteditable') ?? null,
  editorFocused: !!editor && document.activeElement === editor,
  webviewDocumentFocused: document.hasFocus(),
  activeElement: document.activeElement ? {
    tagName: document.activeElement.tagName,
    id: document.activeElement.id,
    className: typeof document.activeElement.className === 'string' ? document.activeElement.className : '',
    ariaLabel: document.activeElement.getAttribute('aria-label')
  } : null,
  editorSelectionWithinEditor: !!selection && !!editor && editor.contains(selection.anchorNode) && editor.contains(selection.focusNode),
  editorSelection: selection && editor && editor.contains(selection.anchorNode) && editor.contains(selection.focusNode) ? selection.toString() : '',
  lastSubjectButton: Array.from(document.querySelectorAll('#subjects button.subject')).at(-1)?.querySelector('small')?.textContent ?? '',
  subjectButtonCount: document.querySelectorAll('#subjects button.subject').length,
  subjectButtonFocused: !!document.activeElement?.matches?.('#subjects button.subject'),
  focusedSubjectButton: document.activeElement?.matches?.('#subjects button.subject') ? document.activeElement.querySelector('small')?.textContent ?? '' : '',
  wailsRpcRequests,
  nativeKeyEvents: window.__teiNativeKeyEvents ? window.__teiNativeKeyEvents.slice() : null,
  graphLabel: document.querySelector('#graph svg')?.getAttribute('aria-label') ?? '',
  graphNodes: Array.from(document.querySelectorAll('#graph g[role="button"]')).map((item) => item.getAttribute('aria-label')),
  diagnosticRows: Array.from(document.querySelectorAll('#commands [aria-label="診断一覧"] li[data-severity]')).map((item) => ({severity:item.dataset.severity,text:item.textContent})),
  xeadStatus: document.querySelector('#xead-status')?.textContent ?? '',
  xeadResult: document.querySelector('#xead-result')?.innerText ?? '',
  xeadSections: Array.from(document.querySelectorAll('#xead-result .xead-table')).map((section) => ({heading:section.querySelector('h3')?.textContent ?? '',items:Array.from(section.querySelectorAll('li')).map((item) => item.textContent)})),
  workspaceOptions: Array.from(document.querySelectorAll('#workspace option')).map((item) => ({value:item.value,label:item.textContent})),
  assets: performance.getEntriesByType('resource').map((item) => new URL(item.name).pathname),
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
            if page_state.get("subjectTitle") == "sales.order" and page_state.get("sourceStatus", "").startswith("参照情報：") and page_state.get("diagnosticSummary") != "未検証":
                break
            time.sleep(0.25)
        facts["page_state"] = page_state
        write_json(evidence_dir / "page-state.json", page_state)
        screenshot_response = webdriver_request(base_url, "GET", f"/session/{driver_session}/screenshot")
        screenshot = screenshot_response.get("value")
        if not screenshot:
            raise RuntimeError("WebDriver screenshot response did not contain image data")
        (evidence_dir / "webview2-workbench-screen.png").write_bytes(base64.b64decode(screenshot))
        if page_state.get("pathname") != "/workbench/":
            raise RuntimeError(f"Desktop did not open the embedded Workbench route: {page_state.get('pathname')!r}")
        if page_state.get("connection") != "実Hostに接続":
            raise RuntimeError(f"Workbench did not connect to the real Host: {page_state.get('connection')!r}")
        if page_state.get("subjectTitle") != "sales.order":
            raise RuntimeError(f"Initial Catalog selection was not inspected: {page_state.get('subjectTitle')!r}")
        expected_options = [{"value": "workspace-1", "label": "workspace (workspace-1)"}]
        if page_state.get("workspaceOptions") != expected_options:
            raise RuntimeError(f"Initial workspace option was not rendered: {page_state.get('workspaceOptions')!r}")
        expected_assets = {"/workbench/main.js", "/workbench/style.css", "/wails/runtime.js"}
        if not expected_assets.issubset(set(page_state.get("assets", []))):
            raise RuntimeError(f"Workbench did not load all embedded runtime assets: {page_state.get('assets')!r}")
        if page_state.get("editorEditable") != "false" or page_state.get("graphLabel") != "読取り専用グラフ":
            raise RuntimeError("Workbench source or graph is not marked read-only")

        # Invoke the same registered Host methods through Wails' runtime and compare
        # their values with the actual WebView projection, rather than relying on a mock.
        bridge_script = """
const done = arguments[arguments.length - 1];
(async () => {
  const runtime = await import('/wails/runtime.js');
  const catalog = await runtime.Call.ByName('main.StudioBridge.CatalogWorkspace', 'workspace-1', '');
  const initialInspect = await runtime.Call.ByName('main.StudioBridge.InspectSubject', 'workspace-1', 'semantic-data', 'sales.order');
  const validation = await runtime.Call.ByName('main.StudioBridge.ValidateWorkspace', 'workspace-1');
  done({catalog, initialInspect, validation});
})().catch((error) => done({error: String(error)}));
"""
        bridge_response = webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/async", {"script": bridge_script, "args": []}, timeout_seconds=30).get("value", {})
        if bridge_response.get("error"):
            raise RuntimeError(f"Could not call the registered StudioBridge from the native WebView: {bridge_response['error']}")
        catalog_envelope = bridge_response.get("catalog", {})
        initial_envelope = bridge_response.get("initialInspect", {})
        validation_envelope = bridge_response.get("validation", {})
        for operation_name, envelope in (("CatalogWorkspace", catalog_envelope), ("InspectSubject", initial_envelope), ("ValidateWorkspace", validation_envelope)):
            if envelope.get("ok") is not True:
                raise RuntimeError(f"StudioBridge {operation_name} failed in the native WebView: {envelope!r}")
        initial_value = initial_envelope.get("value", {})
        if initial_value.get("effect") != "read-only" or initial_value.get("subject") != {"kind": "semantic-data", "id": "sales.order"}:
            raise RuntimeError(f"Initial Inspect response was unexpected: {initial_value!r}")
        displayed_canonical = json.loads(page_state.get("editor", "null"))
        if displayed_canonical != initial_value.get("canonical"):
            raise RuntimeError("Workbench canonical projection differs from the Host Inspect response")

        catalog_value = catalog_envelope.get("value", {})
        entries = catalog_value.get("entries", [])
        direct_children = [entry for entry in entries if entry.get("parent") == {"kind": "semantic-data", "id": "sales.order"} and entry.get("symbol") == "orderId"]
        if len(direct_children) != 1:
            raise RuntimeError(f"Catalog did not return one direct orderId child: {direct_children!r}")
        child = direct_children[0].get("subject", {})
        if "orderId" not in page_state.get("graphNodes", []):
            raise RuntimeError(f"Catalog child was not displayed as a selectable graph node: {page_state.get('graphNodes')!r}")
        click_script = """
const node = document.querySelector('#graph g[role="button"][aria-label="orderId"]');
if (!node) throw new Error('Catalog child graph node is missing');
node.dispatchEvent(new MouseEvent('click', { bubbles: true, cancelable: true, view: window }));
return true;
"""
        webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/sync", {"script": click_script, "args": []})
        graph_inspect_script = """
const done = arguments[arguments.length - 1];
const [kind, subjectId] = arguments;
(async () => {
 const runtime = await import('/wails/runtime.js');
 done(await runtime.Call.ByName('main.StudioBridge.InspectSubject', 'workspace-1', kind, subjectId));
})().catch((error) => done({error:String(error)}));
"""
        graph_response = webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/async", {"script": graph_inspect_script, "args": [child.get("kind"), child.get("id")]}, timeout_seconds=30).get("value", {})
        if graph_response.get("ok") is not True or graph_response.get("value", {}).get("effect") != "read-only":
            raise RuntimeError(f"Graph Inspect did not return a read-only projection: {graph_response!r}")
        graph_state = {}
        graph_deadline = time.monotonic() + 10
        while time.monotonic() < graph_deadline:
            graph_state = webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/sync", {"script": inspect_script, "args": []}).get("value", {})
            if graph_state.get("subjectTitle") == child.get("id"):
                break
            time.sleep(0.2)
        if graph_state.get("subjectTitle") != child.get("id") or json.loads(graph_state.get("editor", "null")) != graph_response.get("value", {}).get("canonical"):
            raise RuntimeError("Graph selection did not render the Host Inspect canonical projection")

        validate_click = """
const button = document.querySelector('#commands button[data-command-id="validate"]');
if (!button || button.disabled) throw new Error('Validate button is unavailable');
button.click();
return true;
"""
        webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/sync", {"script": validate_click, "args": []})
        validation_findings = []
        value = validation_envelope.get("value", {})
        diagnostics = ([] if value.get("Direct") is None else [value["Direct"]]) + [item["diagnostic"] for item in (value.get("Diagnostics") or [])]
        for diagnostic in diagnostics:
            validation_findings.append({"severity": diagnostic["severity"], "text": f"{diagnostic['severity']}: {diagnostic['code']}: {diagnostic['message']}"})
        expected_counts = {severity: sum(item["severity"] == severity for item in validation_findings) for severity in ("error", "warning")}
        expected_summary = f"エラー {expected_counts['error']}・警告 {expected_counts['warning']}"
        validate_state = {}
        validate_deadline = time.monotonic() + 12
        while time.monotonic() < validate_deadline:
            validate_state = webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/sync", {"script": inspect_script, "args": []}).get("value", {})
            if validate_state.get("diagnosticSummary") == expected_summary and validate_state.get("diagnosticRows") == validation_findings:
                break
            time.sleep(0.2)
        if validate_state.get("diagnosticSummary") != expected_summary or validate_state.get("diagnosticRows") != validation_findings:
            raise RuntimeError(f"Workbench diagnostics differ from the Host Validate response: expected={validation_findings!r}, actual={validate_state.get('diagnosticRows')!r}")

        # Exercise the native WebView2 keyboard path against the read-only Editor.
        # W3C key actions are used here instead of synthetic DOM KeyboardEvents.
        last_subject = page_state.get("lastSubjectButton", "")
        subject_ids = [entry.get("id") for entry in entries]
        if not last_subject or not subject_ids or page_state.get("subjectButtonCount") != len(subject_ids):
            raise RuntimeError("Workbench did not expose a last subject button for the native keyboard check")
        expected_last_subject = last_subject.rsplit(" \u00b7 ", 1)[-1]
        focus_last_subject_script = """
const button = document.querySelector('#subjects button.subject:last-of-type');
if (!button) throw new Error('Last subject button is missing');
button.focus();
return {focused:document.activeElement === button, label:button.querySelector('small')?.textContent ?? ''};
"""
        focused_subject = webdriver_request(
            base_url,
            "POST",
            f"/session/{driver_session}/execute/sync",
            {"script": focus_last_subject_script, "args": []},
        ).get("value", {})
        if focused_subject.get("focused") is not True or focused_subject.get("label") != last_subject:
            raise RuntimeError(f"Could not focus the last subject button as the Tab starting point: {focused_subject!r}")
        keyboard_baseline = webdriver_request(
            base_url,
            "POST",
            f"/session/{driver_session}/execute/sync",
            {"script": inspect_script, "args": []},
        ).get("value", {})
        if keyboard_baseline.get("subjectButtonFocused") is not True or keyboard_baseline.get("focusedSubjectButton") != last_subject:
            raise RuntimeError(f"Last subject button focus was not observable before Tab: {keyboard_baseline!r}")
        if keyboard_baseline.get("subjectButtonCount") != len(subject_ids):
            raise RuntimeError(f"Displayed subject buttons do not match the Host catalog: {keyboard_baseline.get('subjectButtonCount')} != {len(subject_ids)}")
        keyboard_editor_text = keyboard_baseline.get("editor", "")
        if not keyboard_editor_text:
            raise RuntimeError("The focused last subject has no Editor text to protect")

        pre_rpc_deadline = time.monotonic() + 8
        previous_rpc_requests: list[dict[str, Any]] | None = None
        stable_rpc_samples = 0
        pre_calibration_state: dict[str, Any] = {}
        while time.monotonic() < pre_rpc_deadline:
            pre_calibration_state = webdriver_request(
                base_url,
                "POST",
                f"/session/{driver_session}/execute/sync",
                {"script": inspect_script, "args": []},
            ).get("value", {})
            pre_calibration_requests = pre_calibration_state.get("wailsRpcRequests")
            if not isinstance(pre_calibration_requests, list):
                raise RuntimeError("WebView2 did not expose Wails runtime RPC Resource Timing entries before calibration")
            if pre_calibration_requests == previous_rpc_requests:
                stable_rpc_samples += 1
            else:
                previous_rpc_requests = pre_calibration_requests
                stable_rpc_samples = 0
            if stable_rpc_samples >= 3:
                break
            time.sleep(0.2)
        if stable_rpc_samples < 3:
            raise RuntimeError("Wails RPC Resource Timing entries did not settle before calibration")
        pre_calibration_requests = pre_calibration_state["wailsRpcRequests"]

        # The Wails runtime export is not writable in the packaged WebView2
        # module. Calibrate Resource Timing with a known read-only Host call,
        # then require the observed /wails/runtime request list to stay fixed.
        calibration_script = """
const done = arguments[arguments.length - 1];
import('/wails/runtime.js').then(async (runtime) => {
  await runtime.Call.ByName('main.StudioBridge.ListWorkspaces');
  done(true);
}).catch((error) => done({error:String(error)}));
"""
        calibration_result = webdriver_request(
            base_url,
            "POST",
            f"/session/{driver_session}/execute/async",
            {"script": calibration_script, "args": []},
            timeout_seconds=30,
        ).get("value", {})
        if calibration_result is not True:
            raise RuntimeError(f"Could not calibrate Wails RPC Resource Timing with a known read-only Host call: {calibration_result!r}")
        stable_rpc_deadline = time.monotonic() + 8
        previous_rpc_requests = None
        stable_rpc_samples = 0
        keyboard_baseline: dict[str, Any] = {}
        while time.monotonic() < stable_rpc_deadline:
            keyboard_baseline = webdriver_request(
                base_url,
                "POST",
                f"/session/{driver_session}/execute/sync",
                {"script": inspect_script, "args": []},
            ).get("value", {})
            rpc_requests = keyboard_baseline.get("wailsRpcRequests")
            if not isinstance(rpc_requests, list):
                raise RuntimeError("WebView2 did not expose Wails runtime RPC Resource Timing entries")
            if rpc_requests == previous_rpc_requests:
                stable_rpc_samples += 1
            else:
                previous_rpc_requests = rpc_requests
                stable_rpc_samples = 0
            if stable_rpc_samples >= 3:
                break
            time.sleep(0.2)
        if stable_rpc_samples < 3:
            raise RuntimeError("Wails RPC Resource Timing entries did not settle before the keyboard sequence")
        baseline_rpc_requests = keyboard_baseline["wailsRpcRequests"]
        if len(baseline_rpc_requests) <= len(pre_calibration_requests):
            raise RuntimeError("Known read-only Host RPC did not appear in WebView2 Resource Timing")

        key_event_listener_script = """
const editor = document.querySelector('#editor .cm-content');
if (!editor) throw new Error('Editor is missing before native key event observation');
window.__teiNativeKeyEvents = [];
document.addEventListener('keydown', (event) => {
  const subject = event.target instanceof Element ? event.target.closest('#subjects button.subject') : null;
  window.__teiNativeKeyEvents.push({
    key:event.key,
    trusted:event.isTrusted,
    targetIsEditor:editor.contains(event.target),
    targetSubject:subject?.querySelector('small')?.textContent ?? ''
  });
}, true);
return true;
"""
        listener_ready = webdriver_request(
            base_url,
            "POST",
            f"/session/{driver_session}/execute/sync",
            {"script": key_event_listener_script, "args": []},
        ).get("value")
        if listener_ready is not True:
            raise RuntimeError("Could not install the passive native key event observer")

        key_events = {
            "tab": [{"type": "keyDown", "value": "\uE004"}, {"type": "keyUp", "value": "\uE004"}],
            "home": [{"type": "keyDown", "value": "\uE011"}, {"type": "keyUp", "value": "\uE011"}],
            "shift_end": [
                {"type": "keyDown", "value": "\uE008"},
                {"type": "keyDown", "value": "\uE007"},
                {"type": "keyUp", "value": "\uE007"},
                {"type": "keyUp", "value": "\uE008"},
            ],
            "type_x": [{"type": "keyDown", "value": "x"}, {"type": "keyUp", "value": "x"}],
            "backspace": [{"type": "keyDown", "value": "\uE003"}, {"type": "keyUp", "value": "\uE003"}],
            "delete": [{"type": "keyDown", "value": "\uE017"}, {"type": "keyUp", "value": "\uE017"}],
        }
        workspace_before_keys = workspace_snapshot(workspace_root)
        xead_before_keys = hashlib.sha256(xead_path.read_bytes()).hexdigest()
        empty_xead_before_keys = hashlib.sha256(empty_xead_path.read_bytes()).hexdigest()
        keyboard_steps: list[dict[str, Any]] = []
        observed_key_event_count = 0
        expected_key_events = {
            "Tab-to-Editor focus": {"Tab"},
            "Home": {"Home"},
            "Shift+End selection": {"Shift", "End"},
            "type_x": {"x"},
            "backspace": {"Backspace"},
            "delete": {"Delete"},
            "settled": set(),
        }

        def keyboard_state_after(step: str) -> dict[str, Any]:
            nonlocal observed_key_event_count
            state = webdriver_request(
                base_url,
                "POST",
                f"/session/{driver_session}/execute/sync",
                {"script": inspect_script, "args": []},
            ).get("value", {})
            step_evidence = {
                "step": step,
                "editor_focused": state.get("editorFocused") is True,
                "webview_document_focused": state.get("webviewDocumentFocused") is True,
                "editor_text_sha256": hashlib.sha256(str(state.get("editor", "")).encode("utf-8")).hexdigest(),
                "selected_text": state.get("editorSelection", ""),
                "wails_rpc_request_count": len(state.get("wailsRpcRequests", [])),
                "native_key_events": state.get("nativeKeyEvents"),
                "route": state.get("pathname"),
                "host_connection": state.get("connection"),
                "active_element": state.get("activeElement"),
            }
            keyboard_steps.append(step_evidence)
            facts["native_webview2_editor_keyboard"] = {"selected_subject": expected_last_subject, "steps": keyboard_steps}
            page_state["nativeWebView2EditorKeyboard"] = facts["native_webview2_editor_keyboard"]
            write_json(evidence_dir / "page-state.json", page_state)
            if state.get("pathname") != "/workbench/" or state.get("connection") != "\u5b9fHost\u306b\u63a5\u7d9a":
                raise RuntimeError(f"Workbench route or real Host connection changed after {step}: {state!r}")
            if state.get("editor") != keyboard_editor_text or state.get("subjectTitle") != expected_last_subject:
                raise RuntimeError(f"Read-only Editor projection changed after {step}")
            if not state.get("editorFocused"):
                raise RuntimeError(f"Native WebView2 Editor lost focus after {step}: {state.get('activeElement')!r}")
            if not state.get("webviewDocumentFocused"):
                raise RuntimeError(f"Native WebView2 document lost focus after {step}")
            rpc_requests = state.get("wailsRpcRequests")
            if not isinstance(rpc_requests, list):
                raise RuntimeError(f"Wails RPC Resource Timing entries became unavailable after {step}")
            if rpc_requests != baseline_rpc_requests:
                raise RuntimeError(f"Workbench made an additional Wails runtime RPC request during the keyboard sequence after {step}: {rpc_requests!r}")
            key_events_seen = state.get("nativeKeyEvents")
            if not isinstance(key_events_seen, list):
                raise RuntimeError(f"Native WebView2 key event observations became unavailable after {step}")
            new_key_events = key_events_seen[observed_key_event_count:]
            observed_key_event_count = len(key_events_seen)
            expected_events = expected_key_events.get(step)
            if expected_events is not None:
                delivered_keys = {item.get("key") for item in new_key_events if item.get("trusted") is True}
                if not expected_events.issubset(delivered_keys):
                    raise RuntimeError(f"Expected trusted key events were not delivered after {step}: expected={sorted(expected_events)!r}, observed={new_key_events!r}")
                if step == "Tab-to-Editor focus":
                    if any(item.get("targetSubject") != last_subject for item in new_key_events if item.get("key") == "Tab"):
                        raise RuntimeError("Tab did not originate from the focused last subject button")
                elif any(not item.get("targetIsEditor") for item in new_key_events if item.get("key") in expected_events):
                    raise RuntimeError(f"Editor keyboard input was not delivered to the Editor DOM: {new_key_events!r}")
            return state

        webdriver_key_actions(base_url, driver_session, key_events["tab"])
        tab_state = keyboard_state_after("Tab-to-Editor focus")
        if tab_state.get("editorEditable") != "false":
            raise RuntimeError(f"Editor is not explicitly read-only before keyboard input: {tab_state.get('editorEditable')!r}")
        if not tab_state.get("editorFocused"):
            raise RuntimeError("Tab from the last subject button did not focus the Editor")

        webdriver_key_actions(base_url, driver_session, key_events["home"])
        home_state = keyboard_state_after("Home")
        if home_state.get("editorSelection"):
            raise RuntimeError("Home unexpectedly left a non-empty DOM selection")
        webdriver_key_actions(base_url, driver_session, key_events["shift_end"])
        selection_state = keyboard_state_after("Shift+End selection")
        selected_text = selection_state.get("editorSelection", "")
        if not selected_text or not selection_state.get("editorSelectionWithinEditor"):
            raise RuntimeError("Home followed by Shift+End did not create an observable Editor selection")

        final_keyboard_state: dict[str, Any] = selection_state
        for step in ("type_x", "backspace", "delete"):
            webdriver_key_actions(base_url, driver_session, key_events[step])
            final_keyboard_state = keyboard_state_after(step)
        time.sleep(0.6)
        final_keyboard_state = keyboard_state_after("settled")

        workspace_after_keys = workspace_snapshot(workspace_root)
        xead_after_keys = hashlib.sha256(xead_path.read_bytes()).hexdigest()
        empty_xead_after_keys = hashlib.sha256(empty_xead_path.read_bytes()).hexdigest()
        if workspace_after_keys != workspace_before_keys:
            raise RuntimeError(f"Scratch Workspace changed during the native keyboard sequence: before={workspace_before_keys!r}, after={workspace_after_keys!r}")
        if xead_after_keys != xead_before_keys or empty_xead_after_keys != empty_xead_before_keys:
            raise RuntimeError("An ObserveXead fixture changed during the native keyboard sequence")
        facts["native_webview2_editor_keyboard"] = {
            "route": "/workbench/",
            "host_connection": "\u5b9fHost\u306b\u63a5\u7d9a",
            "editor_read_only": True,
            "editor_text_sha256_before": hashlib.sha256(keyboard_editor_text.encode("utf-8")).hexdigest(),
            "editor_text_sha256_after": hashlib.sha256(final_keyboard_state["editor"].encode("utf-8")).hexdigest(),
            "selected_text_sha256": hashlib.sha256(selected_text.encode("utf-8")).hexdigest(),
            "selected_text_length": len(selected_text),
            "steps": keyboard_steps,
            "wails_rpc_request_count_before": len(baseline_rpc_requests),
            "wails_rpc_request_count_after": len(final_keyboard_state.get("wailsRpcRequests", [])),
            "new_wails_rpc_requests": [],
            "workspace_before": workspace_before_keys,
            "workspace_after": workspace_after_keys,
            "fixture_sha256_before": {"observe_xead": xead_before_keys, "observe_xead_empty": empty_xead_before_keys},
            "fixture_sha256_after": {"observe_xead": xead_after_keys, "observe_xead_empty": empty_xead_after_keys},
        }
        if facts["native_webview2_editor_keyboard"]["editor_text_sha256_after"] != facts["native_webview2_editor_keyboard"]["editor_text_sha256_before"]:
            raise RuntimeError("Read-only Editor text hash changed during the native keyboard sequence")
        page_state["nativeWebView2EditorKeyboard"] = facts["native_webview2_editor_keyboard"]
        write_json(evidence_dir / "page-state.json", page_state)
        record(
            "native-webview2-read-only-editor-keyboard-verified",
            selected_subject=expected_last_subject,
            editor_text_sha256=facts["native_webview2_editor_keyboard"]["editor_text_sha256_before"],
            selected_text_sha256=facts["native_webview2_editor_keyboard"]["selected_text_sha256"],
            workspace_sha256=workspace_before_keys["sha256"],
            fixture_unchanged=True,
            wails_rpc_request_count=len(baseline_rpc_requests),
            new_wails_rpc_requests=0,
            steps=["Tab", "Home", "Shift+End", "type x", "Backspace", "Delete"],
        )
        save_keyboard_response = webdriver_request(base_url, "GET", f"/session/{driver_session}/screenshot")
        save_keyboard_image = save_keyboard_response.get("value")
        if not save_keyboard_image:
            raise RuntimeError("WebDriver screenshot did not contain the native keyboard-check image")
        (evidence_dir / "webview2-editor-keyboard-screen.png").write_bytes(base64.b64decode(save_keyboard_image))

        # Exercise the existing relative-path input and click handler in the
        # actual Wails WebView using a fixture in a disposable workspace copy.
        input_response = webdriver_request(base_url, "POST", f"/session/{driver_session}/element", {"using": "css selector", "value": "#xead-path"}).get("value", {})
        button_response = webdriver_request(base_url, "POST", f"/session/{driver_session}/element", {"using": "css selector", "value": "#observe-xead"}).get("value", {})
        input_id = input_response.get("element-6066-11e4-a52e-4f735466cecf")
        button_id = button_response.get("element-6066-11e4-a52e-4f735466cecf")
        if not input_id or not button_id:
            raise RuntimeError(f"ObserveXead controls were not found: input={input_response!r}, button={button_response!r}")
        webdriver_request(base_url, "POST", f"/session/{driver_session}/element/{input_id}/value", {"text": relative_xead_path, "value": list(relative_xead_path)})
        webdriver_request(base_url, "POST", f"/session/{driver_session}/element/{button_id}/click", {})
        xead_state: dict[str, Any] = {}
        xead_deadline = time.monotonic() + 15
        while time.monotonic() < xead_deadline:
            xead_state = webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/sync", {"script": inspect_script, "args": []}).get("value", {})
            if xead_state.get("xeadStatus", "").startswith("観察完了："):
                break
            time.sleep(0.2)
        if not xead_state.get("xeadStatus", "").startswith("観察完了："):
            raise RuntimeError(f"ObserveXead UI did not complete: {xead_state.get('xeadStatus')!r}")

        observe_script = """
const done = arguments[arguments.length - 1];
const [relativePath] = arguments;
import('/wails/runtime.js').then(async (runtime) => {
  done(await runtime.Call.ByName('main.StudioBridge.ObserveXead', 'workspace-1', relativePath));
}).catch((error) => done({error: String(error)}));
"""
        observe_envelope = webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/async", {"script": observe_script, "args": [relative_xead_path]}, timeout_seconds=30).get("value", {})
        if observe_envelope.get("error") or observe_envelope.get("ok") is not True:
            raise RuntimeError(f"ObserveXead Host Bridge failed in native WebView: {observe_envelope!r}")
        observation = observe_envelope.get("value", {})
        expected_diagnostics = [
            {"code": "unsupported-attribute", "element": "Instance", "attribute": "Text", "occurrences": 1},
            {"code": "unsupported-attribute", "element": "System", "attribute": "DatabasePassword", "occurrences": 1},
            {"code": "unsupported-attribute", "element": "TableField", "attribute": "Default", "occurrences": 1},
            {"code": "unsupported-element", "element": "Instance", "occurrences": 1},
            {"code": "unsupported-element", "element": "PrivateValue", "occurrences": 1},
        ]
        expected_table = {
            "source": {"sourceKind": "Table", "sourceID": "T1"},
            "name": "Order",
            "fields": [{"source": {"sourceKind": "TableField", "sourceID": "F1"}, "name": "OrderID", "dataType": {"sourceKind": "DataType", "sourceID": "D1"}}],
        }
        if set(observation) != {"schemaVersion", "effect", "observation", "authorityLimits"} or observation.get("schemaVersion") != "tei.operations.xead-observation/v0" or observation.get("effect") != "read-only":
            raise RuntimeError(f"ObserveXead did not return the read-only observation contract: {observation!r}")
        projected = observation.get("observation", {})
        if projected.get("tables") != [expected_table] or projected.get("diagnostics") != expected_diagnostics:
            raise RuntimeError(f"ObserveXead response differed from the fixed allowlisted projection: {observation!r}")
        expected_authority_limits = {
            "xead-observation-is-not-canonical-meaning",
            "xead-source-identifiers-are-not-canonical-identities",
            "xead-observation-does-not-authorize-edits-or-execution",
            "unsupported-source-constructs-remain-unresolved",
        }
        if set(observation.get("authorityLimits", [])) != expected_authority_limits:
            raise RuntimeError(f"ObserveXead authority limits differed: {observation.get('authorityLimits')!r}")
        response_text = json.dumps(observe_envelope, ensure_ascii=False, sort_keys=True)
        displayed_xead = str(xead_state.get("xeadResult", ""))
        secrets = ("synthetic-system-secret", "synthetic-default-secret", "synthetic-instance-secret", "synthetic-unknown-secret")
        for secret in secrets:
            if secret in response_text or secret in displayed_xead or secret in str(xead_state.get("xeadStatus", "")):
                raise RuntimeError(f"ObserveXead exposed a synthetic source value: {secret}")
        for visible in ("Order", "T1", "OrderID", "F1", "D1", "unsupported-attribute", "unsupported-element", "Default", "Instance", "PrivateValue"):
            if visible not in displayed_xead:
                raise RuntimeError(f"ObserveXead did not display expected projection detail {visible!r}: {displayed_xead!r}")
        expected_sections = [
            {"heading": "Order · 元ID Table:T1", "items": ["OrderID · 元ID TableField:F1 · 型参照ID DataType:D1"]},
            {"heading": "観察診断", "items": [
                "コード unsupported-attribute · 要素 Instance · 属性 Text · 該当 1件",
                "コード unsupported-attribute · 要素 System · 属性 DatabasePassword · 該当 1件",
                "コード unsupported-attribute · 要素 TableField · 属性 Default · 該当 1件",
                "コード unsupported-element · 要素 Instance · 該当 1件",
                "コード unsupported-element · 要素 PrivateValue · 該当 1件",
            ]},
        ]
        if xead_state.get("xeadSections") != expected_sections:
            raise RuntimeError(f"ObserveXead UI projection differed from the Host DTO: {xead_state.get('xeadSections')!r}")
        if "synthetic-" in response_text or "text" in observation:
            raise RuntimeError(f"ObserveXead response contained an unapproved source value or field: {response_text}")
        fixture_sha256_after = hashlib.sha256(xead_path.read_bytes()).hexdigest()
        if fixture_sha256_after != fixture_sha256_before:
            raise RuntimeError("ObserveXead changed the fixed fixture bytes")

        def save_xead_screen(filename: str, description: str) -> None:
            response = webdriver_request(base_url, "GET", f"/session/{driver_session}/screenshot")
            encoded = response.get("value")
            if not encoded:
                raise RuntimeError(f"WebDriver screenshot response did not contain image data {description}")
            (evidence_dir / filename).write_bytes(base64.b64decode(encoded))

        def observe_ui_path(
            relative_path: str, expected_status: str, screenshot_name: str
        ) -> dict[str, Any]:
            webdriver_request(base_url, "POST", f"/session/{driver_session}/element/{input_id}/clear", {})
            webdriver_request(base_url, "POST", f"/session/{driver_session}/element/{input_id}/value", {"text": relative_path, "value": list(relative_path)})
            webdriver_request(base_url, "POST", f"/session/{driver_session}/element/{button_id}/click", {})
            deadline = time.monotonic() + 15
            state: dict[str, Any] = {}
            while time.monotonic() < deadline:
                state = webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/sync", {"script": inspect_script, "args": []}).get("value", {})
                if state.get("xeadStatus") == expected_status:
                    save_xead_screen(screenshot_name, f"after reaching {expected_status!r}")
                    return state
                time.sleep(0.2)
            save_xead_screen(screenshot_name, f"while waiting for {expected_status!r}")
            raise RuntimeError(f"ObserveXead UI did not reach {expected_status!r}: {state.get('xeadStatus')!r}")

        def observe_host_path(relative_path: str) -> dict[str, Any]:
            script = """
const done = arguments[arguments.length - 1];
const [relativePath] = arguments;
import('/wails/runtime.js').then(async (runtime) => {
  done(await runtime.Call.ByName('main.StudioBridge.ObserveXead', 'workspace-1', relativePath));
}).catch((error) => done({error: String(error)}));
"""
            return webdriver_request(base_url, "POST", f"/session/{driver_session}/execute/async", {"script": script, "args": [relative_path]}, timeout_seconds=30).get("value", {})

        save_xead_screen("webview2-observe-xead-screen.png", "after successful observation")
        empty_status = "観察完了：テーブル 0件、項目 0件、診断分類 0件"
        empty_state = observe_ui_path(
            relative_empty_xead_path,
            empty_status,
            "webview2-observe-xead-empty-screen.png",
        )
        if empty_state.get("xeadResult") != "表示できるテーブルまたは診断はありません。" or empty_state.get("xeadSections"):
            raise RuntimeError(f"Empty ObserveXead success did not render its empty state: {empty_state!r}")
        empty_envelope = observe_host_path(relative_empty_xead_path)
        empty_value = empty_envelope.get("value", {})
        empty_projection = empty_value.get("observation", {}) if isinstance(empty_value, dict) else {}
        if empty_envelope.get("error") or empty_envelope.get("ok") is not True or "failure" in empty_envelope:
            raise RuntimeError(f"Empty ObserveXead was not a successful Host response: {empty_envelope!r}")
        if (
            not isinstance(empty_value, dict)
            or empty_value.get("effect") != "read-only"
            or not isinstance(empty_projection, dict)
            or empty_projection.get("tables", []) != []
            or empty_projection.get("diagnostics", []) != []
        ):
            raise RuntimeError(f"Empty ObserveXead Host projection was not empty and read-only: {empty_envelope!r}")
        empty_fixture_sha256_after = hashlib.sha256(empty_xead_path.read_bytes()).hexdigest()
        if empty_fixture_sha256_after != empty_fixture_sha256_before:
            raise RuntimeError("ObserveXead changed the empty fixture bytes")

        missing_status = "観察できませんでした：XEAD_SOURCE_READ_FAILED"
        missing_state = observe_ui_path(
            relative_missing_xead_path,
            missing_status,
            "webview2-observe-xead-read-failure-screen.png",
        )
        if missing_state.get("xeadResult") or missing_state.get("xeadSections"):
            raise RuntimeError(f"ObserveXead read failure retained an earlier result: {missing_state!r}")
        if (
            relative_missing_xead_path in missing_state.get("xeadStatus", "")
            or relative_missing_xead_path in missing_state.get("xeadResult", "")
        ):
            raise RuntimeError(f"ObserveXead UI echoed its relative input path in the failure: {missing_state!r}")
        missing_envelope = observe_host_path(relative_missing_xead_path)
        if missing_xead_path.exists():
            raise RuntimeError("ObserveXead created the missing source path during UI and direct Host calls")
        host_failure = missing_envelope.get("failure")
        missing_response_text = json.dumps(missing_envelope, ensure_ascii=False, sort_keys=True)
        if (
            missing_envelope.get("error")
            or missing_envelope.get("ok") is not False
            or not isinstance(host_failure, dict)
            or host_failure.get("code") != "XEAD_SOURCE_READ_FAILED"
            or "value" in missing_envelope
        ):
            raise RuntimeError(f"Missing ObserveXead path did not return the expected Host failure: {missing_envelope!r}")
        if relative_missing_xead_path in missing_response_text or str(workspace_root) in missing_response_text:
            raise RuntimeError(f"ObserveXead Host failure echoed a source path: {missing_envelope!r}")

        facts["observe_xead"] = {
            "workspace_id": "workspace-1", "relative_path": relative_xead_path,
            "schema_version": observation["schemaVersion"], "effect": observation["effect"],
            "table_count": len(projected.get("tables", [])),
            "field_count": sum(len(table.get("fields", [])) for table in projected.get("tables", [])),
            "diagnostic_count": len(projected.get("diagnostics", [])),
            "synthetic_values_redacted": True,
            "fixture_sha256_before": fixture_sha256_before, "fixture_sha256_after": fixture_sha256_after,
        }
        facts["observe_xead_empty"] = {
            "relative_path": relative_empty_xead_path, "status": empty_state["xeadStatus"],
            "tables": 0, "fields": 0, "diagnostics": 0,
            "fixture_sha256_before": empty_fixture_sha256_before, "fixture_sha256_after": empty_fixture_sha256_after,
        }
        facts["observe_xead_read_failure"] = {
            "relative_path": relative_missing_xead_path, "status": missing_state["xeadStatus"],
            "failure_code": host_failure["code"], "relative_path_echoed": False,
            "earlier_result_cleared": True,
        }
        page_state.update({"graphSelection": graph_state, "validationDisplay": validate_state, "bridgeEnvelopes": bridge_response, "graphInspectResponse": graph_response, "observeXeadUi": xead_state, "observeXeadBridge": observe_envelope, "observeXeadEmptyUi": empty_state, "observeXeadEmptyBridge": empty_envelope, "observeXeadMissingUi": missing_state, "observeXeadMissingBridge": missing_envelope})
        write_json(evidence_dir / "page-state.json", page_state)
        record("workbench-catalog-inspect-validate-verified", pathname=page_state["pathname"], catalog_entries=len(entries), initial_subject="sales.order", inspected_subject=child.get("id"), validation_summary=expected_summary, displayed_findings=len(validation_findings))
        record("workbench-observe-xead-ui-and-bridge-verified", workspace_id="workspace-1", relative_path=relative_xead_path, effect="read-only", tables=1, fields=1, diagnostics=len(expected_diagnostics), synthetic_values_redacted=True, fixture_unchanged=True)
        record("workbench-observe-xead-empty-success-verified", relative_path=relative_empty_xead_path, tables=0, fields=0, diagnostics=0, empty_state_rendered=True, fixture_unchanged=True)
        record("workbench-observe-xead-missing-source-failure-verified", failure_code=host_failure["code"], relative_path_echoed=False, earlier_result_cleared=True)

    except Exception as exc:  # Keep a failure artifact even when setup cannot reach the Wails app.
        failure = f"{type(exc).__name__}: {exc}"
        facts["failure"] = failure
        (evidence_dir / "failure.txt").write_text(traceback.format_exc(), encoding="utf-8")
        record("smoke-failed", reason=failure)
    finally:
        if driver_session:
            facts["session_delete_attempted"] = True
            try:
                response = webdriver_request(f"http://127.0.0.1:{facts['webdriver_url'].rsplit(':', 1)[1]}", "DELETE", f"/session/{driver_session}")
                facts["webdriver_delete_session_response"] = response
                facts["session_deleted"] = True
                record("webdriver-session-closed", response=response)
            except Exception as exc:
                facts["webdriver_delete_session_error"] = f"{type(exc).__name__}: {exc}"
                failure = failure or facts["webdriver_delete_session_error"]

        if app_pid is not None and app_process is not None and app_process.poll() is None:
            try:
                closed_windows = request_window_close(app_pid)
                facts["app_window_close_requested"] = True
                facts["app_windows_closed_by_wm_close"] = closed_windows
                record("wails-window-close-requested", pid=app_pid, windows=closed_windows)
            except Exception as exc:
                facts["app_window_close_error"] = f"{type(exc).__name__}: {exc}"
                failure = failure or facts["app_window_close_error"]

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
                elif not facts["session_deleted"]:
                    facts["normal_exit"] = "FAIL_SESSION_NOT_DELETED"
                    failure = failure or "Wails Desktop exited without successful WebDriver session deletion evidence"
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
        if scratch_workspace is not None:
            try:
                scratch_workspace.cleanup()
                facts["scratch_workspace_removed"] = True
            except Exception as exc:
                facts["scratch_workspace_cleanup_error"] = f"{type(exc).__name__}: {exc}"
                failure = failure or facts["scratch_workspace_cleanup_error"]
        driver_stdout.close()
        driver_stderr.close()
        app_stdout.close()
        app_stderr.close()

        facts["result"] = "FAIL" if failure else (
            "PASS_NATIVE_WEBVIEW2_EDITOR_KEYBOARD_AND_OBSERVEXEAD"
            if facts.get("native_webview2_editor_keyboard") and facts.get("observe_xead")
            else "PASS_NATIVE_WEBVIEW_WORKBENCH_READ_ONLY"
        )
        if failure:
            facts["failure"] = failure
            if not (evidence_dir / "failure.txt").exists():
                (evidence_dir / "failure.txt").write_text(failure + "\n", encoding="utf-8")
        write_json(facts_path, facts)
        print(f"SMOKE_RESULT={facts['result']}", flush=True)
    return 1 if failure else 0


if __name__ == "__main__":
    raise SystemExit(main())
