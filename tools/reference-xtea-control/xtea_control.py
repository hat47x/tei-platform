#!/usr/bin/env python3
"""AI-friendly client for the X-TEA Swing control agent.

The client uses only the Python standard library. It reads the endpoint and
token file written by the Java agent, so no GUI automation library, display
coordinate, or screenshot OCR is needed.

Typical use:

    python xtea_control.py --endpoint .xtea-control.json windows
    python xtea_control.py --endpoint .xtea-control.json snapshot --table
    python xtea_control.py --endpoint .xtea-control.json invoke \\
        --role menuitem --text Open
    python xtea_control.py --endpoint .xtea-control.json scenario scenario.json
"""

from __future__ import annotations

import argparse
import base64
import json
import re
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any, Dict, List, Optional


class ControlError(RuntimeError):
    def __init__(self, code: str, message: str, status: int = 0) -> None:
        super().__init__(f"{code}: {message}")
        self.code = code
        self.message = message
        self.status = status


class ControlClient:
    def __init__(self, url: str, token: str, timeout: float = 15.0) -> None:
        self.url = url.rstrip("/")
        self.token = token
        self.timeout = timeout

    @classmethod
    def from_endpoint_file(cls, path: str | Path, timeout: float = 15.0) -> "ControlClient":
        endpoint = json.loads(Path(path).read_text(encoding="utf-8"))
        return cls(endpoint["url"], endpoint["token"], timeout=timeout)

    def __enter__(self) -> "ControlClient":
        return self

    def __exit__(self, *_: object) -> None:
        pass

    def request(self, method: str, route: str, body: Optional[Dict[str, Any]] = None) -> Any:
        data = None
        headers = {"X-Xtea-Token": self.token, "Accept": "application/json"}
        if body is not None:
            data = json.dumps(body, ensure_ascii=False).encode("utf-8")
            headers["Content-Type"] = "application/json; charset=utf-8"
        request = urllib.request.Request(self.url + route, data=data, headers=headers, method=method)
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                payload = json.loads(response.read().decode("utf-8"))
        except urllib.error.HTTPError as error:
            raw = error.read().decode("utf-8", "replace")
            try:
                payload = json.loads(raw)
            except json.JSONDecodeError:
                raise ControlError("HTTP_ERROR", raw[:500], error.code) from error
            failure = payload.get("error", {})
            raise ControlError(failure.get("code", "HTTP_ERROR"),
                               failure.get("message", raw[:500]), error.code) from error
        except urllib.error.URLError as error:
            raise ControlError("CONNECTION_FAILED", str(error.reason)) from error
        if not payload.get("ok"):
            failure = payload.get("error", {})
            raise ControlError(failure.get("code", "CONTROL_ERROR"), failure.get("message", "unknown"))
        return payload.get("value")

    def health(self) -> Dict[str, Any]:
        return self.request("GET", "/health")

    def windows(self) -> Dict[str, Any]:
        return self.request("GET", "/windows")

    def snapshot(self, **options: Any) -> Dict[str, Any]:
        return self.request("POST", "/snapshot", options)

    def menus(self) -> Dict[str, Any]:
        return self.request("POST", "/menus", {})

    def invoke(self, selector: Dict[str, Any], settle_ms: int = 0) -> Dict[str, Any]:
        return self.request("POST", "/invoke", {"selector": selector, "settleMs": settle_ms})

    def set_value(self, selector: Dict[str, Any], value: Any, mode: str = "replace") -> Dict[str, Any]:
        return self.request("POST", "/set", {"selector": selector, "value": value, "mode": mode})

    def select(self, selector: Dict[str, Any], **options: Any) -> Dict[str, Any]:
        body = {"selector": selector}
        if "item_index" in options:
            options["index"] = options.pop("item_index")
        body.update(options)
        return self.request("POST", "/select", body)

    def table_set(self, selector: Dict[str, Any], row: int, column: Any, value: Any) -> Dict[str, Any]:
        return self.request("POST", "/table/set",
                            {"selector": selector, "row": row, "column": column, "value": value})

    def table_read(self, selector: Dict[str, Any], max_rows: int = 50) -> Dict[str, Any]:
        return self.request("POST", "/table/read",
                            {"selector": selector, "maxRows": max_rows})

    def key(self, selector: Optional[Dict[str, Any]] = None, key: Optional[str] = None,
            text: Optional[str] = None, modifiers: Optional[List[str]] = None) -> Dict[str, Any]:
        body: Dict[str, Any] = {"key": key, "text": text, "modifiers": modifiers or []}
        if selector:
            body["selector"] = selector
        return self.request("POST", "/key", body)

    def table_edit(self, selector: Dict[str, Any], action: str = "start", row: int = 0,
                   column: Any = 0) -> Dict[str, Any]:
        return self.request("POST", "/table/edit",
                            {"selector": selector, "action": action, "row": row, "column": column})

    def close(self, selector: Dict[str, Any]) -> Dict[str, Any]:
        return self.request("POST", "/close", {"selector": selector})

    def wait_window(self, title: str, state: str = "present", timeout_ms: int = 10000) -> Dict[str, Any]:
        return self.request("POST", "/wait",
                            {"window": title, "state": state, "timeoutMs": timeout_ms})

    def wait_selector(self, selector: Dict[str, Any], state: str = "present",
                      timeout_ms: int = 10000) -> Dict[str, Any]:
        return self.request("POST", "/wait",
                            {"selector": selector, "state": state, "timeoutMs": timeout_ms})

    def screenshot(self, selector: Optional[Dict[str, Any]] = None,
                   path: Optional[str | Path] = None) -> Dict[str, Any]:
        value = self.request("POST", "/screenshot", {"selector": selector or {}})
        if path:
            output = Path(path)
            output.parent.mkdir(parents=True, exist_ok=True)
            output.write_bytes(base64.b64decode(value["pngBase64"]))
        return value

    def trace(self, since: int = 0) -> Dict[str, Any]:
        return self.request("GET", f"/trace?since={int(since)}")

    def routes(self) -> Dict[str, Any]:
        return self.request("GET", "/routes")

    def shutdown(self, exit_process: bool = False) -> Dict[str, Any]:
        return self.request("POST", "/shutdown", {"exit": bool(exit_process)})

    def find(self, selector: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        snapshot = self.snapshot()
        matches = []
        for window in snapshot.get("windows", []):
            if selector.get("window") and not _regex_matches(selector["window"], window.get("text", "")):
                continue
            if selector.get("windowClass") and not _regex_matches(selector["windowClass"], window.get("class", "")):
                continue
            matches.extend(node for node in _iter_nodes(window) if _node_matches(node, selector))
        matches.sort(key=lambda node: 0 if node.get("showing") else 1 if node.get("visible") else 2)
        index = int(selector.get("index", 0))
        return matches[index] if 0 <= index < len(matches) else None

    def wait_text(self, selector: Dict[str, Any], expected: Optional[str] = None,
                  contains: Optional[str] = None, timeout_ms: int = 10000) -> Dict[str, Any]:
        deadline = time.monotonic() + timeout_ms / 1000.0
        last: Optional[Dict[str, Any]] = None
        while time.monotonic() < deadline:
            node = self.find(selector)
            if node is not None:
                last = node
                text = node.get("textValue", node.get("text", ""))
                if expected is not None and text == expected:
                    return node
                if contains is not None and contains in (text or ""):
                    return node
            time.sleep(0.05)
        raise TimeoutError(f"text condition not met for {selector}; last={last}")


def _iter_nodes(node: Dict[str, Any]):
    yield node
    for child in node.get("children", []):
        yield from _iter_nodes(child)


def _regex_matches(pattern: str, value: str) -> bool:
    try:
        return re.search(pattern, value, re.IGNORECASE) is not None
    except re.error:
        return pattern.lower() in value.lower()


def _node_matches(node: Dict[str, Any], selector: Dict[str, Any]) -> bool:
    role = selector.get("role")
    if role and node.get("role") != role:
        return False
    for key in ("path", "name", "class"):
        if key in selector and selector[key] is not None:
            pattern = str(selector[key])
            if key == "path":
                if node.get(key) != pattern:
                    return False
            elif not _regex_matches(pattern, str(node.get(key, ""))):
                return False
    for key, field in (("text", "text"), ("textContains", "text")):
        if selector.get(key) is not None:
            pattern = str(selector[key])
            target = str(node.get(field, ""))
            if key == "textContains":
                if pattern.lower() not in target.lower():
                    return False
            elif not _regex_matches(pattern, target):
                return False
    return True


def selector_from_args(args: argparse.Namespace) -> Dict[str, Any]:
    selector: Dict[str, Any] = {}
    for key in ("window", "window_class", "path", "role", "text", "text_contains", "name", "class", "index"):
        value = getattr(args, key, None)
        if value is not None:
            selector[{"text_contains": "textContains", "window_class": "windowClass"}.get(key, key)] = value
    return selector


def _print(value: Any) -> None:
    print(json.dumps(value, ensure_ascii=False, indent=2, sort_keys=False))


def run_scenario(client: ControlClient, scenario_path: str | Path) -> None:
    document = json.loads(Path(scenario_path).read_text(encoding="utf-8"))
    print(f"scenario: {document.get('name', scenario_path)}")
    for index, step in enumerate(document.get("steps", []), start=1):
        operation = step.get("op")
        if not operation:
            raise ControlError("INVALID_SCENARIO", f"step {index} has no op")
        print(f"[{index}] {operation}")
        if operation == "invoke":
            client.invoke(step["selector"], int(step.get("settleMs", 0)))
        elif operation == "set":
            client.set_value(step["selector"], step.get("value"), step.get("mode", "replace"))
        elif operation == "select":
            options = {k: v for k, v in step.items() if k not in ("op", "selector")}
            client.select(step["selector"], **options)
        elif operation == "table_set":
            client.table_set(step["selector"], step["row"], step["column"], step.get("value"))
        elif operation == "table_edit":
            client.table_edit(step["selector"], step.get("action", "start"),
                              int(step.get("row", 0)), step.get("column", 0))
        elif operation == "table_assert":
            value = client.table_read(step["selector"], int(step.get("maxRows", 50)))
            row = int(step["row"])
            column = step["column"]
            actual = value["rows"][row][column] if isinstance(column, int) else value["rows"][row][
                value["columns"].index(column)]
            if actual != step.get("equals"):
                raise AssertionError(f"table[{row}][{column}] expected {step.get('equals')!r}, got {actual!r}")
        elif operation == "key":
            client.key(step.get("selector"), step.get("key"), step.get("text"), step.get("modifiers"))
        elif operation == "close":
            client.close(step["selector"])
        elif operation == "wait_window":
            result = client.wait_window(step["title"], step.get("state", "present"),
                                        int(step.get("timeoutMs", 10000)))
            if not result.get("matched"):
                raise TimeoutError(f"window not matched: {step['title']}")
        elif operation == "wait_selector":
            result = client.wait_selector(step["selector"], step.get("state", "present"),
                                          int(step.get("timeoutMs", 10000)))
            if not result.get("matched"):
                raise TimeoutError(f"selector not matched: {step['selector']}")
        elif operation == "wait_text":
            client.wait_text(step["selector"], step.get("equals"), step.get("contains"),
                             int(step.get("timeoutMs", 10000)))
        elif operation == "assert_text":
            node = client.find(step["selector"])
            if node is None:
                raise AssertionError(f"selector not found: {step['selector']}")
            actual = node.get("textValue", node.get("text", ""))
            if step.get("equals") is not None and actual != step["equals"]:
                raise AssertionError(f"expected {step['equals']!r}, got {actual!r}")
            if step.get("contains") is not None and step["contains"] not in (actual or ""):
                raise AssertionError(f"expected to contain {step['contains']!r}, got {actual!r}")
        elif operation == "snapshot":
            value = client.snapshot(**{k: v for k, v in step.items() if k != "op"})
            if step.get("path"):
                output = Path(step["path"])
                output.parent.mkdir(parents=True, exist_ok=True)
                output.write_text(json.dumps(value, ensure_ascii=False, indent=2), encoding="utf-8")
            else:
                _print(value)
        elif operation == "screenshot":
            client.screenshot(step.get("selector"), step.get("path", "screenshot.png"))
        elif operation == "sleep":
            time.sleep(float(step.get("ms", 250)) / 1000.0)
        else:
            raise ControlError("INVALID_SCENARIO", f"unknown op at step {index}: {operation}")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--endpoint", default=".xtea-control.json",
                        help="endpoint JSON written by the agent (default: .xtea-control.json)")
    parser.add_argument("--timeout", type=float, default=15.0, help="HTTP timeout in seconds")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("health")
    sub.add_parser("windows")
    sub.add_parser("menus")
    sub.add_parser("routes")

    snapshot = sub.add_parser("snapshot")
    snapshot.add_argument("--window")
    snapshot.add_argument("--table", action="store_true", help="include table rows")
    snapshot.add_argument("--all", action="store_true", help="include invisible components")

    invoke = sub.add_parser("invoke")
    _add_selector_arguments(invoke)
    invoke.add_argument("--settle-ms", type=int, default=0)

    set_parser = sub.add_parser("set")
    _add_selector_arguments(set_parser)
    set_parser.add_argument("--value", required=True)
    set_parser.add_argument("--mode", choices=["replace", "append"], default="replace")

    select = sub.add_parser("select")
    _add_selector_arguments(select)
    select.add_argument("--row", type=int)
    select.add_argument("--column")
    select.add_argument("--item-index", type=int)
    select.add_argument("--title")

    table_read = sub.add_parser("table-read")
    _add_selector_arguments(table_read)
    table_read.add_argument("--max-rows", type=int, default=50)

    table_set = sub.add_parser("table-set")
    _add_selector_arguments(table_set)
    table_set.add_argument("--row", type=int, required=True)
    table_set.add_argument("--column", required=True)
    table_set.add_argument("--value", required=True)

    table_edit = sub.add_parser("table-edit")
    _add_selector_arguments(table_edit)
    table_edit.add_argument("--action", choices=["start", "commit", "cancel"], default="start")
    table_edit.add_argument("--row", type=int, default=0)
    table_edit.add_argument("--column", default="0")

    key = sub.add_parser("key")
    _add_selector_arguments(key)
    key.add_argument("--key")
    key.add_argument("--type-text")
    key.add_argument("--modifiers", nargs="*", default=[])

    close = sub.add_parser("close")
    _add_selector_arguments(close)

    wait_window = sub.add_parser("wait-window")
    wait_window.add_argument("--title", required=True)
    wait_window.add_argument("--state", choices=["present", "absent"], default="present")
    wait_window.add_argument("--timeout-ms", type=int, default=10000)

    wait_selector = sub.add_parser("wait-selector")
    _add_selector_arguments(wait_selector)
    wait_selector.add_argument("--state", choices=["present", "absent"], default="present")
    wait_selector.add_argument("--timeout-ms", type=int, default=10000)

    screenshot = sub.add_parser("screenshot")
    _add_selector_arguments(screenshot)
    screenshot.add_argument("--out", default="screenshot.png")

    trace = sub.add_parser("trace")
    trace.add_argument("--since", type=int, default=0)

    shutdown = sub.add_parser("shutdown")
    shutdown.add_argument("--exit", action="store_true")

    scenario = sub.add_parser("scenario")
    scenario.add_argument("file")

    return parser


