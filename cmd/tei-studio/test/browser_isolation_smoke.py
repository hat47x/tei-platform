"""Verify two browser contexts keep their Workspace selections separate."""

from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import subprocess
import tempfile
import time
from pathlib import Path
from typing import Any, Awaitable, Callable
from urllib.parse import urlparse

from playwright.async_api import Browser, BrowserContext, Page, Response, Route, async_playwright


LIST_WORKSPACES = "main.StudioBridge.ListWorkspaces"
CATALOG = "main.StudioBridge.CatalogWorkspace"
INSPECT = "main.StudioBridge.InspectSubject"
VALIDATE = "main.StudioBridge.ValidateWorkspace"
OBSERVE_XEAD = "main.StudioBridge.ObserveXead"
WORKBENCH_PATH = "workbench/"
ALPHA_SUBJECT = "isolation_alpha.record_alpha"
BETA_SUBJECT = "isolation_beta.record_beta"
ALPHA_VALIDATION = "isolation_alpha.validation_marker_alpha"
BETA_VALIDATION = "isolation_beta.validation_marker_beta"
ALPHA_XEAD_PATH = "studio-isolation-alpha/observe-alpha.xead"
LATE_ALPHA_PATH = "studio-isolation-alpha/late-alpha.xead"
BETA_XEAD_PATH = "studio-isolation-beta/observe-beta.xead"
ALPHA_XEAD_NAME = "Workspace Alpha Table"
BETA_XEAD_NAME = "Workspace Beta Table"


def runtime_call(request: Any) -> dict[str, Any] | None:
    if urlparse(request.url).path != "/wails/runtime" or request.method != "POST":
        return None
    try:
        body = request.post_data_json
    except Exception:
        return None
    if not isinstance(body, dict):
        return None
    call = body.get("args")
    if not isinstance(call, dict) or not isinstance(call.get("methodName"), str):
        return None
    args = call.get("args")
    if not isinstance(args, list):
        return None
    return {
        "method": call["methodName"],
        "args": args,
        "call_id": call.get("call-id"),
    }


class RPCRecorder:
    """Collect actual browser requests and responses for exact argument checks."""

    def __init__(self, page: Page) -> None:
        self.calls: list[dict[str, Any]] = []
        self.responses: list[dict[str, Any]] = []
        self.waiters: list[
            tuple[str, list[Any], str | None, int, asyncio.Future[dict[str, Any]]]
        ] = []
        self.tasks: set[asyncio.Task[None]] = set()
        self.request_index_by_call_id: dict[str, int] = {}
        self.duplicate_call_ids: set[str] = set()
        page.on("request", self._on_request)
        page.on("response", self._on_response)

    def _on_request(self, request: Any) -> None:
        call = runtime_call(request)
        if call is not None:
            request_index = len(self.calls)
            call["request_index"] = request_index
            self.calls.append(call)
            call_id = call["call_id"]
            if isinstance(call_id, str):
                if call_id in self.request_index_by_call_id:
                    self.duplicate_call_ids.add(call_id)
                else:
                    self.request_index_by_call_id[call_id] = request_index

    def _on_response(self, response: Response) -> None:
        task = asyncio.create_task(self._capture(response))
        self.tasks.add(task)
        task.add_done_callback(self.tasks.discard)

    async def _capture(self, response: Response) -> None:
        call = runtime_call(response.request)
        if call is None:
            return
        try:
            envelope = await response.json()
        except Exception as error:
            envelope = {"capture_error": f"{type(error).__name__}: {error}"}
        request_index = self.request_index_by_call_id.get(call["call_id"], -1)
        call["request_index"] = request_index
        record = {
            "call": call,
            "request_index": request_index,
            "status": response.status,
            "envelope": envelope,
        }
        self.responses.append(record)
        for waiter in tuple(self.waiters):
            method, args, call_id, after_request, future = waiter
            if (
                request_index >= after_request
                and call["method"] == method
                and call["args"] == args
                and (call_id is None or call["call_id"] == call_id)
                and not future.done()
            ):
                future.set_result(record)
                self.waiters.remove(waiter)

    @property
    def request_count(self) -> int:
        return len(self.calls)

    def matching_requests(
        self, method: str, args: list[Any], after_request: int = 0
    ) -> list[dict[str, Any]]:
        return [
            call
            for call in self.calls
            if call["method"] == method
            and call["args"] == args
            and call["request_index"] >= after_request
        ]

    async def wait_for(
        self,
        method: str,
        args: list[Any],
        *,
        call_id: str | None = None,
        after_request: int = 0,
    ) -> dict[str, Any]:
        for record in self.responses:
            call = record["call"]
            if (
                record["request_index"] >= after_request
                and call["method"] == method
                and call["args"] == args
                and (call_id is None or call["call_id"] == call_id)
            ):
                return record
        future: asyncio.Future[dict[str, Any]] = asyncio.get_running_loop().create_future()
        waiter = (method, args, call_id, after_request, future)
        self.waiters.append(waiter)
        try:
            return await asyncio.wait_for(future, timeout=15)
        finally:
            if waiter in self.waiters:
                self.waiters.remove(waiter)

    async def drain(self) -> None:
        if self.tasks:
            await asyncio.gather(*tuple(self.tasks))
        assert not self.duplicate_call_ids, self.duplicate_call_ids


