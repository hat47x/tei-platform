#!/usr/bin/env python3
"""Derive a static JSON-API form from the actual generated First Web HTML.

Never author another business form schema here. Only change its transport and
introduce a Browser-owned result surface; any unexpected original fails closed.
"""
import pathlib
import sys

if len(sys.argv) != 2:
    raise SystemExit("usage: migrate_browser.py /staged/browser/index.html")
path = pathlib.Path(sys.argv[1])
html = path.read_text("utf-8")
before = '<form method="post" action="/custom-submit" autocomplete="off">'
after = '<form id="json-command" method="post" action="/scenario/submit" autocomplete="off">'
if html.count(before) != 1 or html.count("</form>") != 1:
    raise SystemExit("unexpected generated form; refuse migration")
html = html.replace(before, after, 1)
anchor = '</form>\n</section>'
if html.count(anchor) != 1 or html.count('</body>') != 1:
    raise SystemExit("unexpected generated HTML structure")
html = html.replace(anchor, '</form>\n<section class="notice" id="command-feedback" aria-live="polite" role="status" hidden></section>\n</section>', 1)
html = html.replace("</body>", '<script defer src="/browser-command.js"></script>\n</body>', 1)
if html.count('src="/browser-command.js"') != 1:
    raise SystemExit("missing client transport")
for forbidden in ("editor-secret", "viewer-secret", "reference/legacy-auth/", "workspace.Load",
                  "driver.NewResolver", "sourceMappingURL", "<script>"):
    if forbidden in html:
        raise SystemExit("generated browser contains forbidden source or secret marker")
path.write_text(html, encoding="utf-8")
