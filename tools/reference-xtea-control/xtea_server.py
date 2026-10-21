#!/usr/bin/env python3
"""Thin, AI-friendly client for the XeadServer HTTP surface.

XeadServer is already a servlet application; unlike the three Swing tools it
does not need an in-process GUI bridge. This client enumerates the command
branches of DBMethod and Service so an agent can exercise them from JSON
instead of hand-written HTTP requests.

It does not start a servlet container. Point --base-url at a deployed XeadServer
(for example http://127.0.0.1:8080/xteaServer).
"""

from __future__ import annotations

import argparse
import base64
import json
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple


DBMETHOD_ROUTES = [
    "GET  /DBMethod (connection status page)",
    "POST /DBMethod METHOD=SELECT ...",
    "POST /DBMethod METHOD=SELECT COUNT(...) ...",
    "POST /DBMethod METHOD=INSERT ...",
    "POST /DBMethod METHOD=UPDATE ...",
    "POST /DBMethod METHOD=DELETE ...",
    "POST /DBMethod METHOD=COMMIT",
    "POST /DBMethod METHOD=ROLLBACK",
    "POST /DBMethod METHOD=CALL ...",
    "POST /DBMethod METHOD=IP",
]

SERVICE_ROUTES = [
    "GET  /Service (session status page)",
    "POST /Service FUNCTION=... [USER=... PASSWORD=...]",
]


class ServerError(RuntimeError):
    def __init__(self, status: int, message: str, body: str = "") -> None:
        super().__init__(f"HTTP {status}: {message}")
        self.status = status
        self.body = body


