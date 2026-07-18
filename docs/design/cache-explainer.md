# Design: call cache — forecast and explain

Status: forecast (pre-submission) implemented; explain (post-mortem) designed
but not built. Audience: contributors.

Two moments, one engine. The **forecast** answers "what will actually run if I
submit this?" before any money is spent — a biopsy. The **explain** answers
"why did this rerun?" after the fact — an autopsy. They share the fingerprint
comparison, the root-cause/cascade classification and the backend gating; they
differ only in where the two sides being compared come from.

## Problem

"Why did this task not hit the cache?" is the most common and worst-served
question in day-to-day Cromwell use. The answer exists in the metadata, but
in a form nobody reads: a nested tree of opaque MD5 hashes per call, which is
meaningless in isolation — it only becomes an answer when diffed against
another run.

Today Pumbaa consumes call caching only to *recover provenance*: `CacheHit`,
`CacheResult`, and the `cacheResolver` that follows `Cache Hit: <wf>:<fqn>:<shard>`
pointers so the diff can compare real metrics. It never asks the inverse
question — why a call **missed**.

The user-facing symptom is the worst part: a pipeline that should have been
free reruns 40 tasks, and the user has no way to tell whether that is one
real change or forty.

## What Cromwell actually gives us

Per call, under `callCaching`:

| Field | Use |
|---|---|
| `hit` (bool), `result` (string) | already mapped |
| `effectiveCallCachingMode` | `ReadAndWriteCache` / `WriteCache` / `CallCachingOff` |
| `allowResultReuse` (bool) | whether this call's results can be reused later |
| `hashes` (tree) | the per-call hash fingerprint — the heart of this feature |
| `hitFailures` (list) | documented, but **never observed** — see below |

**Validated empirically** against Cromwell 91 (`example/showcase`, Local +
Docker backend) — `hashes` *is* returned by the default `/metadata` endpoint,
with no `includeKey` needed, alongside `allowResultReuse` and
`effectiveCallCachingMode`.

The shape is a **nested tree**, not the flat `"input: File x"` map one might
expect from Cromwell's log output:

```json
"hashes": {
  "backend name":     "509820290D57F333403F490DDE7316F4",
  "command template": "94A353F002A219A742B02214A920582A",
  "input count":      "ECCBC87E4B5CE2FE28308FD9F2A7BAF3",
  "output count":     "C81E728D9D4C2F636F067F89CC14862C",
  "runtime attribute": {
    "docker":               "5CF5ADB49E5191F0A1DA579CA8C9FB0E",
    "continueOnReturnCode": "CFCD208495D565EF66E7DFF9F98764DA",
    "failOnStderr":         "68934A3E9455FA72420237EB05902327"
  },
  "input": {
    "String output_basename": "7A6BE2B0057A7F4DE29BC1AC77497B5F",
    "String docker":          "6E5CA1E819BA223A974C66FF74978588",
    "File input_vcf":         "41a44e64f3c014c39dfc5b7b09fbf75c"
  },
  "output expression": {
    "File out_vcf":       "3BE037CC96CE1D29EC506AF62BBDDDAF",
    "File out_vcf_index": "6F8614DCFF6FA6B9673D60891E648AAA"
  }
}
```

Two consequences:

- The domain must **flatten** this tree to `"runtime attribute: docker"` style
  keys before diffing. `diff.go` already has `flattenJSON`/`flattenValue`
  doing exactly this for inputs/options — reuse the pattern.
- Note the casing: metadata hashes are uppercase MD5, while **file** hashes
  (`File input_vcf`) are lowercase — they are the file's *content* MD5 under
  the `"file"` hashing strategy. So a differing file hash distinguishes
  "same path, new content" from "different path", which is a strictly better
  answer than either alone.

`hitFailures` was **absent** on a first-ever run: it appears only when a
candidate was actually found and then rejected. So it cannot be relied on as
the primary signal — it is the bonus diagnostic A, not the backbone.

## The two diagnostics

These are genuinely different features and it matters not to conflate them.

