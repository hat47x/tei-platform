"""Check appearance changes against a real read-only Studio Host in Chromium."""

import re
import sys
from pathlib import Path
from urllib.parse import urlparse

from playwright.sync_api import sync_playwright


def contrast(foreground, background):
    def luminance(value):
        channels = [int(n) / 255 for n in re.findall(r"\d+", value)[:3]]
        linear = [v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4 for v in channels]
        return sum(v * w for v, w in zip(linear, [0.2126, 0.7152, 0.0722]))
    a, b = sorted([luminance(foreground), luminance(background)])
    return (b + 0.05) / (a + 0.05)


def smoke_appearance(base, screenshot_directory=None):
    parsed = urlparse(base)
    assert parsed.scheme == "http" and parsed.hostname == "127.0.0.1"
    screenshots = Path(screenshot_directory) if screenshot_directory else None
    if screenshots:
        screenshots.mkdir(parents=True, exist_ok=True)
    with sync_playwright() as p:
        browser = p.chromium.launch()
        context = browser.new_context(viewport={"width": 1440, "height": 1000}, color_scheme="light")
        page = context.new_page()
        errors = []
        page.on("pageerror", lambda error: errors.append(str(error)))
        page.goto(base.rstrip("/") + "/workbench/")
        page.locator(".subject").first.wait_for()
        page.locator(".subject").first.click()
        page.wait_for_function("document.querySelector('#source-status').textContent !== '未取得'")
        page.locator("#xead-path").fill("draft/未保存.xead")
        page.evaluate("window.initialEditor = document.querySelector('.cm-editor')")
        title = page.locator("#subject-title").inner_text()
        revision = page.locator("#revision").inner_text()
        selected = page.locator(".subject[aria-pressed=true]").inner_text()
        calls = []
        page.on("request", lambda request: calls.append(request.url) if "/wails/runtime" in request.url else None)

        for theme in ["dark", "light"]:
            page.locator("#appearance-theme").select_option(theme)
            assert page.locator("html").get_attribute("data-tei-theme") == theme
            for selector in [".center", ".primary"]:
                colors = page.locator(selector).evaluate("el => { const s = getComputedStyle(el); return [s.color, s.backgroundColor]; }")
                assert contrast(*colors) >= 4.5, (theme, selector, colors)
            hint = page.locator(".hint").first.evaluate("el => getComputedStyle(el).color")
            background = page.locator(".center").evaluate("el => getComputedStyle(el).backgroundColor")
            assert contrast(hint, background) >= 4.5, (theme, hint, background)
            assert page.evaluate("window.initialEditor === document.querySelector('.cm-editor')")
            assert page.locator("#xead-path").input_value() == "draft/未保存.xead"
            assert page.locator("#subject-title").inner_text() == title
            assert page.locator("#revision").inner_text() == revision
            assert page.locator(".subject[aria-pressed=true]").inner_text() == selected
            if screenshots:
                page.screenshot(path=str(screenshots / f"studio-{theme}.png"), full_page=True)

        row = page.locator(".subject").first
        before = row.bounding_box()["height"]
        page.locator("#appearance-density").select_option("compact")
        assert row.bounding_box()["height"] < before
        assert row.bounding_box()["height"] >= 32
        assert not calls, calls
        page.reload()
        assert page.locator("#appearance-theme").input_value() == "light"
        assert page.locator("#appearance-density").input_value() == "compact"

        # An application CSS override works in both built-in palettes without
        # rebuilding a component or selecting an operation Provider.
        custom = page.add_style_tag(content=":root { --tei-accent: rgb(91, 33, 182); --tei-on-accent: white; }")
        for theme in ["light", "dark"]:
            page.locator("#appearance-theme").select_option(theme)
            assert page.locator(".primary").evaluate("el => getComputedStyle(el).backgroundColor") == "rgb(91, 33, 182)"
        custom.evaluate("el => el.remove()")

        page.locator("#appearance-theme").select_option("system")
        page.locator("#xead-path").fill("draft/編集中.xead")
        page.locator("#xead-path").focus()
        page.emulate_media(color_scheme="dark")
        page.wait_for_function("document.documentElement.dataset.teiTheme === 'dark'")
        assert page.locator("#xead-path").input_value() == "draft/編集中.xead"
        assert page.locator("#xead-path").evaluate("el => el === document.activeElement")
        for width in [1440, 1024, 390]:
            page.set_viewport_size({"width": width, "height": 1000})
            assert page.evaluate("document.documentElement.scrollWidth <= window.innerWidth"), width
        scale = page.add_style_tag(content=":root { --tei-font-size: 28px; }")
        assert page.locator(".primary").evaluate("el => getComputedStyle(el).fontSize") == "28px"
        assert page.evaluate("document.documentElement.scrollWidth <= window.innerWidth")
        scale.evaluate("el => el.remove()")
        if screenshots:
            page.screenshot(path=str(screenshots / "studio-narrow.png"), full_page=True)

        page.evaluate("localStorage.setItem('tei.studio.appearance.v1', JSON.stringify({theme: 'invalid', density: 'invalid'}))")
        page.reload()
        assert page.locator("#appearance-theme").input_value() == "system"
        assert page.locator("#appearance-density").input_value() == "comfortable"
        assert not errors, errors
        context.close()

        # Storage may be disabled by the host/browser; appearance still works.
        blocked = browser.new_context()
        blocked.add_init_script("Object.defineProperty(window, 'localStorage', {get() { throw new Error('storage unavailable'); }});")
        page = blocked.new_page()
        page.goto(base.rstrip("/") + "/workbench/")
        page.locator("#appearance-theme").select_option("dark")
        assert page.locator("html").get_attribute("data-tei-theme") == "dark"
        blocked.close()
        browser.close()
    print("PASS: theme/density, state retention, CSS override, contrast samples, responsive layout, system preference, storage fallback")


if __name__ == "__main__":
    smoke_appearance(sys.argv[1], sys.argv[2] if len(sys.argv) > 2 else None)