def assert_success(record: dict[str, Any], method: str, args: list[Any]) -> dict[str, Any]:
    call = record["call"]
    assert call["method"] == method and call["args"] == args, call
    assert isinstance(call["call_id"], str) and call["call_id"], call
    assert record["status"] == 200, record
    envelope = record["envelope"]
    assert isinstance(envelope, dict) and envelope.get("ok") is True, envelope
    assert "value" in envelope, envelope
    return envelope


def marker_text(value: Any) -> str:
    return json.dumps(value, ensure_ascii=False, sort_keys=True)


def create_workspace_fixture(scratch: Path, variant: str) -> Path:
    namespace = f"isolation_{variant}"
    record_name = f"Record{variant.title()}"
    subject_id = f"{namespace}.record_{variant}"
    validation_id = f"{namespace}.validation_marker_{variant}"
    root = scratch / f"workspace-{variant}"
    semantic = root / "semantic"
    semantic.mkdir(parents=True)
    (root / "tei.workspace.yaml").write_text(
        f"workspace: browser_isolation_{variant}\nsources:\n  - semantic\n",
        encoding="utf-8",
    )
    (semantic / f"{variant}-record.tei").write_text(
        f"namespace {namespace}\n\ndata {record_name} {{\n"
        "    attribute probe type std.identifier\n"
        "}\n\n"
        f"validation {validation_id} positive {{\n"
        f"    attribute {subject_id}.missing_attribute\n"
        "}\n",
        encoding="utf-8",
    )

    xead_name = f"Workspace {variant.title()} Table"
    xead_path = root / f"studio-isolation-{variant}" / f"observe-{variant}.xead"
    xead_path.parent.mkdir(parents=True, exist_ok=True)
    xead_content = (
        "<System>\n"
        f'  <Table ID="{variant.upper()}_TABLE_1" Name="{xead_name}">\n'
        f'    <TableField ID="{variant.upper()}_FIELD_1" '
        f'Name="Workspace{variant.title()}Field" '
        f'DataTypeID="{variant.upper()}_TYPE_1" />\n'
        "  </Table>\n"
        "</System>\n"
    )
    xead_path.write_text(xead_content, encoding="utf-8")
    if variant == "alpha":
        late_path = root / LATE_ALPHA_PATH
        late_path.parent.mkdir(parents=True, exist_ok=True)
        late_path.write_text(xead_content, encoding="utf-8")
    return root


