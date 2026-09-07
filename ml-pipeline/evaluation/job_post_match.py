#!/usr/bin/env python3
"""Post-match evaluation job - delegates to backend API (same logic as store.go)."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

from dotenv import load_dotenv

load_dotenv()


def run_via_backend(base_url: str) -> dict:
    url = f"{base_url.rstrip('/')}/api/v1/evaluation/run"
    req = urllib.request.Request(
        url,
        method="POST",
        headers={"Content-Type": "application/json"},
        data=b"{}",
    )
    with urllib.request.urlopen(req, timeout=120) as resp:
        return json.loads(resp.read().decode())


def main() -> int:
    backend_url = os.getenv("BACKEND_URL", "http://localhost:8082")
    try:
        result = run_via_backend(backend_url)
    except urllib.error.HTTPError as exc:
        body = exc.read().decode()
        print(f"Backend evaluation HTTP {exc.code}: {body}", file=sys.stderr)
        return 1
    except urllib.error.URLError as exc:
        print(f"Backend unreachable at {backend_url}: {exc.reason}", file=sys.stderr)
        return 1

    evaluated = result.get("evaluated", 0)
    message = result.get("message", "Evaluation terminee")
    print(f"{message} ({evaluated} outcomes)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