def _add_selector_arguments(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--window")
    parser.add_argument("--window-class")
    parser.add_argument("--path")
    parser.add_argument("--role")
    parser.add_argument("--text")
    parser.add_argument("--text-contains")
    parser.add_argument("--name")
    parser.add_argument("--class")
    parser.add_argument("--index", type=int)


def main(argv: Optional[List[str]] = None) -> int:
    args = build_parser().parse_args(argv)
    client = ControlClient.from_endpoint_file(args.endpoint, timeout=args.timeout)
    try:
        if args.command == "health":
            _print(client.health())
        elif args.command == "windows":
            _print(client.windows())
        elif args.command == "menus":
            _print(client.menus())
        elif args.command == "routes":
            _print(client.routes())
        elif args.command == "snapshot":
            options: Dict[str, Any] = {"includeTableRows": args.table, "includeInvisible": args.all}
            if args.window:
                options["window"] = args.window
            _print(client.snapshot(**options))
        elif args.command == "invoke":
            _print(client.invoke(selector_from_args(args), args.settle_ms))
        elif args.command == "set":
            _print(client.set_value(selector_from_args(args), args.value, args.mode))
        elif args.command == "select":
            options = {key: value for key, value in
                       (("row", args.row), ("column", args.column), ("index", args.item_index),
                        ("title", args.title)) if value is not None}
            _print(client.select(selector_from_args(args), **options))
        elif args.command == "table-read":
            _print(client.table_read(selector_from_args(args), args.max_rows))
        elif args.command == "table-set":
            _print(client.table_set(selector_from_args(args), args.row, args.column, args.value))
        elif args.command == "table-edit":
            _print(client.table_edit(selector_from_args(args), args.action, args.row, args.column))
        elif args.command == "key":
            _print(client.key(selector_from_args(args) or None, args.key, args.type_text, args.modifiers))
        elif args.command == "close":
            _print(client.close(selector_from_args(args)))
        elif args.command == "wait-window":
            _print(client.wait_window(args.title, args.state, args.timeout_ms))
        elif args.command == "wait-selector":
            _print(client.wait_selector(selector_from_args(args), args.state, args.timeout_ms))
        elif args.command == "screenshot":
            value = client.screenshot(selector_from_args(args) or None, args.out)
            value.pop("pngBase64", None)
            _print(value)
        elif args.command == "trace":
            _print(client.trace(args.since))
        elif args.command == "shutdown":
            _print(client.shutdown(args.exit))
        elif args.command == "scenario":
            run_scenario(client, args.file)
        else:
            raise ControlError("INVALID_COMMAND", args.command)
    except (ControlError, TimeoutError, AssertionError, KeyError) as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
