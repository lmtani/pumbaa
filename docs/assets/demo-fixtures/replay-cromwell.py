#!/usr/bin/env python3
"""Replay a Cromwell server from recorded fixtures, just enough for `pumbaa
dashboard` and the debug view to render the showcase runs.

Serves the handful of endpoints pumbaa hits (query, per-workflow metadata and
status, engine health) from the JSON files next to this script. Metadata log
paths are stored with a `__FIXROOT__` token that is rewritten at serve time to
the absolute `execroot/` path here, so the debug view reads the committed log
files straight off disk.

    ./replay-cromwell.py [port]        # default port 8010

Used both by the update-demo-gif GitHub workflow and for regenerating the GIF
locally. No third-party dependencies.
"""
import os, re, sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

HERE = os.path.dirname(os.path.abspath(__file__))
FIXROOT = os.path.join(HERE, "execroot")


def read(*parts):
    with open(os.path.join(HERE, *parts)) as f:
        return f.read()


def body_for(path):
    """Return (json_text, 404?) for a request path, ignoring the query string."""
    if path.startswith("/engine/v1/status"):
        return read("status_engine.json")

    m = re.match(r"^/api/workflows/v1/query", path)
    if m:
        return read("query.json")

    m = re.match(r"^/api/workflows/v1/([^/]+)/metadata", path)
    if m:
        return read("metadata", f"{m.group(1)}.json").replace("__FIXROOT__", FIXROOT)

    m = re.match(r"^/api/workflows/v1/([^/]+)/status", path)
    if m:
        return read("status", f"{m.group(1)}.json")

    return None


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        try:
            payload = body_for(self.path)
        except FileNotFoundError:
            payload = None
        if payload is None:
            self.send_error(404, "no fixture for %s" % self.path)
            return
        data = payload.encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *args):
        pass  # keep the recording output clean


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8010
    ThreadingHTTPServer(("127.0.0.1", port), Handler).serve_forever()