def prepare_workspaces(scratch: Path) -> tuple[Path, Path]:
    alpha = create_workspace_fixture(scratch, "alpha")
    beta = create_workspace_fixture(scratch, "beta")
    if alpha.resolve() == beta.resolve() or alpha.resolve() in beta.resolve().parents:
        raise RuntimeError("Workspace A and B do not have independent roots")
    for root in (alpha, beta):
        manifest = (root / "tei.workspace.yaml").read_text(encoding="utf-8")
        assert "sources:\n  - semantic\n" in manifest
        semantic_sources = list((root / "semantic").glob("*.tei"))
        assert len(semantic_sources) == 1, semantic_sources
        assert not any(
            "import " in source.read_text(encoding="utf-8")
            for source in semantic_sources
        )
    assert ALPHA_SUBJECT not in (beta / "semantic" / "beta-record.tei").read_text(encoding="utf-8")
    assert BETA_SUBJECT not in (alpha / "semantic" / "alpha-record.tei").read_text(encoding="utf-8")
    return alpha, beta


def fixture_snapshot(roots: tuple[Path, Path]) -> dict[str, str]:
    snapshot = {}
    for root in roots:
        for path in sorted(root.rglob("*")):
            if path.is_file():
                key = f"{root.name}/{path.relative_to(root).as_posix()}"
                snapshot[key] = hashlib.sha256(path.read_bytes()).hexdigest()
    return snapshot


async def wait_for_host(
    playwright: Any,
    process: subprocess.Popen[bytes],
    base_url: str,
    log_path: Path,
) -> None:
    request = await playwright.request.new_context()
    try:
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            if process.poll() is not None:
                raise RuntimeError(
                    f"Wails Server exited with code {process.returncode}:\n"
                    f"{log_path.read_text(encoding='utf-8', errors='replace')}"
                )
            try:
                response = await request.get(base_url + "health", timeout=1000)
                if response.status == 200:
                    payload = await response.json()
                    if payload.get("status") == "ok":
                        return
            except Exception:
                pass
            # This only gives the just-started process time to bind its port;
            # response-order testing below uses an explicit asyncio.Event.
            await asyncio.sleep(0.1)
        raise RuntimeError(
            f"Wails Server did not become healthy:\n"
            f"{log_path.read_text(encoding='utf-8', errors='replace')}"
        )
    finally:
        await request.dispose()


async def wait_workspace_ready(page: Page, workspace_id: str, marker: str) -> None:
    await page.wait_for_function(
        """({workspaceId, marker}) =>
          document.querySelector('#connection')?.textContent === '実Hostに接続'
          && document.querySelector('#workspace')?.value === workspaceId
          && document.querySelector('#subjects')?.innerText.includes(marker)
          && document.querySelector('#commands')?.innerText.includes(marker)""",
        arg={"workspaceId": workspace_id, "marker": marker},
        timeout=15_000,
    )


def assert_workspace_list(envelope: dict[str, Any]) -> None:
    value = envelope.get("value")
    assert isinstance(value, list), value
    assert [item.get("id") for item in value] == ["workspace-1", "workspace-2"], value
    assert [item.get("label") for item in value] == ["workspace-alpha", "workspace-beta"], value


def assert_catalog_marker(envelope: dict[str, Any], subject_id: str) -> None:
    entries = envelope.get("value", {}).get("entries")
    assert isinstance(entries, list), envelope
    assert any(
        isinstance(entry, dict)
        and entry.get("subject", {}).get("id") == subject_id
        for entry in entries
    ), subject_id


async def perform_and_wait(
    recorder: RPCRecorder,
    method: str,
    args: list[Any],
    action: Callable[[], Awaitable[Any]],
    *,
    after_request: int | None = None,
) -> dict[str, Any]:
    if after_request is None:
        after_request = recorder.request_count
    await action()
    record = await recorder.wait_for(method, args, after_request=after_request)
    new_calls = recorder.matching_requests(method, args, after_request)
    assert len(new_calls) == 1, new_calls
    assert record["call"]["call_id"] == new_calls[0]["call_id"], record
    return record


