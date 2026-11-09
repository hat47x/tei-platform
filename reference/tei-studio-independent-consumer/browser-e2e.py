"""Real Chromium DOM qualification of the built, independent journal consumer.

Requires Playwright Python and an installed Chromium. This test never mocks
CodeMirror, Graph or Command and does not exercise TEI Studio or Wails WebView.
"""
import json
import os
from pathlib import Path
from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from threading import Thread
from playwright.sync_api import sync_playwright

HERE = Path(__file__).resolve().parent
DIST = HERE / "dist"


class ModuleHandler(SimpleHTTPRequestHandler):
    extensions_map = {**SimpleHTTPRequestHandler.extensions_map, ".mjs": "text/javascript"}

    def log_message(self, format, *args):
        pass


def main():
    if not (DIST / "index.html").is_file():
        raise SystemExit("BLOCKED: run npm install && npm run build in a fixed snapshot")
    server = ThreadingHTTPServer(
        ("127.0.0.1", 0), partial(ModuleHandler, directory=str(DIST))
    )
    background = Thread(target=server.serve_forever, daemon=True)
    background.start()
    try:
        with sync_playwright() as playwright:
            options = {"headless": True}
            if os.getenv("TEI_STUDIO_CHROMIUM"):
                options["executable_path"] = os.environ["TEI_STUDIO_CHROMIUM"]
            browser = playwright.chromium.launch(**options)
            try:
                page = browser.new_page(viewport={"width": 900, "height": 800})
                errors = []
                page.on("pageerror", lambda error: errors.append(str(error)))
                page.goto(f"http://127.0.0.1:{server.server_port}/", wait_until="networkidle")
                editor = page.locator("#editor-slot .cm-editor")
                editor.wait_for()
                assert page.locator("#editor-slot .cm-content").get_attribute("contenteditable") == "false"
                assert "Neighborhood walk" in editor.inner_text()
                assert page.locator("#editor-slot").get_attribute("data-document-id") == "journal:map"
                page.select_option("#note-picker", "sketch")
                assert "Window sketch" in editor.inner_text()
                assert page.locator("#editor-slot").get_attribute("data-document-id") == "journal:sketch"

                graph = page.locator("#graph-slot [data-graph-reference]")
                graph.wait_for()
                geometry = graph.evaluate("""el => {
                    const svg = el.querySelector('svg');
                    return {
                        width: svg.getBoundingClientRect().width,
                        clientWidth: el.clientWidth, scrollWidth: el.scrollWidth,
                        fontSize: getComputedStyle(svg).fontSize
                    };
                }""")
                assert geometry["width"] >= 480, geometry
                assert geometry["scrollWidth"] > geometry["clientWidth"], geometry
                assert geometry["fontSize"] == "14px", geometry
                page.locator('#graph-slot [aria-label="Tea tasting"][role="button"]').click()
                assert page.locator("#note-picker").input_value() == "tea"
                assert "Tea tasting" in editor.inner_text()

                page.get_by_role("button", name="Count related entries").click()
                page.wait_for_function("""() => {
                    const s = document.querySelector('#command-slot [role=status]');
                    return s && s.textContent.includes('2 related entries');
                }""")
                page.get_by_role("button", name="Delete entry (denied)").click()
                page.wait_for_function("""() => {
                    const s = document.querySelector('#command-slot [role=status]');
                    return s && s.textContent.includes('拒否');
                }""")
                # Three separate unmount/remount contracts, using the *same*
                # actual common modules as the Studio workbench.
                page.locator("#dispose-editor").click()
                assert page.locator("#editor-slot .cm-editor").count() == 0
                assert page.locator("#editor-slot").get_attribute("data-lifecycle") == "disposed"
                page.select_option("#note-picker", "map")
                assert page.locator("#editor-slot .cm-editor").count() == 0
                page.locator("#remount-editor").click()
                assert page.locator("#editor-slot .cm-editor").count() == 1
                assert page.locator("#editor-slot").get_attribute("data-document-id") == "journal:map"
                page.select_option("#note-picker", "sketch")
                assert "Window sketch" in page.locator("#editor-slot .cm-editor").inner_text()

                page.locator("#dispose-graph").click()
                assert page.locator("#graph-slot [data-graph-reference]").count() == 0
                page.locator("#remount-graph").click()
                assert page.locator("#graph-slot [data-graph-reference]").count() == 1
                # Remount selects the current consumer note; the callback still
                # updates both the picker and actual CodeMirror projection.
                page.locator('#graph-slot [aria-label="Tea tasting"][role="button"]').click()
                assert page.locator("#note-picker").input_value() == "tea"
                assert "Tea tasting" in page.locator("#editor-slot .cm-editor").inner_text()
                page.locator('#graph-slot [aria-label="Window sketch"][role="button"]').focus()
                page.locator('#graph-slot [aria-label="Window sketch"][role="button"]').press("Enter")
                assert page.locator("#note-picker").input_value() == "sketch"

                page.locator("#dispose-command").click()
                assert page.locator("#command-slot .tei-command-panel").count() == 0
                page.locator("#remount-command").click()
                assert page.locator("#command-slot .tei-command-panel").count() == 1
                page.get_by_role("button", name="Count related entries").click()
                page.wait_for_function("""() => {
                    const s = document.querySelector('#command-slot [role=status]');
                    return s && s.textContent.includes('0 related entries');
                }""")
                page.evaluate("window.dispatchEvent(new Event('pagehide'))")
                assert page.locator("#editor-slot .cm-editor").count() == 0
                assert page.locator("#graph-slot [data-graph-reference]").count() == 0
                assert page.locator("#command-slot .tei-command-panel").count() == 0
                assert errors == [], errors
                print(json.dumps({"result": "PASS", "browser": browser.version,
                                  "graphGeometry": geometry, "tested": [
                                      "CodeMirror readOnly", "setDocument",
                                      "Graph min-width and selection",
                                      "Command success/denied", "all components destroy/remount/dispose", "Graph keyboard selection"]}))
            finally:
                browser.close()
    finally:
        server.shutdown()
        server.server_close()


if __name__ == "__main__":
    main()