class ServerClient:
    def __init__(self, base_url: str, user: str = "", password: str = "", timeout: float = 30.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.user = user
        self.password = password
        self.timeout = timeout

    def _post(self, path: str, fields: Dict[str, str]) -> Tuple[int, str, bytes]:
        body = urllib.parse.urlencode(fields).encode("utf-8")
        request = urllib.request.Request(
            self.base_url + path,
            data=body,
            headers={"Content-Type": "application/x-www-form-urlencoded; charset=utf-8"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                return response.status, response.headers.get("Content-Type", ""), response.read()
        except urllib.error.HTTPError as error:
            raw = error.read().decode("utf-8", "replace")
            raise ServerError(error.code, raw[:500], raw) from error
        except urllib.error.URLError as error:
            raise ServerError(0, str(error.reason)) from error

    def _get(self, path: str) -> Tuple[int, str, bytes]:
        request = urllib.request.Request(self.base_url + path, method="GET")
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                return response.status, response.headers.get("Content-Type", ""), response.read()
        except urllib.error.HTTPError as error:
            raw = error.read().decode("utf-8", "replace")
            raise ServerError(error.code, raw[:500], raw) from error
        except urllib.error.URLError as error:
            raise ServerError(0, str(error.reason)) from error

    def status(self) -> Dict[str, Any]:
        code, content_type, body = self._get("/DBMethod")
        return {
            "status": code,
            "contentType": content_type,
            "body": _decode_text(body, content_type),
        }

    def dbmethod(self, method: str, session: str = "", db: str = "") -> Dict[str, Any]:
        fields = {"METHOD": method, "SESSION": session, "DB": db}
        code, content_type, body = self._post("/DBMethod", fields)
        text = _decode_text(body, content_type)
        result: Dict[str, Any] = {
            "status": code,
            "contentType": content_type,
            "method": method,
            "session": session,
            "db": db,
        }
        if _looks_serialized(content_type, body):
            result["serialized"] = True
            result["bytesBase64"] = base64.b64encode(body).decode("ascii")
            result["byteLength"] = len(body)
        else:
            result["body"] = text
        return result

    def query(self, sql: str, session: str = "", db: str = "") -> Dict[str, Any]:
        return self.dbmethod(sql, session=session, db=db)

    def count(self, sql: str, session: str = "", db: str = "") -> Dict[str, Any]:
        return self.dbmethod(sql, session=session, db=db)

    def insert(self, sql: str, session: str = "", db: str = "") -> Dict[str, Any]:
        return self.dbmethod(sql, session=session, db=db)

    def update(self, sql: str, session: str = "", db: str = "") -> Dict[str, Any]:
        return self.dbmethod(sql, session=session, db=db)

    def delete(self, sql: str, session: str = "", db: str = "") -> Dict[str, Any]:
        return self.dbmethod(sql, session=session, db=db)

    def commit(self, session: str, db: str = "") -> Dict[str, Any]:
        return self.dbmethod("COMMIT", session=session, db=db)

    def rollback(self, session: str, db: str = "") -> Dict[str, Any]:
        return self.dbmethod("ROLLBACK", session=session, db=db)

    def call(self, sql: str, session: str = "", db: str = "") -> Dict[str, Any]:
        return self.dbmethod(sql, session=session, db=db)

    def ip(self, session: str = "", db: str = "") -> Dict[str, Any]:
        return self.dbmethod("IP", session=session, db=db)

    def service(self, function: str, params: Optional[Dict[str, Any]] = None,
                user: Optional[str] = None, password: Optional[str] = None) -> Dict[str, Any]:
        fields: Dict[str, str] = {"FUNCTION": function}
        if user is not None:
            fields["USER"] = user
        elif self.user:
            fields["USER"] = self.user
        if password is not None:
            fields["PASSWORD"] = password
        elif self.password:
            fields["PASSWORD"] = self.password
        for key, value in (params or {}).items():
            if isinstance(value, (list, tuple)):
                fields[key] = list(value)  # type: ignore[assignment]
            else:
                fields[key] = str(value)
        body = urllib.parse.urlencode(fields, doseq=True).encode("utf-8")
        request = urllib.request.Request(
            self.base_url + "/Service",
            data=body,
            headers={"Content-Type": "application/x-www-form-urlencoded; charset=utf-8"},
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                content_type = response.headers.get("Content-Type", "")
                raw = response.read()
                return {
                    "status": response.status,
                    "contentType": content_type,
                    "body": _decode_text(raw, content_type),
                }
        except urllib.error.HTTPError as error:
            raw = error.read().decode("utf-8", "replace")
            raise ServerError(error.code, raw[:500], raw) from error
        except urllib.error.URLError as error:
            raise ServerError(0, str(error.reason)) from error

    def routes(self) -> Dict[str, Any]:
        return {"dbmethod": DBMETHOD_ROUTES, "service": SERVICE_ROUTES}


def _looks_serialized(content_type: str, body: bytes) -> bool:
    return body[:2] == b"\xac\xed" or "application/octet-stream" in content_type


def _decode_text(body: bytes, content_type: str) -> str:
    charset = "utf-8"
    for part in content_type.split(";"):
        part = part.strip()
        if part.lower().startswith("charset="):
            charset = part.split("=", 1)[1].strip().strip('"')
            break
    try:
        return body.decode(charset, "replace")
    except LookupError:
        return body.decode("utf-8", "replace")


def _print(value: Any) -> None:
    print(json.dumps(value, ensure_ascii=False, indent=2))


def main(argv: Optional[List[str]] = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", required=True, help="e.g. http://127.0.0.1:8080/xteaServer")
    parser.add_argument("--user", default="")
    parser.add_argument("--password", default="")
    parser.add_argument("--timeout", type=float, default=30.0)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("routes")
    sub.add_parser("status")
    for name in ("dbmethod", "query", "count", "insert", "update", "delete", "call", "ip"):
        command = sub.add_parser(name)
        if name not in ("ip",):
            command.add_argument("--method", default="" if name == "dbmethod" else None)
        command.add_argument("--session", default="")
        command.add_argument("--db", default="")
    service = sub.add_parser("service")
    service.add_argument("--function", required=True)
    service.add_argument("--param", action="append", default=[], metavar="KEY=VALUE")
    args = parser.parse_args(argv)
    client = ServerClient(args.base_url, args.user, args.password, args.timeout)
    try:
        if args.command == "routes":
            _print(client.routes())
        elif args.command == "status":
            _print(client.status())
        elif args.command == "service":
            params: Dict[str, Any] = {}
            for item in args.param:
                key, _, value = item.partition("=")
                params[key] = value
            _print(client.service(args.function, params))
        elif args.command == "ip":
            _print(client.ip(args.session, args.db))
        else:
            method = args.method
            if args.command != "dbmethod":
                method = {"query": "SELECT ", "count": "SELECT COUNT(*) ", "insert": "INSERT ",
                          "update": "UPDATE ", "delete": "DELETE ", "call": "CALL "}[args.command] + (method or "")
            if not method:
                raise SystemExit("--method is required for dbmethod")
            _print(client.dbmethod(method, args.session, args.db))
    except ServerError as error:
        print(f"error: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
