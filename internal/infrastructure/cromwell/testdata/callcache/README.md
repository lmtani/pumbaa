# Call-caching metadata fixtures

Four metadata payloads captured from a real Cromwell 91 server, kept because the
server they came from used an in-memory database: once it stopped, those runs
were gone for good.

They exist so the call-cache analysis can be exercised — including the parts
that depend on how Cromwell actually behaves — without a live server.

| File | What it pins |
|---|---|
| `run1_reference.json` | A baseline run: every call a miss, with full hashes |
| `run2_all_hits.json` | The same submission again: both calls memoised, carrying `Cache Hit: <id>:<fqn>:<shard>` pointers |
| `run3_docker_changed.json` | Only the upstream task's image changed: one root cause, one cascade |
| `run5_outputs_deleted.json` | Identical fingerprints and still a miss, because the cached outputs had been deleted |

## How they were produced

A disposable Cromwell (`example/showcase/start-cromwell.sh`, Local backend with
Docker) ran the `vcf_index_stats` workflow — a two-task chain where the second
consumes the first's output.

1. Submit it. → `run1_reference.json`
2. Submit the identical thing again. → `run2_all_hits.json`
3. Submit with only the first task's container image changed. → `run3_docker_changed.json`
4. Delete every copy of the first task's output, then submit the original again.
   → `run5_outputs_deleted.json`

## What they established

These are the observations the implementation rests on, each of which was a
guess before being measured:

- **Hashes are returned by the plain `/metadata` endpoint**, with no `includeKey`
  needed, as a **nested tree** rather than the flat `"input: File x"` map that
  Cromwell's logs suggest.
- **The command-template digest does not move when input values do.** In
  `run3`, the second task received a different input path and its command digest
  was unchanged — which is what makes a command change detectable from WDL text
  alone, and therefore detectable for tasks inside imports, whose source a run
  never records.
- **File inputs are hashed by content.** A file at a new path with the same bytes
  is the same input; the same path with new bytes is not.
- **A miss with an identical fingerprint means the cached copy was unusable**,
  not that something changed — `run5` is that case, and it is the only way to
  tell "you changed something" from "your cache was destroyed".
- **`hitFailures` never appeared**, not even in `run5`, where a candidate existed
  and could not be copied from. It cannot be relied on.

Reproduce by re-running the four steps above; the workflow and its inputs are in
`example/showcase/`.
