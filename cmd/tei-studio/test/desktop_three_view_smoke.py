"""Open the three-view page in the Wails Desktop WebView2 window and walk Business -> Function -> CRUD -> Source.

The launch, WebDriver and shutdown helpers are those of desktop_webview_smoke.py.
This script adds one read-only flow on the synthetic three-view Workspace and
records what the native window showed next to what the registered Bridge
method returned. It does not show that a real X-TEA design source is read
correctly, and a pass is not an Owner acceptance of the screen.
"""

from __future__ import annotations

import base64
import hashlib
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
import traceback
import urllib.parse
from pathlib import Path
from typing import Any

import desktop_webview_smoke as smoke

SOURCE_PATH = "design/order.xead"

PAGE_STATE_SCRIPT = """
const text = (selector) => document.querySelector(selector)?.innerText ?? '';
const labels = (selector) => Array.from(document.querySelectorAll(selector + ' button')).map((item) => item.textContent);
return {
  pathname: location.pathname,
  readyState: document.readyState,
  navigateDisabled: document.querySelector('#navigate')?.disabled ?? null,
  workspaceOptions: Array.from(document.querySelectorAll('#workspace option')).map((item) => item.value),
  state: text('#state'),
  coverage: text('#coverage'),
  data: labels('#view-data'),
  function: labels('#view-function'),
  business: labels('#view-business'),
  focus: text('#focus'),
  focusHeading: document.querySelector('#focus h3')?.textContent ?? '',
  pin: text('#pin-state'),
};
"""

# Clicks the first button under `scope` whose text starts with `prefix`.
CLICK_SCRIPT = """
const [scope, prefix] = arguments;
const buttons = Array.from(document.querySelectorAll(scope + ' button')).filter((item) => item.textContent.startsWith(prefix));
if (buttons.length !== 1) throw new Error('expected one button for ' + prefix + ' under ' + scope + ', found ' + buttons.length);
if (buttons[0].disabled) throw new Error('button is disabled: ' + prefix);
buttons[0].click();
return true;
"""

BRIDGE_SCRIPT = """
const done = arguments[arguments.length - 1];
const [sourcePath, occurrence, digest] = arguments;
import('/wails/runtime.js').then(async (runtime) => {
  done(await runtime.Call.ByName('main.StudioBridge.NavigateThreeView', 'workspace-1', sourcePath, occurrence, digest));
}).catch((error) => done({error: String(error)}));
"""


