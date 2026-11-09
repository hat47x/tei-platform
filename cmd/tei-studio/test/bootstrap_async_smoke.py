"""Exercise the real Studio Server bootstrap's stale Catalog RPC in Chromium.

This verifies the / bootstrap, not the Desktop default /workbench/ route.
"""

from __future__ import annotations

import argparse
import asyncio
import subprocess
import tempfile
from pathlib import Path

from playwright.async_api import Browser, Page, Route, async_playwright

from browser_isolation_smoke import (
    ALPHA_SUBJECT,
    BETA_SUBJECT,
    CATALOG,
    fixture_snapshot,
    prepare_workspaces,
    runtime_call,
    wait_for_host,
)


async def exercise(browser: Browser, base_url: str) -> None:
    context = await browser.new_context(service_workers="block")
    release = {"workspace-1": asyncio.Event(), "workspace-2": asyncio.Event()}
    captured: dict[str, asyncio.Future[dict]] = {}
    delivered = {"workspace-1": asyncio.Event(), "workspace-2": asyncio.Event()}
    loop = asyncio.get_running_loop()
    for workspace_id in release:
        captured[workspace_id] = loop.create_future()
    page: Page = await context.new_page()

    async def intercept(route: Route) -> None:
        call = runtime_call(route.request)
        if call is None or call["method"] != CATALOG:
            await route.fallback()
            return
        args = call["args"]
        if not args or args[0] not in release:
            await route.fallback()
            return
        workspace_id = args[0]
        try:
            if captured[workspace_id].done():
                raise AssertionError(f"duplicate delayed Catalog RPC: {workspace_id}")
            response = await route.fetch()
            payload = await response.json()
            if response.status != 200 or not isinstance(payload, dict) or not payload.get("ok"):
                raise AssertionError(f"real Host Catalog failed: {workspace_id}: {payload}")
            captured[workspace_id].set_result({"call": call, "payload": payload})
            await release[workspace_id].wait()
            await route.fulfill(response=response)
        except BaseException as exc:
            if not captured[workspace_id].done():
                captured[workspace_id].set_exception(exc)
            raise
        finally:
            delivered[workspace_id].set()

    try:
        await context.route("**/wails/runtime", intercept)
        await page.goto(base_url, wait_until="load", timeout=15_000)
        await page.locator('#workspace option[value="workspace-2"]').wait_for(
            state="attached", timeout=15_000
        )
        assert await page.locator("#workspace").input_value() == "workspace-1"
        assert await page.locator("#state").get_attribute("role") == "status"
        assert await page.locator("#state").get_attribute("aria-live") == "polite"

        await page.locator("#catalog").click()
        old = await asyncio.wait_for(captured["workspace-1"], timeout=15)
        assert old["call"]["args"] == ["workspace-1", ""], old
        assert ALPHA_SUBJECT in str(old["payload"]), old

        # Switching while the old Host result is held invalidates its UI epoch.
        await page.locator("#workspace").select_option("workspace-2")
        assert not await page.locator("#catalog").is_disabled()
        await page.locator("#catalog").click()
        fresh = await asyncio.wait_for(captured["workspace-2"], timeout=15)
        assert fresh["call"]["args"] == ["workspace-2", ""], fresh
        assert BETA_SUBJECT in str(fresh["payload"]), fresh

        # First release the old result. It must neither publish nor re-enable
        # the button while the newer request is still in flight.
        release["workspace-1"].set()
        await asyncio.wait_for(delivered["workspace-1"].wait(), timeout=15)
        await page.evaluate(
            "() => new Promise(resolve => "
            "requestAnimationFrame(() => requestAnimationFrame(resolve)))"
        )
        assert await page.locator("#catalog").is_disabled(), "stale finally unlocked W2"
        old_dom = await page.locator("#output").inner_text()
        assert ALPHA_SUBJECT not in old_dom, old_dom

        release["workspace-2"].set()
        await asyncio.wait_for(delivered["workspace-2"].wait(), timeout=15)
        await page.wait_for_function(
            "(subject) => document.querySelector('#output')?.textContent?.includes(subject)",
            arg=BETA_SUBJECT,
            timeout=15_000,
        )
        new_dom = await page.locator("#output").inner_text()
        assert ALPHA_SUBJECT not in new_dom, new_dom
        assert await page.locator("#workspace").input_value() == "workspace-2"
        assert not await page.locator("#catalog").is_disabled()
        print(
            "PASS_BOOTSTRAP_LATE_CATALOG real Server Host, Chromium DOM/ARIA, "
            "held W1 Host response, W2 request busy, old response discarded, "
            "W2 final result retained"
        )
        print(
            "SCOPE_LIMITATION / bootstrap, not Desktop /workbench/; "
            "no Host grant, Provider effect, or Receipt"
        )
    finally:
        release["workspace-1"].set()
        release["workspace-2"].set()
        await context.close()


async def run(args: argparse.Namespace, scratch: Path) -> None:
    server = Path(args.server).resolve(strict=True)
    roots = prepare_workspaces(scratch)
    before = fixture_snapshot(roots)
    log_path = scratch / "bootstrap-host.log"
    base_url = f"http://127.0.0.1:{args.port}/"
    with log_path.open("wb") as log:
        process = subprocess.Popen(
            [str(server), "-workspace", str(roots[0]), "-workspace", str(roots[1]),
             "-port", str(args.port)],
            cwd=server.parent, stdout=log, stderr=subprocess.STDOUT,
        )
        try:
            async with async_playwright() as playwright:
                await wait_for_host(playwright, process, base_url, log_path)
                browser = await playwright.chromium.launch()
                try:
                    await exercise(browser, base_url)
                finally:
                    await browser.close()
            assert before == fixture_snapshot(roots), "Browser smoke mutated fixture bytes"
        finally:
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--server", required=True)
    parser.add_argument("--port", type=int, default=43130)
    args = parser.parse_args()
    if not 1 <= args.port <= 65535:
        parser.error("invalid port")
    with tempfile.TemporaryDirectory(prefix="tei-studio-bootstrap-async-") as temp:
        asyncio.run(run(args, Path(temp)))


if __name__ == "__main__":
    main()
