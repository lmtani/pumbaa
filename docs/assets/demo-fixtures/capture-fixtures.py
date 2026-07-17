#!/usr/bin/env python3
"""Record replay fixtures for the demo GIF from a live showcase Cromwell.

Run only when the showcase workflows change. Bring up the local showcase
Cromwell, submit the five runs (example/showcase/run.sh), then:

    CROMWELL_HOST=http://localhost:8010 python3 docs/assets/demo-fixtures/capture-fixtures.py

Workflows are matched by name (not by UUID), so a fresh set of runs is picked up
automatically. Each is given a fixed, well-spaced submission timestamp so the
dashboard's newest-first sort — and therefore the tape's keyboard navigation —
stays deterministic. See README.md for how the fixtures are used.
"""
import json, os, shutil, urllib.request
from datetime import datetime

BASE = os.environ.get("CROMWELL_HOST", "http://localhost:8010").rstrip("/")
HERE = os.path.dirname(os.path.abspath(__file__))
# Anything under this prefix (task logs, call roots) is tokenised so the stub can
# relocate it to the checkout at serve time. Override if your executions live
# elsewhere.
SRC_PREFIX = os.environ.get(
    "PUMBAA_SHOWCASE_DIR",
    os.path.abspath(os.path.join(HERE, "..", "..", "..", "example", "showcase")),
)
TOKEN = "__FIXROOT__"

# name -> submission timestamp, oldest first. The dashboard sorts newest first,
# so the first entry here becomes the LAST row (which the tape navigates to).
ORDER = [
    ("FastqToUnmappedBam", "2026-07-17T20:10:00.000Z"),
    ("BamToCram",          "2026-07-17T20:11:00.000Z"),
    ("CramToBam",          "2026-07-17T20:12:00.000Z"),
    ("BamQualityCheck",    "2026-07-17T20:13:00.000Z"),
    ("VcfIndexAndStats",   "2026-07-17T20:14:00.000Z"),
]


def get(path):
    with urllib.request.urlopen(BASE + path) as r:
        return r.read().decode()


def parse_ts(ts):
    return datetime.fromisoformat(ts.replace("Z", "+00:00"))


def fmt_ts(dt):
    return dt.strftime("%Y-%m-%dT%H:%M:%S.") + f"{dt.microsecond // 1000:03d}Z"


def retokenize(obj):
    if isinstance(obj, str):
        return obj.replace(SRC_PREFIX, TOKEN)
    if isinstance(obj, list):
        return [retokenize(x) for x in obj]
    if isinstance(obj, dict):
        return {k: retokenize(v) for k, v in obj.items()}
    return obj


def copy_logs(meta):
    for atts in meta.get("calls", {}).values():
        for a in atts:
            for k in ("stderr", "stdout"):
                p = a.get(k, "")
                if p and p.startswith(SRC_PREFIX) and os.path.exists(p):
                    dst = os.path.join(HERE, "execroot", p[len(SRC_PREFIX):].lstrip("/"))
                    os.makedirs(os.path.dirname(dst), exist_ok=True)
                    shutil.copyfile(p, dst)


# Resolve the latest workflow id for each expected name.
q = json.loads(get("/api/workflows/v1/query?includeSubworkflows=false"))
latest = {}
for w in q.get("results", []):
    latest.setdefault(w["name"], w["id"])  # query is newest-first
missing = [n for n, _ in ORDER if n not in latest]
if missing:
    raise SystemExit(f"missing runs for: {', '.join(missing)} — submit them first")

for sub in ("metadata", "status"):
    shutil.rmtree(os.path.join(HERE, sub), ignore_errors=True)
    os.makedirs(os.path.join(HERE, sub))
shutil.rmtree(os.path.join(HERE, "execroot"), ignore_errors=True)

with open(os.path.join(HERE, "status_engine.json"), "w") as f:
    f.write(get("/engine/v1/status"))

results = []
for name, sub in reversed(ORDER):  # newest first in the query response
    wid = latest[name]
    meta = json.loads(get(f"/api/workflows/v1/{wid}/metadata"))
    copy_logs(meta)
    with open(os.path.join(HERE, "metadata", f"{wid}.json"), "w") as f:
        json.dump(retokenize(meta), f, indent=2)
    with open(os.path.join(HERE, "status", f"{wid}.json"), "w") as f:
        f.write(get(f"/api/workflows/v1/{wid}/status"))
    # Keep the real run duration, but anchored at the clean submission time, so
    # the dashboard's end - start stays positive.
    end = fmt_ts(parse_ts(sub) + (parse_ts(meta["end"]) - parse_ts(meta["start"])))
    results.append({"id": wid, "name": name, "status": "Succeeded",
                    "submission": sub, "start": sub, "end": end})

with open(os.path.join(HERE, "query.json"), "w") as f:
    json.dump({"results": results, "totalResultsCount": len(results)}, f, indent=2)

print(f"recorded {len(results)} workflows into {HERE}")
