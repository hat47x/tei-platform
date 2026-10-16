"""Exercise the embedded bootstrap and Workbench through a real browser."""

import json
import sys
from urllib.parse import urljoin, urlparse

from playwright.sync_api import sync_playwright


def runtime_call(request):
    if urlparse(request.url).path != "/wails/runtime" or request.method != "POST":
        return None
    body = request.post_data_json
    if not isinstance(body, dict):
        return None
    call = body.get("args")
    if not isinstance(call, dict):
        return None
    method = call.get("methodName")
    if not isinstance(method, str):
        return None
    return method, call.get("args")


def has_runtime_method(response, method):
    call = runtime_call(response.request)
    return call is not None and call[0] == method


def successful_response(response):
    assert response.status == 200, response.status
    envelope = response.json()
    assert envelope.get("ok") is True, envelope
    return envelope


def validation_counts(envelope):
    value = envelope.get("value")
    assert isinstance(value, dict), value
    assert "Direct" in value and "Diagnostics" in value, value
    diagnostics = []
    if value["Direct"] is not None:
        diagnostics.append(value["Direct"])
    projected = value["Diagnostics"]
    assert projected is None or isinstance(projected, list), projected
    for item in projected or []:
        assert (
            isinstance(item, dict)
            and isinstance(item.get("diagnostic"), dict)
        ), item
        diagnostics.append(item["diagnostic"])

    counts = {"error": 0, "warning": 0}
    for diagnostic in diagnostics:
        assert isinstance(diagnostic, dict), diagnostic
        severity = diagnostic.get("severity")
        assert severity in ("info", "warning", "error"), diagnostic
        if severity in counts:
            counts[severity] += 1
    return counts


def smoke_bootstrap(browser, base_url):
    runtime_module = None
    runtime_response = None
    page = browser.new_page()

    def capture_response(response):
        nonlocal runtime_module, runtime_response
        path = urlparse(response.url).path
        if path == "/wails/runtime.js":
            runtime_module = response
        if has_runtime_method(response, "main.StudioBridge.ListWorkspaces"):
            runtime_response = response

    page.on("response", capture_response)
    page.goto(base_url, wait_until="load", timeout=10_000)
    page.wait_for_function(
        "document.querySelector('#workspace option')?.value === 'workspace-1'",
        timeout=10_000,
    )

    assert runtime_module is not None, "bootstrap did not load /wails/runtime.js"
    assert runtime_module.status == 200, runtime_module.status
    assert runtime_response is not None, "bootstrap did not call ListWorkspaces"
    call = runtime_call(runtime_response.request)
    assert call == ("main.StudioBridge.ListWorkspaces", []), call
    envelope = successful_response(runtime_response)
    assert envelope.get("value") == [{"id": "workspace-1", "label": "workspace"}], envelope

    option = page.locator("#workspace option").first
    assert option.get_attribute("value") == "workspace-1"
    assert "workspace (workspace-1)" in option.inner_text()
    assert "Operation応答" in page.locator("#state").inner_text()
    displayed = json.loads(page.locator("#output").inner_text())
    assert displayed == envelope, displayed
    print("Bootstrap browser smoke: ListWorkspaces args=[]; workspace-1 returned")
    page.close()