**A. Self-contained** — needs only the run itself:

- Caching was off or write-only (`effectiveCallCachingMode`). The dumbest and
  most frequent cause; needs no comparison at all.
- `allowResultReuse: false` on the source side.

`hitFailures` was originally expected to carry a third case here. **It does
not** — see "The unusable-candidate case" below.

**B. Referenced** — needs a run to compare against:

Cromwell does not record "I considered candidate X and its command template
differed." To explain a plain `Cache Miss` you must diff the current call's
hash map against the hash map of the run the user *expected* it to hit. The
differing keys are the answer.

Enrichment matters here: a differing `input: File ref_fasta` tells you which
input changed but the hashes are opaque MD5s. `call.Inputs["ref_fasta"]` is
already in the domain on both sides, so the report can show the two actual
paths. That is the difference between "an input changed" and "the reference
moved from `gs://.../hg38.fa` to `gs://.../hg38_v2.fa`".

## The lede: root cause vs cascade

This is the feature's real value and the reason it should not be a flat
per-task table.

One task missing forces every downstream task to miss, because its inputs
changed. A naive report says "40 misses" and buries the single fact the user
needs. The explainer must classify:

- **Root cause** — the call differs in something *not* explained by an
  upstream miss: `command template`, `runtime attribute: docker`, a backend
  change, or an input file that is a workflow-level input rather than another
  task's output.
- **Cascade** — every differing key is an `input: File X` whose value traces
  to another call *in the same run that itself missed*.

The producer lookup is pure and needs no new data: the differing input's value
is a path like `…/call-AlignReads/shard-0/out.bam`, matched against other
calls' `Outputs` values or by `CallRoot` prefix. Both are already on `Call`.

**Validated on a real three-run experiment** (`vcf_index_stats`, the chain
`IndexVcf → StatsVcf`, Cromwell 91). Run 1 = reference; run 2 identical
(both calls hit, pointing at run 1); run 3 with only `IndexVcf.docker`
overridden from bcftools 1.11 to 1.12. Flattening and diffing run 3 against
run 1 gives, with zero ambiguity:

```
IndexVcf: 2 of 12 keys differ
   input: String docker          6E5CA1E8… → CE9D8997…
   runtime attribute: docker     5CF5ADB4… → AC2FDB31…
StatsVcf: 1 of 12 keys differ
   input: File input_vcf         048ac8ff… → 650d94a8…
```

`IndexVcf` differs only in docker → root cause. `StatsVcf` differs only in an
input file whose value is *exactly* `IndexVcf.outputs["out_vcf"]` (verified:
plain string equality on the path) → cascade. The classification falls out of
the data with no heuristics.

Two refinements the experiment surfaced:

- **Collapse duplicate signals.** A docker change shows up twice when the WDL
  passes docker as a task input (`input: String docker` *and* `runtime
  attribute: docker`). The report must present that as one finding, not two.
- **Content hashing propagates the cascade.** Under the `file` hashing
  strategy the input hash is the file's *content* MD5, so the cascade fires
  even when a rerun produces a logically equivalent file — as it did here
  (bcftools 1.11 vs 1.12 wrote a byte-different `.vcf.gz`). This is why the
  cascade must be reported as "downstream of X" and not as an independent
  cause: the user can do nothing about it except fix X.

Target headline:

```
40 of 52 calls missed the cache.
Root cause: AlignReads — docker changed
  gcr.io/…/bwa:0.7.17  →  gcr.io/…/bwa:0.7.18
The other 39 misses are downstream of it.
```

This mirrors `CalculateFailureSummary`, which already collapses failures to
deduplicated root causes. Same philosophy, applied to cache.

## The unusable-candidate case (and the death of `hitFailures`)

A fourth run tested the canonical "cached outputs were deleted from the
bucket" scenario: every physical copy of `IndexVcf`'s output file was removed,
then an identical workflow resubmitted.

