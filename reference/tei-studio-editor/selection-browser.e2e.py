"""Real CodeMirror keyboard-selection E2E; requires the compiled test-only page.

Run: python selection-browser.e2e.py http://127.0.0.1:4179/
This MUST NOT be counted as PASS when dependencies/browser cannot launch.
"""
import json
import sys
from playwright.sync_api import sync_playwright

URL = sys.argv[1] if len(sys.argv) > 1 else "http://127.0.0.1:4179/"
with sync_playwright() as p:
    browser = p.chromium.launch(headless=True)
    try:
        page = browser.new_page()
        page.goto(URL, wait_until="load")
        page.wait_for_function("Boolean(window.__editorSelectionProbe)", timeout=5000)
        initial = page.evaluate("""() => {
          const p = window.__editorSelectionProbe;
          return {text: p.editor.snapshot().text,
            contenteditable: p.content.getAttribute('contenteditable'),
            diag: [...p.host.querySelectorAll('.cm-diagnosticText')].map(n => n.textContent)};
        }""")
        assert initial["text"] == "a😀x\r\nsecond🧪!", initial
        assert initial["contenteditable"] == "false", initial

        page.evaluate("""() => {
          const p = window.__editorSelectionProbe;
          p.editor.select({start:{line:2,column:1},end:{line:2,column:10},unit:'utf16-unit'});
          if (JSON.stringify(p.editor.snapshot().selection) !== '{"from":6,"to":15}')
            throw Error('HOST_SELECTION_MISMATCH');
          if (p.events.length) throw Error('PROGRAMMATIC_SELECTION_EMITTED_CALLBACK');
          p.placeCaret(); p.clearEvents();
        }""")
        focused = page.evaluate("""() => {
          const p = window.__editorSelectionProbe;
          return document.activeElement === p.content || p.content.contains(document.activeElement);
        }""")
        assert focused, "CodeMirror content did not receive focus"
        page.keyboard.press("Shift+End")  # not locator.fill()/type() on noneditable div
        result = page.evaluate("""() => {
          const p = window.__editorSelectionProbe;
          return {keys: [...p.keys], events: [...p.events],
            selection: p.editor.snapshot().selection,
            unchanged: p.editor.snapshot().text === p.text,
            nativeText: window.getSelection()?.toString() ?? ''};
        }""")
        print(json.dumps(result, ensure_ascii=True))
        assert "End" in result["keys"], "Shift+End did not reach CodeMirror"
        assert result["events"], "No user-originated selection callback (NOT PASS)"
        assert result["events"][-1]["selection"] == {"from": 6, "to": 15}, result
        assert result["unchanged"], "Read-only editor text changed"
        for label, trigger in (
            ("text input", lambda: page.keyboard.type("x")),
            ("Backspace", lambda: page.keyboard.press("Backspace")),
            ("Delete", lambda: page.keyboard.press("Delete")),
        ):
            trigger()
            unchanged = page.evaluate("""() => {
              const p = window.__editorSelectionProbe;
              return p.editor.snapshot().text === p.text;
            }""")
            assert unchanged, f"{label} changed read-only source"
        print("PASS: real CodeMirror keyboard selection callback roundtrip 6..15; text input/deletion remain read-only")
    finally:
        browser.close()
