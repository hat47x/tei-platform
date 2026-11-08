#!/usr/bin/env python3
"""M0→M1: compare actual TEI CLI A/J with real Workbench Initial/List events.

Public, ephemeral Source only. This test never injects a response adapter:
CLI and Studio are separate real Producers, compared before one Browser consumes
Studio's real Wails RPC. It does not prove atomic reads or production authority.
"""

from __future__ import annotations

import argparse
import asyncio
import hashlib
import json
import subprocess
import tempfile
from pathlib import Path

from playwright.async_api import async_playwright

from browser_isolation_smoke import (
    CATALOG,
    INSPECT,
    VALIDATE,
    RPCRecorder,
    assert_success,
    wait_for_host,
)
from cli_catalog_parity import call_catalog, free_loopback_port, normalize_catalog

A = {"kind": "semantic-data", "id": "sales.order"}
J = {"kind": "semantic-data", "id": "sales.unique"}
WORKSPACE_ID = "workspace-1"


def create_fixture(root: Path) -> None:
    semantic = root / "semantic"
    semantic.mkdir(parents=True)
    (root / "tei.workspace.yaml").write_text(
        "workspace: m0m1_real_cli_selection\nformatVersion: 1\nsources:\n  - semantic\n",
        encoding="utf-8",
    )
    # Keep the hard-coded Workbench Initial target sales.order ambiguous,
    # while sales.unique is an unrelated, uniquely inspectable List target.
    (semantic / "a.tei").write_text(
        "namespace sales\n\n"
        "data Order {\n    attribute orderId type std.identifier\n}\n\n"
        "data Unique {\n    attribute uniqueId type std.identifier\n}\n",
        encoding="utf-8",
    )
    (semantic / "b.tei").write_text(
        "namespace sales\n\n"
        "data Order {\n    attribute duplicateId type std.identifier\n}\n",
        encoding="utf-8",
    )


def digest_workspace(root: Path) -> dict[str, str]:
    return {
        p.relative_to(root).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
        for p in sorted(root.rglob("*")) if p.is_file()
    }


def cli_json(cli: Path, *args: str, expected_exit: int | None = 0) -> tuple[dict, str]:
    proc = subprocess.run([str(cli), *args], capture_output=True, check=False)
    digest = hashlib.sha256(proc.stdout).hexdigest()
    try:
        result = json.loads(proc.stdout)
    except ValueError as error:
        raise AssertionError(f"real CLI returned non-JSON: {args}, exit={proc.returncode}") from error
    assert isinstance(result, dict), result
    if expected_exit is not None:
        assert proc.returncode == expected_exit, (args, proc.returncode, result)
    return result, digest


def observe_cli(cli: Path, root: Path, expected_head: str) -> tuple[dict, dict, dict]:
    version, version_sha256 = cli_json(cli, "version", "-json")
    assert version.get("vcsRevision") == expected_head, version
    assert version.get("vcsModified", False) is False, version
    catalog_envelope, catalog_sha256 = cli_json(
        cli, "model", "catalog", "-workspace", str(root)
    )
    assert catalog_envelope.get("schemaVersion") == "tei.cli.model/v0", catalog_envelope
    assert catalog_envelope.get("ok") is True, catalog_envelope
    catalog = catalog_envelope.get("result")
    assert isinstance(catalog, dict) and catalog.get("effect") == "read-only", catalog
    entries = catalog.get("entries")
    assert isinstance(entries, list), catalog
    a_rows = [e for e in entries if e.get("subject") == A]
    j_rows = [e for e in entries if e.get("subject") == J]
    assert len(a_rows) == 2 and all(e.get("ambiguous") is True for e in a_rows), a_rows
    assert len(j_rows) == 1 and j_rows[0].get("ambiguous") is not True, j_rows
    assert catalog.get("modelRevision") and catalog.get("worldFingerprint") and catalog.get("sourceIndexFingerprint"), catalog

    unique_envelope, unique_sha256 = cli_json(
        cli, "inspect", "-workspace", str(root),
        "-subject-kind", J["kind"], "-subject-id", J["id"],
    )
    assert unique_envelope.get("schemaVersion") == "tei.cli.inspect/v0", unique_envelope
    assert unique_envelope.get("ok") is True, unique_envelope
    unique = unique_envelope.get("result")
    assert isinstance(unique, dict) and unique.get("subject") == J, unique
    for field in ("worldFingerprint", "sourceIndexFingerprint"):
        assert unique.get(field) == catalog.get(field), (field, unique, catalog)

    ambiguous_envelope, ambiguous_sha256 = cli_json(
        cli, "inspect", "-workspace", str(root),
        "-subject-kind", A["kind"], "-subject-id", A["id"],
        expected_exit=None,
    )
    assert ambiguous_envelope.get("ok") is False, ambiguous_envelope
    assert ambiguous_envelope.get("result") is None, ambiguous_envelope
    assert isinstance(ambiguous_envelope.get("error", {}).get("code"), str), ambiguous_envelope
    print(
        "ACTUAL_CLI_PRODUCER",
        json.dumps({
            "expectedHead": expected_head,
            "versionStdoutSHA256": version_sha256,
            "catalogStdoutSHA256": catalog_sha256,
            "uniqueInspectStdoutSHA256": unique_sha256,
            "ambiguousInspectStdoutSHA256": ambiguous_sha256,
            "catalogEntriesA": len(a_rows),
            "catalogEntriesJ": len(j_rows),
            "modelRevision": catalog["modelRevision"],
            "worldFingerprint": catalog["worldFingerprint"],
            "sourceIndexFingerprint": catalog["sourceIndexFingerprint"],
        }, sort_keys=True),
    )
    return catalog, unique, ambiguous_envelope


