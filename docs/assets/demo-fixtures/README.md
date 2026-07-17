# Demo GIF replay fixtures

These files let the README's `demo.gif` be regenerated without a live Cromwell.
`replay-cromwell.py` is a tiny stub server that answers the handful of endpoints
`pumbaa dashboard` and the debug view hit, from recorded responses:

```
query.json            # GET /api/workflows/v1/query          (the 5 showcase runs)
status_engine.json    # GET /engine/v1/status                (health → "Healthy")
metadata/<id>.json    # GET /api/workflows/v1/<id>/metadata
status/<id>.json      # GET /api/workflows/v1/<id>/status
execroot/…/stderr     # the real task logs the debug view reads off disk
```

Metadata log paths are stored with a `__FIXROOT__` token that the stub rewrites
to the absolute `execroot/` path at serve time, so the debug view reads the
committed `stderr`/`stdout` files directly (that is how the Picard log shows up
in the GIF). Submission timestamps in `query.json` are spaced one minute apart so
the dashboard's newest-first sort is deterministic — which keeps the tape's
keyboard navigation stable (`FastqToUnmappedBam` is always the last row).

## Regenerate the GIF locally

```bash
# from the repo root, with pumbaa on PATH and vhs/ttyd/ffmpeg installed
python3 docs/assets/demo-fixtures/replay-cromwell.py 8010 &
vhs docs/assets/demo.tape
```

This is exactly what the **Update demo GIF** GitHub workflow
(`.github/workflows/update-demo-gif.yml`, manual trigger) does, before opening a
PR with the result.

## Re-record the fixtures

Only needed when the showcase workflows themselves change. Bring up the local
showcase Cromwell (`example/showcase/`), submit the runs, then:

```bash
CROMWELL_HOST=http://localhost:8010 python3 docs/assets/demo-fixtures/capture-fixtures.py
```

See `capture-fixtures.py` for the workflow IDs and timestamps it records.