Cromwell reported a plain `Cache Miss` and **`hitFailures` was absent from the
metadata entirely** — the `callCaching` block held only the same five keys
(`allowResultReuse`, `effectiveCallCachingMode`, `hashes`, `hit`, `result`).
So `hitFailures` cannot be the backbone of diagnostic A, and it cannot be used
to infer the reference run either.

But the same run produced a **better signal by elimination**:

```
IndexVcf: hit=false, differing hash keys vs reference = 0
StatsVcf: hit=false, differing keys = input: File input_vcf, input: File input_vcf_index
```

`IndexVcf`'s fingerprint was byte-identical to the reference and it *still*
missed. Nothing changed — so the only possible cause is that the candidate
existed and was unusable (outputs deleted, entry invalidated). That yields a
clean three-way verdict, which is the explainer's core output:

| Evidence | Verdict | What the user should do |
|---|---|---|
| Hashes differ | Something changed: `<key>` | Fix or accept the change |
| Hashes identical, still missed | The cached copy was unusable | Outputs were deleted / entry invalidated |
| `effectiveCallCachingMode` off/write-only | Caching was not enabled for reads | Fix the config |

Distinguishing row 1 from row 2 is the single most confusing situation in
day-to-day Cromwell use, and no existing tool answers it. This is a stronger
feature than the original design assumed.

## Choosing the reference run automatically

`hitFailures` is out (above), so the fallback becomes the primary: **the
previous run of the same workflow name**, via the existing query port — the
same heuristic a human uses. `--against <id>` stays as the explicit override.

If no reference can be found, degrade to diagnostic A and say so rather than
failing. Note that row 2 of the verdict table *requires* a reference: without
one, "identical hashes" is unknowable.

## Predicting before submitting

The forecast cannot read hashes for a run that has not happened. It does not
try to reimplement Cromwell's hashing either — that would be fragile and
version-bound. Instead it compares *values* along the axes Cromwell
fingerprints, which is sound because of two properties the experiments proved:

- **The command template is hashed pre-substitution**, so the hash depends only
  on the WDL text. Better still, the hash is *reproducible*: it is the MD5 of
  the template with each line trimmed, rejoined with newlines, and `~{expr}`
  rewritten to `${expr}`. That formula matches every task in the captured
  fixtures, so a command change is detectable by computing the hash locally and
  comparing it against the reference's fingerprint — **without needing the
  reference run's WDL at all**. That matters more than it first appears: a
  run's metadata carries only the top-level workflow source, never its imported
  files, so text comparison could never have covered tasks inside a
  subworkflow.

  The normalisation is not a documented contract, so the forecast calibrates
  before trusting it: if no call's computed hash reproduces its recorded one,
  the axis is dropped with a warning rather than reporting every task as
  changed.
- **File inputs are hashed by content**, and the reference run's metadata
  records those hashes. So the candidate file's MD5 — from GCS object metadata
  without downloading, or computed locally — is directly comparable.

Comparison axes, per call: command template text, docker image, and every
input's value (files by content hash, scalars by value). A call whose own
fingerprint is unchanged but whose upstream will rerun is a cascade.

### Subworkflows and imports

**Cromwell caches leaf tasks, not subworkflow calls.** A forecast that treated
`call sub.AlignSample` as one unit would be answering a question the cache
never asks. So the graph is *flattened*: a resolved subworkflow contributes its
leaves under their call path (`AlignSample.Align`), and the subworkflow call
itself disappears from the graph.

Flattening requires following values across the boundary in both directions:

- **Inward**, a leaf input bound to a subworkflow input is rewritten to
  whatever the parent passed — the top-level input, a literal, or another
  parent call's output. An input the parent does not pass falls back to the
  subworkflow's own default, and failing that is scoped to the call path, which
  is where Cromwell expects it in the inputs JSON (`Top.Sub.name`).
- **Outward**, a consumer of `Sub.result` must depend on the *leaf that
  produces that output*, resolved through the subworkflow's output
  declarations. Depending on all leaves would be conservative but wrong in
  practice: an unrelated rerun inside the subworkflow would cascade to
  consumers that never read its result.