async def inspect_events(browser, base_url: str, cli_catalog: dict, cli_unique: dict) -> None:
    context = await browser.new_context(service_workers="block")
    try:
        page = await context.new_page()
        # Attach *before* navigation so even Initial requests are recorded.
        recorder = RPCRecorder(page)
        navigation = await page.goto(base_url + "workbench/", wait_until="load", timeout=15_000)
        assert navigation is not None and navigation.status == 200, navigation
        catalog_record = await recorder.wait_for(CATALOG, [WORKSPACE_ID, ""])
        live_catalog = assert_success(catalog_record, CATALOG, [WORKSPACE_ID, ""])["value"]
        assert normalize_catalog(cli_catalog) == normalize_catalog(live_catalog), (
            "actual CLI / actual Wails Catalog producer mismatch", cli_catalog, live_catalog
        )
        await recorder.wait_for(VALIDATE, [WORKSPACE_ID])
        await page.wait_for_function(
            """() => document.querySelector('#workspace')?.value === 'workspace-1'
              && document.querySelector('#subject-title')?.textContent === 'sales.order'
              && document.querySelector('#source-status')?.textContent === '対象未確定'
              && document.querySelector('#editor .cm-content')?.textContent?.includes('SUBJECT_AMBIGUOUS')""",
            timeout=15_000,
        )
        # Workbench explicitly prioritizes sales.order as Initial, even when
        # its two Catalog rows are ambiguous. No unqualified Inspect allowed.
        assert not [c for c in recorder.calls if c["method"] == INSPECT], recorder.calls
        assert "対象未確定" in await page.locator("#subjects").inner_text()

        await page.locator("#search").fill(A["id"])
        # Search also matches child IDs, so select exact semantic-data rows.
        candidates = page.locator("#subjects button")
        a_indexes = await candidates.evaluate_all(
            "(nodes, id) => nodes.flatMap((node, index) => "
            "node.querySelector('small')?.textContent === "
            "`semantic-data · ${id} · 対象未確定` ? [index] : [])",
            A["id"],
        )
        assert len(a_indexes) == 2, (a_indexes, await candidates.all_text_contents())
        before = recorder.request_count
        await candidates.nth(a_indexes[0]).click()
        await page.wait_for_function(
            """() => document.querySelector('#source-status')?.textContent === '対象未確定'
              && document.querySelector('#editor .cm-content')?.textContent?.includes('SUBJECT_AMBIGUOUS')"""
        )
        assert not [c for c in recorder.calls[before:] if c["method"] == INSPECT], recorder.calls[before:]

        # An unrelated unique J still works, in *the same* Browser page.
        await page.locator("#search").fill(J["id"])
        unique_buttons = page.locator("#subjects button")
        j_indexes = await unique_buttons.evaluate_all(
            "(nodes, id) => nodes.flatMap((node, index) => "
            "node.querySelector('small')?.textContent === "
            "`semantic-data · ${id}` ? [index] : [])",
            J["id"],
        )
        assert len(j_indexes) == 1, (j_indexes, await unique_buttons.all_text_contents())
        before = recorder.request_count
        await unique_buttons.nth(j_indexes[0]).click()
        record = await recorder.wait_for(
            INSPECT, [WORKSPACE_ID, J["kind"], J["id"]], after_request=before
        )
        live_unique = assert_success(record, INSPECT, [WORKSPACE_ID, J["kind"], J["id"]])["value"]
        inspect_calls = [c for c in recorder.calls[before:] if c["method"] == INSPECT]
        assert len(inspect_calls) == 1 and inspect_calls[0]["args"] == [WORKSPACE_ID, J["kind"], J["id"]], inspect_calls
        assert live_unique.get("canonical") == cli_unique.get("canonical"), (live_unique, cli_unique)
        for field in ("worldFingerprint", "sourceIndexFingerprint"):
            assert live_unique.get(field) == cli_unique.get(field), field
        await page.wait_for_function(
            """() => document.querySelector('#subject-title')?.textContent === 'sales.unique'
              && document.querySelector('#source-status')?.textContent?.startsWith('参照情報：')""",
            timeout=10_000,
        )
        editor = await page.locator("#editor .cm-content").inner_text()
        assert json.loads(editor) == cli_unique["canonical"], editor
        await recorder.drain()
        assert not recorder.duplicate_call_ids, recorder.duplicate_call_ids
        assert len([c for c in recorder.calls if c["method"] == INSPECT]) == 1, recorder.calls
        print("PASS_REAL_WORKBENCH_EVENTS Initial(A)=0, List(A)=0, List(J)=1 InspectSubject")
        print("PROVENANCE real CLI stdout + real Wails RPC + real Workbench DOM; testAdapterUsed=false")
        print("SCOPE_NOT_PROVEN atomic Source snapshot, current production Authority/Grant/Policy, external Source, Owner acceptance")
    finally:
        await context.close()