def smoke_workbench(browser, base_url):
    calls = []
    responses = {}
    assets = {}
    page = browser.new_page()

    def capture_request(request):
        call = runtime_call(request)
        if call is not None:
            calls.append(call)

    def capture_response(response):
        path = urlparse(response.url).path
        if path in {
            "/workbench/main.js",
            "/workbench/style.css",
            "/wails/runtime.js",
        }:
            assets[path] = response
        call = runtime_call(response.request)
        if call is not None:
            responses[call[0]] = response

    page.on("request", capture_request)
    page.on("response", capture_response)
    navigation = page.goto(
        urljoin(base_url, "workbench/"), wait_until="load", timeout=10_000
    )
    assert navigation is not None and navigation.status == 200
    page.wait_for_function(
        """() => document.querySelector("#connection")?.textContent === "実Hostに接続"
        && document.querySelector("#workspace")?.value === "workspace-1"
        && document.querySelector("#subject-title")?.textContent === "sales.order"
        && document.querySelector("#source-status")?.textContent?.startsWith("参照情報：")"""
    )
    page.wait_for_function(
        "document.querySelector('#notice')?.textContent?.includes('検証') === true",
        timeout=10_000,
    )

    for path in ("/workbench/main.js", "/workbench/style.css", "/wails/runtime.js"):
        assert path in assets, "Workbench did not load " + path
        assert assets[path].status == 200, (path, assets[path].status)

    expected_calls = {
        "main.StudioBridge.ListWorkspaces": [],
        "main.StudioBridge.CatalogWorkspace": ["workspace-1", ""],
        "main.StudioBridge.InspectSubject": [
            "workspace-1",
            "semantic-data",
            "sales.order",
        ],
        "main.StudioBridge.ValidateWorkspace": ["workspace-1"],
    }
    envelopes = {}
    for method, args in expected_calls.items():
        assert (method, args) in calls, (method, args, calls)
        assert method in responses, (method, calls)
        envelopes[method] = successful_response(responses[method])

    initial_inspection = envelopes["main.StudioBridge.InspectSubject"]["value"]
    assert initial_inspection["effect"] == "read-only", initial_inspection
    assert initial_inspection["subject"] == {
        "kind": "semantic-data",
        "id": "sales.order",
    }, initial_inspection
    assert isinstance(initial_inspection["canonical"], dict), initial_inspection
    displayed_canonical = json.loads(
        page.locator("#editor .cm-content").inner_text()
    )
    assert displayed_canonical == initial_inspection["canonical"], displayed_canonical

    counts = validation_counts(envelopes["main.StudioBridge.ValidateWorkspace"])
    expected_summary = f"エラー {counts['error']}・警告 {counts['warning']}"
    assert page.locator("#diagnostic-summary").inner_text() == expected_summary
    expected_notice = (
        f"検証で{counts['error']}件のエラーが見つかりました。"
        if counts["error"]
        else f"検証結果：エラーなし、警告{counts['warning']}件。"
    )
    assert page.locator("#notice").inner_text() == expected_notice

    editor = page.locator("#editor .cm-content")
    assert editor.count() == 1, "CodeMirror editor did not mount"
    assert editor.get_attribute("contenteditable") == "false"
    assert page.locator('#graph svg[aria-label="読取り専用グラフ"]').count() == 1
    assert page.locator("#graph g[role='button']").count() > 1
    initial_subject = {"kind": "semantic-data", "id": "sales.order"}
    catalog_value = envelopes["main.StudioBridge.CatalogWorkspace"]["value"]
    catalog_entries = catalog_value.get("entries")
    assert isinstance(catalog_entries, list), catalog_value
    target_entries = [
        entry
        for entry in catalog_entries
        if entry.get("parent") == initial_subject and entry.get("symbol") == "orderId"
    ]
    assert len(target_entries) == 1, target_entries
    target_entry = target_entries[0]
    target_subject = target_entry["subject"]
    target_node = page.get_by_role(
        "button", name=target_entry["symbol"], exact=True
    )
    assert target_node.count() == 1, "graph does not expose the orderId child node"
    command_ids = page.locator("#commands [data-command-id]").evaluate_all(
        "(nodes) => nodes.map((node) => node.dataset.commandId).sort()"
    )
    assert command_ids == ["inspect", "refresh", "validate"], command_ids

    with page.expect_response(
        lambda response: has_runtime_method(
            response, "main.StudioBridge.InspectSubject"
        ),
        timeout=10_000,
    ) as graph_inspection:
        target_node.click()
    graph_call = runtime_call(graph_inspection.value.request)
    assert graph_call is not None, graph_call
    graph_args = graph_call[1]
    assert graph_args == [
        "workspace-1",
        target_subject["kind"],
        target_subject["id"],
    ], graph_call
    graph_subject = {"kind": graph_args[1], "id": graph_args[2]}
    is_direct_parent_or_child = any(
        (entry.get("subject") == graph_subject and entry.get("parent") == initial_subject)
        or (
            entry.get("subject") == initial_subject
            and entry.get("parent") == graph_subject
        )
        for entry in catalog_entries
    )
    assert is_direct_parent_or_child, (initial_subject, graph_subject)
    graph_envelope = successful_response(graph_inspection.value)
    graph_value = graph_envelope.get("value")
    assert isinstance(graph_value, dict), graph_value
    assert graph_value.get("effect") == "read-only", graph_value
    assert graph_value.get("subject") == graph_subject, graph_value
    page.wait_for_function(
        """(subjectId) =>
          document.querySelector('#subject-title')?.textContent === subjectId
          && document.querySelector('#source-status')?.textContent?.startsWith('参照情報：')""",
        arg=graph_args[2],
        timeout=10_000,
    )
    page.wait_for_function(
        """(expected) => {
          const text = document.querySelector('#editor .cm-content')?.innerText;
          try {
            return JSON.stringify(JSON.parse(text)) === JSON.stringify(expected);
          } catch {
            return false;
          }
        }""",
        arg=graph_value.get("canonical"),
        timeout=10_000,
    )
    displayed_graph_projection = json.loads(editor.inner_text())
    assert (
        displayed_graph_projection == graph_value.get("canonical")
    ), displayed_graph_projection

    validate_button = page.locator(
        '#commands button[data-command-id="validate"]'
    )
    page.wait_for_function(
        """() => {
          const button = document.querySelector(
            '#commands button[data-command-id="validate"]'
          );
          return button && !button.disabled;
        }""",
        timeout=10_000,
    )
    with page.expect_response(
        lambda response: has_runtime_method(
            response, "main.StudioBridge.ValidateWorkspace"
        ),
        timeout=10_000,
    ) as validation:
        validate_button.click()
    validation_envelope = successful_response(validation.value)
    assert validation_counts(validation_envelope) == counts, validation_envelope
    print(
        "Workbench browser smoke: "
        f"Catalog parent={initial_subject['kind']}:{initial_subject['id']} "
        f"child={target_subject['kind']}:{target_subject['id']} "
        f"symbol={target_entry['symbol']}; "
        f"Inspect args={graph_args!r} effect={graph_value['effect']} "
        f"canonical={graph_value['subject']['kind']}:{graph_value['subject']['id']} "
        "matched editor; "
        f"Validate ok={validation_envelope['ok']} "
        f"errors={counts['error']} warnings={counts['warning']} "
        f"summary={page.locator('#diagnostic-summary').inner_text()!r}"
    )
    page.close()


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: browser_smoke.py http://127.0.0.1:PORT/")

    base_url = sys.argv[1]
    base = urlparse(base_url)
    assert base.scheme == "http" and base.hostname == "127.0.0.1", base_url
    assert base.path in ("", "/"), base_url

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        try:
            smoke_bootstrap(browser, base_url)
            smoke_workbench(browser, base_url)
        finally:
            browser.close()


if __name__ == "__main__":
    main()