async def assert_initial_workspace(
    page: Page,
    recorder: RPCRecorder,
    workspace_id: str,
    subject_id: str,
    validation_marker: str,
) -> None:
    await wait_workspace_ready(page, workspace_id, validation_marker)
    list_record = await recorder.wait_for(LIST_WORKSPACES, [])
    assert_workspace_list(assert_success(list_record, LIST_WORKSPACES, []))

    catalog_record = await recorder.wait_for(CATALOG, [workspace_id, ""])
    catalog = assert_success(catalog_record, CATALOG, [workspace_id, ""])
    assert_catalog_marker(catalog, subject_id)

    validate_args = [workspace_id]
    validation_record = await recorder.wait_for(VALIDATE, validate_args)
    validation = assert_success(validation_record, VALIDATE, validate_args)
    assert validation_marker in marker_text(validation), validation
    assert validation_marker in await page.locator("#commands").inner_text()


async def inspect_marker(
    page: Page,
    recorder: RPCRecorder,
    workspace_id: str,
    subject_id: str,
) -> None:
    await page.locator("#search").fill(subject_id)
    target = page.locator("#subjects button").filter(has_text=subject_id)
    assert await target.count() == 1, subject_id
    args = [workspace_id, "semantic-data", subject_id]
    record = await perform_and_wait(recorder, INSPECT, args, target.click)
    envelope = assert_success(record, INSPECT, args)
    value = envelope.get("value")
    assert isinstance(value, dict) and value.get("subject", {}).get("id") == subject_id, value
    await page.wait_for_function(
        """(id) => document.querySelector('#subject-title')?.textContent === id
          && document.querySelector('#source-status')?.textContent?.startsWith('参照情報：')""",
        arg=subject_id,
        timeout=10_000,
    )
    displayed = await page.locator("#editor .cm-content").inner_text()
    assert json.loads(displayed) == value.get("canonical"), displayed


async def validate_marker(
    page: Page,
    recorder: RPCRecorder,
    workspace_id: str,
    marker: str,
) -> None:
    button = page.locator('#commands button[data-command-id="validate"]')
    args = [workspace_id]
    record = await perform_and_wait(recorder, VALIDATE, args, button.click)
    envelope = assert_success(record, VALIDATE, args)
    assert marker in marker_text(envelope), envelope
    await page.wait_for_function(
        """(marker) => document.querySelector('#commands')?.innerText.includes(marker)""",
        arg=marker,
        timeout=10_000,
    )


async def observe_marker(
    page: Page,
    recorder: RPCRecorder,
    workspace_id: str,
    path: str,
    marker: str,
) -> dict[str, Any]:
    await page.locator("#xead-path").fill(path)
    args = [workspace_id, path]
    record = await perform_and_wait(
        recorder,
        OBSERVE_XEAD,
        args,
        page.locator("#observe-xead").click,
    )
    envelope = assert_success(record, OBSERVE_XEAD, args)
    assert marker in marker_text(envelope), envelope
    await page.wait_for_function(
        """(marker) => document.querySelector('#xead-result')?.innerText.includes(marker)
          && document.querySelector('#xead-status')?.textContent?.startsWith('観察完了：')""",
        arg=marker,
        timeout=10_000,
    )
    return envelope