async def run(args: argparse.Namespace, scratch: Path) -> None:
    cli = Path(args.cli).resolve(strict=True)
    server = Path(args.server).resolve(strict=True)
    root = scratch / "workspace"
    create_fixture(root)
    original_bytes = digest_workspace(root)
    catalog, unique, _ = observe_cli(cli, root, args.expected_head)
    port = free_loopback_port()
    base_url = f"http://127.0.0.1:{port}/"
    log_path = scratch / "wails-server.log"
    process = None
    with log_path.open("wb") as log_file:
        try:
            process = subprocess.Popen(
                [str(server), "-workspace", str(root), "-port", str(port)],
                cwd=server.parent,
                stdout=log_file,
                stderr=subprocess.STDOUT,
            )
            async with async_playwright() as playwright:
                await wait_for_host(playwright, process, base_url, log_path)
                # Parity is between actual producers, not an adapter mapping.
                live_catalog = call_catalog(base_url.rstrip("/"), WORKSPACE_ID, "", "m0m1-direct-producer")
                assert live_catalog.get("ok") is True, live_catalog
                assert normalize_catalog(live_catalog.get("value", {})) == normalize_catalog(catalog)
                browser = await playwright.chromium.launch()
                try:
                    await inspect_events(browser, base_url, catalog, unique)
                finally:
                    await browser.close()
        finally:
            if process is not None and process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)
    assert digest_workspace(root) == original_bytes, "temporary fixture Source was mutated"
    print(f"PASS_SOURCE_UNCHANGED files={len(original_bytes)}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--cli", required=True)
    parser.add_argument("--server", required=True)
    parser.add_argument("--expected-head", required=True)
    args = parser.parse_args()
    assert len(args.expected_head) == 40 and all(c in "0123456789abcdef" for c in args.expected_head)
    with tempfile.TemporaryDirectory(prefix="tei-m0m1-real-cli-") as temp:
        asyncio.run(run(args, Path(temp)))


if __name__ == "__main__":
    main()
