"""Tests for post-match evaluation job."""

from __future__ import annotations

import json
from http.server import BaseHTTPRequestHandler, HTTPServer
from threading import Thread
from unittest.mock import patch

from evaluation import job_post_match


class EvalHandler(BaseHTTPRequestHandler):
    def do_POST(self):  # noqa: N802
        if self.path == "/api/v1/evaluation/run":
            body = json.dumps({"evaluated": 42, "message": "ok"}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(body)
            return
        self.send_response(404)
        self.end_headers()

    def log_message(self, format, *args):  # noqa: A003
        return


def test_job_post_match_calls_backend():
    server = HTTPServer(("127.0.0.1", 0), EvalHandler)
    port = server.server_address[1]
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        with patch.dict("os.environ", {"BACKEND_URL": f"http://127.0.0.1:{port}"}):
            assert job_post_match.main() == 0
    finally:
        server.shutdown()