def main() -> int:
    sys.stdout.reconfigure(encoding="utf-8", errors="strict")
    sys.stderr.reconfigure(encoding="utf-8", errors="strict")
    if len(sys.argv) != 3:
        print("usage: desktop_three_view_smoke.py <desktop-exe> <evidence-dir>", file=sys.stderr)
        return 2
    app_path = Path(sys.argv[1]).resolve()
    evidence_dir = Path(sys.argv[2]).resolve()
    evidence_dir.mkdir(parents=True, exist_ok=True)
    facts: dict[str, Any] = {
        "result": "FAIL_NOT_COMPLETED",
        "scope": "native Wails WebView2 three-view page on the synthetic Workspace: Business -> Function -> CRUD -> Source, and a pin kept across reselection",
        "not_covered": ["real X-TEA design sources", "editing", "Owner acceptance of the screen"],
        "app_path": str(app_path),
        "candidate_head_sha": os.environ.get("CANDIDATE_HEAD_SHA"),
        "events": [],
    }
    facts_path = evidence_dir / "summary.json"
    logs = {name: (evidence_dir / f"{name}.log").open("w", encoding="utf-8", errors="replace")
            for name in ("msedgedriver.stdout", "msedgedriver.stderr", "wails-app.stdout", "wails-app.stderr")}
    driver_process = None
    app_process = None
    app_handle = None
    session = None
    base_url = ""
    scratch = None
    failure = None

    def record(event: str, **values: Any) -> None:
        facts["events"].append({"event": event, **values})
        smoke.write_json(facts_path, facts)
        print(f"{event}: {json.dumps(values, ensure_ascii=False, sort_keys=True)}", flush=True)

    def execute(script: str, *args: Any) -> Any:
        return smoke.webdriver_request(base_url, "POST", f"/session/{session}/execute/sync", {"script": script, "args": list(args)}).get("value")

    def page() -> dict[str, Any]:
        return execute(PAGE_STATE_SCRIPT) or {}

    def wait_for(description: str, condition) -> dict[str, Any]:
        deadline = time.monotonic() + 20
        state: dict[str, Any] = {}
        while time.monotonic() < deadline:
            state = page()
            if condition(state):
                return state
            time.sleep(0.2)
        raise RuntimeError(f"The three-view page did not reach: {description}. Last state: {state!r}")

    def bridge(occurrence: str = "", digest: str = "") -> dict[str, Any]:
        response = smoke.webdriver_request(
            base_url, "POST", f"/session/{session}/execute/async",
            {"script": BRIDGE_SCRIPT, "args": [SOURCE_PATH, occurrence, digest]}, timeout_seconds=30,
        ).get("value", {})
        if response.get("error") or response.get("ok") is not True:
            raise RuntimeError(f"StudioBridge.NavigateThreeView failed in the native WebView: {response!r}")
        return response["value"]

    try:
        if not app_path.is_file():
            raise RuntimeError(f"Desktop executable does not exist: {app_path}")
        facts["exe_sha256"] = hashlib.sha256(app_path.read_bytes()).hexdigest()
        if smoke.find_processes(smoke.APP_IMAGE):
            raise RuntimeError("A process with the smoke executable name existed before launch")

        fixture = Path(__file__).resolve().parent.parent / "plugins" / "xteathreeview" / "testdata" / "workspace"
        if not (fixture / SOURCE_PATH).is_file():
            raise RuntimeError(f"The synthetic three-view Workspace is missing: {fixture}")
        scratch = tempfile.TemporaryDirectory(prefix="tei-studio-three-view-smoke-")
        workspace_root = Path(scratch.name) / "workspace"
        shutil.copytree(fixture, workspace_root)
        workspace_before = smoke.workspace_snapshot(workspace_root)
        facts["workspace_before"] = workspace_before

        runtime_versions = smoke.query_runtime_versions()
        if not runtime_versions:
            raise RuntimeError("WebView2 Evergreen Runtime version was not found")
        runtime_version = runtime_versions[0]["version"]
        facts["webview2_runtime_version"] = runtime_version
        driver_path = evidence_dir.parent.parent / "tei-studio-webdriver" / "msedgedriver.exe"
        if not driver_path.is_file():
            driver_path = smoke.download_compatible_driver(runtime_version, evidence_dir.parent, facts)
        version_output = subprocess.run([str(driver_path), "--version"], check=True, capture_output=True, text=True, timeout=15)
        match = smoke.VERSION_RE.search(version_output.stdout + version_output.stderr)
        if not match or smoke.version_prefix(match.group(1)) != smoke.version_prefix(runtime_version):
            raise RuntimeError(f"WebView2/Edge WebDriver version mismatch: Runtime={runtime_version}, Driver={version_output.stdout!r}")
        facts["webdriver_version"] = match.group(1)

        debug_port = smoke.free_loopback_port()
        app_env = os.environ.copy()
        app_env.pop("WAILS_SERVER_HOST", None)
        app_env.pop("FRONTEND_DEVSERVER_URL", None)
        app_process = subprocess.Popen(
            [str(app_path), f"--workspace={workspace_root}", f"--webview-debug-port={debug_port}"],
            stdout=logs["wails-app.stdout"], stderr=logs["wails-app.stderr"], env=app_env, text=True,
        )
        app_handle = smoke.open_process_handle(app_process.pid)
        record("wails-process-launched", pid=app_process.pid)

        debug_url = f"http://localhost:{debug_port}"
        deadline = time.monotonic() + 60
        last_error = None
        while time.monotonic() < deadline:
            if app_process.poll() is not None:
                raise RuntimeError(f"Wails Desktop exited during startup with code {app_process.returncode}")
            try:
                if smoke.webdriver_request(debug_url, "GET", "/json/version").get("webSocketDebuggerUrl"):
                    break
            except Exception as exc:
                last_error = f"{type(exc).__name__}: {exc}"
            time.sleep(0.25)
        else:
            raise RuntimeError(f"WebView2 remote debugging endpoint did not become ready: {last_error}")
        exposed = sorted({str(item.get("LocalAddress")) for item in smoke.query_listening_sockets(debug_port)}
                         - {"127.0.0.1", "::1", "::ffff:127.0.0.1"})
        if exposed:
            raise RuntimeError(f"WebView2 debugger endpoint is not loopback-only: {exposed}")
        record("webview2-debugger-ready", port=debug_port)

        driver_port = smoke.free_loopback_port()
        base_url = f"http://127.0.0.1:{driver_port}"
        driver_env = os.environ.copy()
        driver_env["MSEDGEDRIVER_TELEMETRY_OPTOUT"] = "1"
        driver_process = subprocess.Popen(
            [str(driver_path), f"--port={driver_port}"],
            stdout=logs["msedgedriver.stdout"], stderr=logs["msedgedriver.stderr"], env=driver_env, text=True,
        )
        deadline = time.monotonic() + 20
        while time.monotonic() < deadline:
            if driver_process.poll() is not None:
                raise RuntimeError(f"msedgedriver exited before readiness with code {driver_process.returncode}")
            try:
                if smoke.webdriver_request(base_url, "GET", "/status").get("value", {}).get("ready") is True:
                    break
            except Exception:
                pass
            time.sleep(0.25)
        else:
            raise RuntimeError("msedgedriver did not become ready on its loopback port")
        created = smoke.webdriver_request(
            base_url, "POST", "/session",
            {"capabilities": {"alwaysMatch": {"browserName": "webview2", "ms:edgeOptions": {"debuggerAddress": f"localhost:{debug_port}"}}}},
            timeout_seconds=30,
        )
        session = created.get("sessionId") or created.get("value", {}).get("sessionId")
        if not session:
            raise RuntimeError(f"WebDriver did not return a session ID: {created!r}")
        record("wails-webview-session-attached", session_id=session)

        # The Desktop window opens the Workbench. The three-view page is another
        # embedded asset of the same window and origin.
        opened = smoke.webdriver_request(base_url, "GET", f"/session/{session}/url").get("value", "")
        target = urllib.parse.urljoin(opened, "/three-view/")
        facts["desktop_initial_url"] = opened
        facts["three_view_url"] = target
        smoke.webdriver_request(base_url, "POST", f"/session/{session}/url", {"url": target}, timeout_seconds=30)
        ready = wait_for("the Workspace list", lambda s: s.get("pathname") == "/three-view/" and s.get("navigateDisabled") is False)
        if ready.get("workspaceOptions") != ["workspace-1"]:
            raise RuntimeError(f"The page did not list the one allowlisted Workspace: {ready.get('workspaceOptions')!r}")

        execute(CLICK_SCRIPT, ".controls", "読み取る")
        shown = wait_for("a completed read", lambda s: s.get("state", "").startswith("読取り完了"))
        value = bridge()
        if value.get("schemaVersion") != "tei.operations.three-view-navigation/v0" or value.get("effect") != "read-only":
            raise RuntimeError(f"Unexpected three-view answer: {value.get('schemaVersion')!r} {value.get('effect')!r}")
        subjects = {item["occurrence"]: item for item in value["subjects"]}
        by_kind = {item["sourceFact"]["kind"]: item for item in value["subjects"]}
        if sorted(by_kind) != ["Function", "Table", "Task", "TaskAction"] or len(value["crud"]) != 1:
            raise RuntimeError(f"The synthetic Workspace was not read as one Data, one Function and one Business subject with an action: {sorted(by_kind)!r}")
        for view in ("data", "function", "business"):
            want = sum(1 for item in value["subjects"] if item["view"] == view)
            if len(shown.get(view, [])) != want:
                raise RuntimeError(f"The {view} view shows {len(shown.get(view, []))} subjects, the Host returned {want}")
        if "全要素を配置しました" not in shown.get("coverage", ""):
            raise RuntimeError(f"Coverage of the fully placed fixture was not shown as complete: {shown.get('coverage')!r}")
        record("three-views-rendered", data=shown["data"], function=shown["function"], business=shown["business"])

        action, function, table = by_kind["TaskAction"], by_kind["Function"], by_kind["Table"]
        row = value["crud"][0]
        if row["function"] != function["occurrence"] or row["data"] != table["occurrence"]:
            raise RuntimeError("The CRUD row does not join the Function and Data occurrences of the three views")

        # Business: the action has no source ID, is shown as such, and leads to the Function.
        execute(CLICK_SCRIPT, "#view-business", action["sourceFact"]["name"])
        at_action = wait_for("the business action", lambda s: s.get("focusHeading", "").startswith(action["sourceFact"]["name"]))
        for expected in ("Source IDなし", action["sourceFact"]["locator"], "action-uses-function", "結び付けなし",
                         "業務Actionに固有のCRUDは表示しません"):
            if expected not in at_action["focus"] and expected not in at_action["focusHeading"]:
                raise RuntimeError(f"The business action panel lacks {expected!r}: {at_action['focus']!r}")

        # Function: reached through the relation, with its declared CRUD row.
        execute(CLICK_SCRIPT, "#focus", function["sourceFact"]["name"] + " [")
        at_function = wait_for("the function", lambda s: s.get("focusHeading", "").startswith(function["sourceFact"]["name"]))
        for expected in (function["sourceFact"]["locator"], "function-data-operation", row["dataName"], "原典に明記"):
            if expected not in at_function["focus"]:
                raise RuntimeError(f"The function panel lacks {expected!r}: {at_function['focus']!r}")

        # Pin the Function, then look at the Data subject: the pin stays.
        execute(CLICK_SCRIPT, ".panel", "選択中の対象を固定")
        pinned = wait_for("a held pin", lambda s: "保持中" in s.get("pin", ""))
        execute(CLICK_SCRIPT, "#view-data", table["sourceFact"]["name"])
        at_data = wait_for("the data subject", lambda s: s.get("focusHeading", "").startswith(table["sourceFact"]["name"]))
        if "保持中" not in at_data.get("pin", "") or function["sourceFact"]["name"] not in at_data.get("pin", ""):
            raise RuntimeError(f"Selecting another subject changed the pin: {at_data.get('pin')!r}")
        field_locators = [item["sourceFact"]["locator"] for item in value.get("fields", [])]
        for expected in [table["sourceFact"]["locator"], "key-member", *field_locators]:
            if expected not in at_data["focus"]:
                raise RuntimeError(f"The data panel lacks {expected!r}: {at_data['focus']!r}")
        with_pin = bridge(function["occurrence"], value["source"]["artifactDigest"])
        if with_pin.get("pin") != {"occurrence": function["occurrence"], "sourceArtifactDigest": value["source"]["artifactDigest"], "status": "held"}:
            raise RuntimeError(f"The Host did not return the pin unchanged and held: {with_pin.get('pin')!r}")
        if function["occurrence"] not in subjects:
            raise RuntimeError("The pinned occurrence is not a subject of the answer")

        screenshot = smoke.webdriver_request(base_url, "GET", f"/session/{session}/screenshot").get("value")
        if screenshot:
            (evidence_dir / "webview2-three-view-screen.png").write_bytes(base64.b64decode(screenshot))
        workspace_after = smoke.workspace_snapshot(workspace_root)
        facts["workspace_after"] = workspace_after
        if workspace_after != workspace_before:
            raise RuntimeError("The read-only three-view flow changed the Workspace")
        smoke.write_json(evidence_dir / "page-state.json", {
            "business": at_action, "function": at_function, "pinned": pinned, "data": at_data, "bridgeValue": value, "bridgeWithPin": with_pin,
        })
        record("three-view-business-function-crud-source-verified",
               source_path=SOURCE_PATH, subjects=len(value["subjects"]), crud_rows=len(value["crud"]),
               pin_held_after_reselection=True, workspace_unchanged=True)
    except Exception as exc:
        failure = f"{type(exc).__name__}: {exc}"
        (evidence_dir / "failure.txt").write_text(traceback.format_exc(), encoding="utf-8")
        record("smoke-failed", reason=failure)
    finally:
        if session:
            try:
                smoke.webdriver_request(base_url, "DELETE", f"/session/{session}")
            except Exception as exc:
                failure = failure or f"{type(exc).__name__}: {exc}"
        if app_process is not None and app_process.poll() is None:
            try:
                smoke.request_window_close(app_process.pid)
            except Exception as exc:
                facts["app_window_close_error"] = f"{type(exc).__name__}: {exc}"
        if app_handle is not None:
            try:
                exit_code = smoke.wait_process(app_handle, 20)
                facts["app_exit_code"] = exit_code
                if exit_code is None:
                    failure = failure or "Wails Desktop process did not exit after its window was closed"
                    subprocess.run(["taskkill.exe", "/F", "/T", "/PID", str(app_process.pid)], capture_output=True, text=True, timeout=15)
                elif exit_code != 0:
                    failure = failure or f"Wails Desktop exited with code {exit_code}"
            except Exception as exc:
                failure = failure or f"{type(exc).__name__}: {exc}"
            finally:
                smoke.close_process_handle(app_handle)
        if driver_process is not None and driver_process.poll() is None:
            driver_process.terminate()
            try:
                driver_process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                driver_process.kill()
        if scratch is not None:
            try:
                scratch.cleanup()
            except Exception as exc:
                facts["scratch_workspace_cleanup_error"] = f"{type(exc).__name__}: {exc}"
        for log in logs.values():
            log.close()
        facts["result"] = "FAIL" if failure else "PASS_NATIVE_WEBVIEW2_THREE_VIEW_READ_ONLY"
        if failure:
            facts["failure"] = failure
        smoke.write_json(facts_path, facts)
        print(f"SMOKE_RESULT={facts['result']}", flush=True)
    return 1 if failure else 0


if __name__ == "__main__":
    raise SystemExit(main())
