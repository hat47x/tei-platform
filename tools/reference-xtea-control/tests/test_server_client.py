#!/usr/bin/env python3
"""Contract test for the XeadServer HTTP client using a local mock servlet."""

from __future__ import annotations

import json
import sys
import threading
import unittest
import urllib.parse
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from xtea_server import ServerClient  # noqa: E402


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_: object) -> None:
        pass

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/DBMethod":
            body = b"<html>status</html>"
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=UTF-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        else:
            self.send_error(404)

    def do_POST(self) -> None:  # noqa: N802
        length = int(self.headers.get("Content-Length", "0"))
        fields = urllib.parse.parse_qs(self.rfile.read(length).decode("utf-8"))
        if self.path == "/DBMethod":
            method = fields.get("METHOD", [""])[0]
            if method.startswith("SELECT COUNT("):
                body = b"2"
                content_type = "text/html; charset=UTF-8"
            elif method.startswith("SELECT "):
                body = b"\xac\xed\x00\x05mock"
                content_type = "application/octet-stream"
            elif method == "COMMIT":
                body = b"The transaction was commited."
                content_type = "text/html; charset=UTF-8"
            elif method == "ROLLBACK":
                body = b"The transaction was rollbacked."
                content_type = "text/html; charset=UTF-8"
            elif method in ("IP",) or method.startswith(("INSERT ", "UPDATE ", "DELETE ", "CALL ")):
                body = b"1"
                content_type = "text/html; charset=UTF-8"
            else:
                body = b"0"
                content_type = "text/html; charset=UTF-8"
        elif self.path == "/Service":
            body = json.dumps({"function": fields.get("FUNCTION", [""])[0],
                               "params": {k: v for k, v in fields.items() if k != "FUNCTION"}},
                              ensure_ascii=False).encode("utf-8")
            content_type = "application/json;charset=UTF-8"
        else:
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


class ServerClientTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.httpd = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        cls.thread = threading.Thread(target=cls.httpd.serve_forever, daemon=True)
        cls.thread.start()
        cls.client = ServerClient(f"http://127.0.0.1:{cls.httpd.server_port}")

    @classmethod
    def tearDownClass(cls) -> None:
        cls.httpd.shutdown()
        cls.httpd.server_close()

    def test_status(self) -> None:
        status = self.client.status()
        self.assertEqual(status["status"], 200)
        self.assertIn("status", status["body"])

    def test_db_commands(self) -> None:
        self.assertEqual(self.client.count("SELECT COUNT(*) FROM T")["body"], "2")
        query = self.client.query("SELECT * FROM T")
        self.assertTrue(query["serialized"])
        self.assertGreater(query["byteLength"], 0)
        self.assertEqual(self.client.insert("INSERT INTO T VALUES (1)")["body"], "1")
        self.assertEqual(self.client.update("UPDATE T SET A=1")["body"], "1")
        self.assertEqual(self.client.delete("DELETE FROM T")["body"], "1")
        self.assertEqual(self.client.call("CALL P()")["body"], "1")
        self.assertEqual(self.client.ip()["body"], "1")
        self.assertIn("commited", self.client.commit("S1")["body"])
        self.assertIn("rollbacked", self.client.rollback("S1")["body"])

    def test_service(self) -> None:
        result = self.client.service("AF200J", {"REQUEST_TYPE": "ListSections"})
        payload = json.loads(result["body"])
        self.assertEqual(payload["function"], "AF200J")
        self.assertEqual(payload["params"]["REQUEST_TYPE"], ["ListSections"])

    def test_routes_cover_all_branches(self) -> None:
        routes = self.client.routes()
        joined = "\n".join(routes["dbmethod"] + routes["service"])
        for branch in ("SELECT", "COUNT", "INSERT", "UPDATE", "DELETE", "COMMIT", "ROLLBACK", "CALL", "IP"):
            self.assertIn(branch, joined)


if __name__ == "__main__":
    unittest.main(verbosity=2)