async def exercise_two_contexts(browser: Browser, base_url: str) -> None:
    context_a: BrowserContext | None = None
    context_b: BrowserContext | None = None
    release_old_response: asyncio.Event | None = None
    old_route_done: asyncio.Future[None] | None = None
    page_a: Page | None = None
    recorder_a: RPCRecorder | None = None
    old_route_handler: Any = None
    route_installed = False
    try:
        context_a = await browser.new_context(service_workers="block")
        context_b = await browser.new_context(service_workers="block")
        page_a = await context_a.new_page()
        page_b = await context_b.new_page()
        recorder_a = RPCRecorder(page_a)
        recorder_b = RPCRecorder(page_b)

        await asyncio.gather(
            page_a.goto(base_url + WORKBENCH_PATH, wait_until="load", timeout=15_000),
            page_b.goto(base_url + WORKBENCH_PATH, wait_until="load", timeout=15_000),
        )
        await asyncio.gather(
            assert_initial_workspace(
                page_a, recorder_a, "workspace-1", ALPHA_SUBJECT, ALPHA_VALIDATION
            ),
            assert_initial_workspace(
                page_b, recorder_b, "workspace-1", ALPHA_SUBJECT, ALPHA_VALIDATION
            ),
        )

        workspace_change_boundary = recorder_b.request_count
        beta_catalog_args = ["workspace-2", ""]
        beta_catalog = await perform_and_wait(
            recorder_b,
            CATALOG,
            beta_catalog_args,
            lambda: page_b.locator("#workspace").select_option("workspace-2"),
            after_request=workspace_change_boundary,
        )
        await wait_workspace_ready(page_b, "workspace-2", BETA_VALIDATION)
        assert_catalog_marker(
            assert_success(beta_catalog, CATALOG, beta_catalog_args), BETA_SUBJECT
        )
        beta_validation_args = ["workspace-2"]
        beta_validation = await recorder_b.wait_for(
            VALIDATE,
            beta_validation_args,
            after_request=workspace_change_boundary,
        )
        beta_validation_calls = recorder_b.matching_requests(
            VALIDATE, beta_validation_args, workspace_change_boundary
        )
        assert len(beta_validation_calls) == 1, beta_validation_calls
        assert (
            beta_validation["call"]["call_id"]
            == beta_validation_calls[0]["call_id"]
        ), beta_validation
        assert BETA_VALIDATION in marker_text(
            assert_success(beta_validation, VALIDATE, beta_validation_args)
        )

        # Browser A remains on A while Browser B switches to B; neither DOM
        # contains the other Workspace's unique Catalog/Validate marker.
        await wait_workspace_ready(page_a, "workspace-1", ALPHA_VALIDATION)
        assert ALPHA_SUBJECT in await page_a.locator("#subjects").inner_text()
        assert BETA_SUBJECT not in await page_a.locator("#subjects").inner_text()
        assert BETA_VALIDATION not in await page_a.locator("#commands").inner_text()
        assert BETA_SUBJECT in await page_b.locator("#subjects").inner_text()
        assert ALPHA_SUBJECT not in await page_b.locator("#subjects").inner_text()
        assert ALPHA_VALIDATION not in await page_b.locator("#commands").inner_text()

        await asyncio.gather(
            inspect_marker(page_a, recorder_a, "workspace-1", ALPHA_SUBJECT),
            inspect_marker(page_b, recorder_b, "workspace-2", BETA_SUBJECT),
        )
        await asyncio.gather(
            validate_marker(page_a, recorder_a, "workspace-1", ALPHA_VALIDATION),
            validate_marker(page_b, recorder_b, "workspace-2", BETA_VALIDATION),
        )
        await asyncio.gather(
            observe_marker(
                page_a,
                recorder_a,
                "workspace-1",
                ALPHA_XEAD_PATH,
                ALPHA_XEAD_NAME,
            ),
            observe_marker(
                page_b,
                recorder_b,
                "workspace-2",
                BETA_XEAD_PATH,
                BETA_XEAD_NAME,
            ),
        )

        loop = asyncio.get_running_loop()
        fetched_old_response: asyncio.Future[dict[str, Any]] = loop.create_future()
        old_route_done = loop.create_future()
        release_old_response = asyncio.Event()
        intercepted_call_ids: set[str] = set()
        route_count = 0
        target_call_id: str | None = None
        async def delay_one_real_alpha_response(route: Route) -> None:
            nonlocal route_count, target_call_id
            request_call = runtime_call(route.request)
            if request_call is None:
                await route.fallback()
                return
            is_target = (
                request_call["method"] == OBSERVE_XEAD
                and request_call["args"] == ["workspace-1", LATE_ALPHA_PATH]
            )
            if not is_target:
                await route.fallback()
                return
            delay_this_request = False
            try:
                call_id = request_call["call_id"]
                if not isinstance(call_id, str) or not call_id:
                    raise AssertionError(f"Target RPC lacks a call-id: {request_call!r}")
                if target_call_id is None:
                    target_call_id = call_id
                if call_id != target_call_id:
                    await route.fallback()
                    return
                delay_this_request = True
                route_count += 1
                if route_count != 1 or call_id in intercepted_call_ids:
                    raise AssertionError(
                        f"More than one target response was intercepted: {request_call!r}"
                    )
                intercepted_call_ids.add(call_id)
                host_response = await route.fetch()
                envelope = await host_response.json()
                if (
                    host_response.status != 200
                    or not isinstance(envelope, dict)
                    or envelope.get("ok") is not True
                ):
                    raise AssertionError(
                        f"Held response was not a successful real Host response: "
                        f"status={host_response.status}, body={envelope!r}"
                    )
                fetched_old_response.set_result(
                    {"call": request_call, "response": host_response, "envelope": envelope}
                )
                await release_old_response.wait()
                await route.fulfill(response=host_response)
            except BaseException as error:
                if not fetched_old_response.done():
                    fetched_old_response.set_exception(error)
                raise
            finally:
                if delay_this_request and old_route_done is not None and not old_route_done.done():
                    old_route_done.set_result(None)

        old_route_handler = delay_one_real_alpha_response
        await context_a.route("**/wails/runtime", old_route_handler)
        route_installed = True
        old_request_boundary = recorder_a.request_count
        await page_a.locator("#xead-path").fill(LATE_ALPHA_PATH)
        await page_a.locator("#observe-xead").click()
        held = await asyncio.wait_for(fetched_old_response, timeout=15)
        old_call = held["call"]
        old_call_id = old_call["call_id"]
        old_envelope = held["envelope"]
        assert old_call["method"] == OBSERVE_XEAD
        assert old_call["args"] == ["workspace-1", LATE_ALPHA_PATH]
        assert old_call_id in intercepted_call_ids and route_count == 1
        delayed_calls = recorder_a.matching_requests(
            OBSERVE_XEAD,
            ["workspace-1", LATE_ALPHA_PATH],
            old_request_boundary,
        )
        assert (
            len(delayed_calls) == 1
            and delayed_calls[0]["call_id"] == old_call_id
        ), delayed_calls
        assert ALPHA_XEAD_NAME in marker_text(old_envelope), old_envelope

        # Switch this same client to B, display B's real result, then deliver
        # the already-fetched A response and verify it cannot replace B.
        await page_a.locator("#workspace").select_option("workspace-2")
        await wait_workspace_ready(page_a, "workspace-2", BETA_VALIDATION)
        await observe_marker(
            page_a,
            recorder_a,
            "workspace-2",
            BETA_XEAD_PATH,
            BETA_XEAD_NAME,
        )
        assert BETA_XEAD_NAME in await page_b.locator("#xead-result").inner_text()
        assert ALPHA_XEAD_NAME not in await page_b.locator("#xead-result").inner_text()

        release_old_response.set()
        old_delivery = await recorder_a.wait_for(
            OBSERVE_XEAD,
            ["workspace-1", LATE_ALPHA_PATH],
            call_id=old_call_id,
            after_request=old_request_boundary,
        )
        delivered_envelope = assert_success(
            old_delivery, OBSERVE_XEAD, ["workspace-1", LATE_ALPHA_PATH]
        )
        assert delivered_envelope == old_envelope, (
            "The held response changed before browser delivery"
        )
        assert old_route_done is not None
        await asyncio.wait_for(old_route_done, timeout=10)
        # Let the browser process the fulfilled response and its stale-result
        # guard through two animation frames; no time-based response ordering.
        await page_a.evaluate(
            "() => new Promise((resolve) => "
            "requestAnimationFrame(() => requestAnimationFrame(resolve)))"
        )
        current_a_xead = await page_a.locator("#xead-result").inner_text()
        assert BETA_XEAD_NAME in current_a_xead, current_a_xead
        assert ALPHA_XEAD_NAME not in current_a_xead, current_a_xead
        assert await page_a.locator("#workspace").input_value() == "workspace-2"
        assert await page_b.locator("#workspace").input_value() == "workspace-2"
        await recorder_a.drain()
        await recorder_b.drain()

        print(
            "PASS_BROWSER_CONTEXT_ISOLATION "
            "same loopback Host; two independent BrowserContexts; "
            "Catalog/Inspect/Validate/ObserveXead markers matched request args and "
            "displayed responses; one real alpha response fetched once, held, then "
            "delivered after beta; beta display retained"
        )
        print(
            "SCOPE_LIMITATION same OS user and loopback Host only; "
            "Workspace-ID authorization, shared-web multi-user isolation, and "
            "Principal-specific authorization not tested"
        )
    finally:
        if release_old_response is not None:
            release_old_response.set()
        if route_installed and context_a is not None and old_route_handler is not None:
            if old_route_done is not None and not old_route_done.done():
                try:
                    await asyncio.wait_for(old_route_done, timeout=5)
                except Exception:
                    pass
            try:
                await context_a.unroute("**/wails/runtime", old_route_handler)
            except Exception:
                pass
        contexts = [context for context in (context_a, context_b) if context is not None]
        if contexts:
            await asyncio.gather(*(context.close() for context in contexts), return_exceptions=True)