On the reference side this needs metadata fetched with
`expandSubWorkflows=true`, since a subworkflow's calls live under the parent
call's own metadata rather than at the top level. Lookup walks the path one
segment at a time.

**An imported task is not a subworkflow.** The first iteration classified any
call whose target was not a task in the current document as a subworkflow,
which silently withheld a verdict for `import "tasks.wdl"` — the most common
pattern in production pipelines, and enough to make the forecast useless on
them. Resolution now distinguishes the two by parsing the imported document and
asking whether the target is one of its tasks or its workflow.

Sources come from `--dependencies` when given, otherwise from WDL files beside
the workflow, which is how a checkout resolves. Anything still unresolved keeps
its call opaque and undetermined, with a warning naming it — never assumed
unchanged, because an invisible task body hides command changes.

Two traps found while validating against the live server, both about
call-scoped overrides in the inputs JSON:

- Cromwell accepts `Workflow.Call.docker` overrides that **never appear as call
  bindings in the WDL**. Iterating only the WDL's explicit bindings silently
  predicted "reuse" for a submission that changed a task's image — the very
  manoeuvre the original experiment used to force a miss. The input list is
  therefore taken from the *reference call's* recorded inputs, which is
  authoritative, with WDL bindings only supplying provenance.
- That form works **only for calls of the top-level workflow**. Cromwell
  rejects `Top.Sub.Leaf.docker` outright ("Unexpected input provided"), so the
  override is not consulted for nested calls; their values come from the
  subworkflow's own WDL. Discovered by a submission that failed input
  processing, which is a better outcome than a forecast quietly disagreeing
  with a server that would never have accepted the run.

## Layering

```
pkg/wdl/callgraph.go            BuildCallGraph: call → upstream calls, plus
                                where each input's value comes from
                                (workflow input / literal / another call).
pkg/wdl/taskspec.go             TaskSpecs: command template, runtime, input
                                defaults — the statically comparable parts.
domain/workflow/                CallFingerprint + FlattenHashes + Compare;
  cachefingerprint.go           change categories (docker collapses two keys).
domain/workflow/                BackendKind gating, PredictedFate, and
  cachepredict.go               PredictCacheReuse: cascade propagation with
                                unknown poisoning downstream. Pure, no IO.
infrastructure/cromwell/        callCachingInfo gains hashes/mode/
                                allowResultReuse; the mapper flattens the tree
                                onto Call.Fingerprint.
infrastructure/storage/         GetContentHash on both backends: GCS reads
                                MD5 from object metadata (no download), local
                                streams the file.
application/workflow/           CacheForecastUseCase: resolve the reference,
  cacheforecast.go              compare, propagate.
interfaces/cli/handler/         cache-forecast command, table + --json.
  cacheforecast.go
```

The domain must treat *missing* hashes as "cannot determine" and never as an
error — the same rule guided-submit uses for a WDL the parser cannot read.
Cromwell is the authority; both the forecast and the explain are advisory.

## Interface

Implemented:

```
pumbaa workflow cache-forecast -w <wdl> [-i <inputs>] [--against <id>] [--json]
```

The reference defaults to the latest successful run of the same workflow, so
the common case needs no extra argument.

Still to build — the post-mortem side:

```
pumbaa workflow cache <id> [--against <id>] [--task NAME] [--json]
```

TUI: a natural follow-up on the debug screen next to the `$` cost modal, out
of scope for this iteration.

Agent: read-only, so it fits the current tool policy with no confirmation UX,
and it serves the agent's main job directly — "why did my pipeline rerun?"

## What the live validation measured

Two runs of a parent workflow calling a subworkflow (`example/showcase`,
`cohort_qc.wdl` → `qc_sub.wdl`), forecast first and then actually submitted.

An identical resubmission was predicted as full reuse, and all three leaf calls
hit. Bumping `IndexVcf`'s docker inside the subworkflow produced:

| Call | Predicted | Actual |
|---|---|---|
| `VcfQc.IndexVcf` | will run again (docker) | missed |
| `VcfQc.StatsVcf` | may run again, downstream | **hit** |
| `Summarize` | may run again, downstream | **hit** |

