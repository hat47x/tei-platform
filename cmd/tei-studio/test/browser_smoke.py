"""Exercise the embedded Studio bootstrap through a real headless browser."""

import json
import sys
from urllib.parse import urlparse

from playwright.sync_api import sync_playwright


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: browser_smoke.py http://127.0.0.1:PORT/")

    base_url = sys.argv[1]
    base = urlparse(base_url)
    assert base.scheme == "http" and base.hostname == "127.0.0.1", base_url

    runtime_module = None
    runtime_request = None
    runtime_response = None

    with sync_playwright() as playwright:
        browser = playwright.chromium.launch()
        try:
            page = browser.new_page()

            def capture_response(response):
                nonlocal runtime_module, runtime_response
                path = urlparse(response.url).path
                if path == "/wails/runtime.js":
                    runtime_module = response
                elif path == "/wails/runtime":
                    runtime_response = response

            def capture_request(request):
                nonlocal runtime_request
                if urlparse(request.url).path == "/wails/runtime":
                    runtime_request = request

            page.on("response", capture_response)
            page.on("request", capture_request)
            page.goto(base_url, wait_until="load", timeout=10_000)
            page.wait_for_function(
                "document.querySelector('#workspace option')?.value === 'workspace-1'",
                timeout=10_000,
            )

            assert runtime_module is not None, "embedded page did not load /wails/runtime.js"
            assert runtime_module.status == 200, runtime_module.status
            assert runtime_request is not None, "Call.ByName did not issue a Wails runtime request"
            assert runtime_request.method == "POST", runtime_request.method
            request_body = runtime_request.post_data_json
            runtime_call = request_body["args"]
            assert runtime_call["methodName"] == "main.StudioBridge.ListWorkspaces", request_body
            assert runtime_call["args"] == [], request_body

            assert runtime_response is not None, "Wails runtime request did not receive a response"
            assert runtime_response.status == 200, runtime_response.status
            envelope = runtime_response.json()
            assert envelope.get("ok") is True, envelope
            assert envelope.get("value") == [{"id": "workspace-1", "label": "workspace"}], envelope

            option = page.locator("#workspace option").first
            assert option.get_attribute("value") == "workspace-1"
            assert "workspace (workspace-1)" in option.inner_text()
            assert "Operation応答" in page.locator("#state").inner_text()
            displayed = json.loads(page.locator("#output").inner_text())
            assert displayed == envelope, displayed
        finally:
            browser.close()


if __name__ == "__main__":
    main()