async def run(args: argparse.Namespace, scratch: Path) -> None:
    server = Path(args.server).resolve(strict=True)
    alpha, beta = prepare_workspaces(scratch)
    roots = (alpha, beta)
    before = fixture_snapshot(roots)
    base_url = f"http://127.0.0.1:{args.port}/"
    parsed = urlparse(base_url)
    assert parsed.scheme == "http" and parsed.hostname == "127.0.0.1", base_url

    log_path = scratch / "wails-server.log"
    log_file = log_path.open("wb")
    process: subprocess.Popen[bytes] | None = None
    try:
        process = subprocess.Popen(
            [
                str(server),
                "-workspace",
                str(alpha),
                "-workspace",
                str(beta),
                "-port",
                str(args.port),
            ],
            cwd=server.parent,
            stdout=log_file,
            stderr=subprocess.STDOUT,
        )
        async with async_playwright() as playwright:
            await wait_for_host(playwright, process, base_url, log_path)
            browser = await playwright.chromium.launch()
            try:
                await exercise_two_contexts(browser, base_url)
            finally:
                await browser.close()
        after = fixture_snapshot(roots)
        assert after == before, "Host or Workbench changed temporary Workspace fixture bytes"
        print(f"PASS_FIXTURE_UNCHANGED files={len(before)}")
    finally:
        if process is not None and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
        log_file.close()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--server", required=True, help="Path to the built Wails Server executable")
    parser.add_argument("--port", type=int, default=43124)
    args = parser.parse_args()
    if not 1 <= args.port <= 65535:
        parser.error("--port must be between 1 and 65535")
    return args


def main() -> None:
    args = parse_args()
    with tempfile.TemporaryDirectory(prefix="tei-studio-browser-isolation-") as scratch:
        asyncio.run(run(args, Path(scratch)))


if __name__ == "__main__":
    main()