The root cause was identified exactly, including that it lives *inside* a
subworkflow and that the parent's `Summarize` traces back to it rather than to
the nearest hop. Both downstream calls still hit, because bcftools 1.12 wrote a
byte-identical output to 1.11.

That is the irreducible limit, not a defect — and it is why the summary counts
downstream calls separately ("1 will run again and 2 more may") instead of
folding them into the rerun total. An earlier wording claimed "none of the 3
would be reused" for a run where two thirds of it was in fact free.

Import handling was measured against a real production pipeline (Illumina
TSO500, 24 WDL files): with the imports zip, 23 calls resolve and **none** are
undetermined, with both subworkflows flattened into their leaves and scattered
calls flagged. Without it, 20 of 21 calls are undetermined — which is what the
first iteration effectively did to every import-heavy pipeline.

## Supported backends

Prediction is claimed only for **local** and **GCP** (`PAPIv2*`, `GCPBATCH`,
`JES`). `ClassifyBackend` maps anything else to `BackendUnsupported`, and the
forecast then reports every call as undetermined with a warning naming the
backend, rather than guessing.

The reason is not effort but correctness: the file hashing strategy is a
backend *and configuration* property (`hashing-strategy: "file" | "path" |
"path+modtime"`). Comparing content hashes under a path-based strategy would
produce confident nonsense. As a second guard, a reference hash that is not
32 hex characters is not an MD5 — the comparison is refused and the call
reported as undetermined instead.

## Fixtures

The experiment's four metadata payloads are committed under
`internal/infrastructure/cromwell/testdata/callcache/` (52 KB total), because
the showcase Cromwell uses an in-memory database and these runs cannot be
reproduced once the server stops:

| File | What it exercises |
|---|---|
| `run1_reference.json` | Baseline; every call a miss with full hashes |
| `run2_all_hits.json` | Both calls hit, `Cache Hit: <id>:<fqn>:<shard>` pointers |
| `run3_docker_changed.json` | Root cause (docker) + cascade in one run |
| `run5_outputs_deleted.json` | Identical hashes yet a miss — the unusable-candidate verdict |

Together they cover every branch of the verdict table without a live server.
Reproduce with `example/showcase/start-cromwell.sh` and the `vcf_index_stats`
workflow.

## Risks

1. **Payload size.** Hashes inflate metadata on large runs (thousands of
   shards × ~12 keys). Measured here at ~11 KB for a 2-call run, of which the
   hashes are a meaningful share. May need `includeKey`/`excludeKey` tuning so
   only the explainer pays that cost, not every metadata fetch.
2. **Hashes absent on old runs**, on servers configured to exclude them, or
   when caching was off. Must degrade, per the rule above.
3. **Reference alignment across shards and subworkflows.** `FindCall(name,
   shard)` handles the flat case; subworkflows need the same scoping the cost
   breakdown had to adopt (shards scoped per subworkflow instance). The
   experiment only covered flat, unsharded calls — this is the main untested
   dimension.
4. **Hash key taxonomy is Cromwell-version dependent.** Treat unknown keys
   generically ("`<key>` differs") rather than switching exhaustively on them.
   Validated on Cromwell 91 only.
5. **Backend-dependent file hashing.** The showcase uses the Local backend
   with `hashing-strategy: "file"` (content MD5). GCS-backed Cromwell hashes
   differently (typically the object's crc32c). The *shape* of the diff is
   unaffected, but the phrasing "file content changed" is only safe under
   content hashing — keep the wording neutral ("this input's hash changed")
   unless the strategy is known.

## Open questions

- Should the aggregate report "money left on the table" — the cost the missed
  calls would *not* have incurred on a full hit? Attractive, but it must
  respect the real-vs-estimated cost separation from #46 and not sum units.
- Is the unusable-candidate verdict (row 2) worth a distinct exit code, so CI
  can tell "my cache got wiped" from "my pipeline legitimately changed"?
