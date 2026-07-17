# Design: guided submit (scaffold + preflight)

Status: implemented (first iteration).
Audience: contributors. See `features/submit.md` for user documentation.

## Problem

Pumbaa is strong once a run exists — dashboard, watch, failure summary, cost,
preemption. It is weakest exactly where newcomers spend most of their pain:
**getting to a correct submission**.

The newcomer's loop today is: receive a WDL, hand-write an `inputs.json`
without knowing which fields are required or what types they take, guess,
submit, wait, and get a failure that is about infrastructure (a wrong
`gs://` path, a missing input) 20 minutes and a few dollars later.

The only guard rail today is `wdl.ValidateInputs`, called by the submit use
case: it detects missing required inputs and nothing else.

## Goals

1. **Scaffold**: generate an `inputs.json` the user can fill, from the WDL
   itself — required inputs first, typed placeholders, defaults and
   `parameter_meta` descriptions surfaced.
2. **Preflight**: answer "is this submission going to work?" before spending
   time and money — server reachable, WDL parses, required inputs present,
   no placeholders left, types plausible, no unknown keys, file paths
   actually exist.
3. **Fail in 2 seconds, not 20 minutes**: submit runs the preflight checks
   automatically.

Non-goals (for this iteration): fixing anything automatically and estimating
cost (separate design). Exposing scaffold/preflight through the chat agent
was a follow-up, now done (see Follow-ups).

## Why the data is already there

The WDL parser (`pkg/wdl`, ANTLR) already produces everything needed:

- `ast.Workflow.Inputs []*Declaration` — each with `Type` and `Expression`
  (nil when there is no default).
- `ast.Type{Base, Optional, ArrayType, MapKey/MapValue, PairLeft/PairRight,
  NonEmpty}` plus a `String()` renderer.
- `ast.Workflow.ParameterMeta map[string]any` — the WDL convention for
  documenting inputs.
- `pkg/wdl/validate.go` already walks required inputs (`requiredInputNames`)
  and is already wired into `SubmitUseCase`.

So this feature is mostly *surfacing* knowledge the parser already has.

## Design

### Layering

```
pkg/wdl/inputs.go          Pure WDL knowledge: extract specs, render the
                           template, check an inputs JSON against the WDL.
                           No IO, no network.
application/workflow/      ScaffoldInputsUseCase, PreflightUseCase:
  scaffold.go              orchestration + IO through ports
  preflight.go             (FileProvider, HealthChecker).
interfaces/cli/handler/    `workflow scaffold`, `workflow preflight`,
                           shared checklist rendering; submit renders the
                           same report when it blocks.
```

The checks that need the outside world (does `gs://…` exist, is Cromwell
up) live in the use case behind existing ports; everything decidable from
the WDL and the JSON alone stays in `pkg/wdl` and is trivially testable.

### Scaffold

`WorkflowInputs(source)` returns an ordered `[]InputSpec{Name, Type,
Optional, Default, Description, Required}`; `Name` is qualified
(`workflow.input`) as Cromwell expects.

`ScaffoldInputs(source, opts)` renders JSON **preserving declaration order**
(built by hand, not via a Go map, which would shuffle keys), required inputs
first:

- Required input → placeholder sentinel: `"<FILL: File — path to reads>"`
  (type, plus the `parameter_meta` description when present).
- Optional inputs are omitted by default (the file stays minimal and
  submit-ready once filled) and included with `--all`, rendered as their
  default value when it is a literal, `null` otherwise.

The sentinel is deliberately machine-detectable: preflight flags any
remaining `<FILL:` value, so "I generated the template and submitted it
unedited" fails instantly with a clear message instead of confusing Cromwell.

### Checks (`CheckInputs`)

Returns findings with two severities — **error** (blocks submit) and
**warning** (reported, does not block) — plus the list of file-typed values
for the use case to verify:

| Check | Severity | Rationale |
|---|---|---|
| WDL does not parse | warning | Our parser may lag the WDL spec; Cromwell is the authority. Same philosophy as today's `ValidateInputs`, which returns nil on parse failure. |
| Required input missing | error | Cromwell will reject it. |
| Value still a `<FILL:` placeholder | error | Template was not filled in. |
| Clear type mismatch (array where scalar expected, object where string expected, non-number for `Int`…) | error | Cromwell will reject it. |
| Suspicious type (quoted number for `Int`, `Int` for `Float`) | warning | Cromwell coerces some of these; a false block would be worse than a note. |
| Key not declared by the workflow | warning | Usually a typo (`sampleName` vs `sample_name`), but subworkflow-qualified keys exist that our parser does not model. |

The type checker is deliberately conservative: it only calls something an
error when no reasonable coercion exists. Being annoying about a valid
submission would burn the trust the feature depends on.

### Preflight use case

Runs, in order, and always reports every check (never stops at the first
failure — the newcomer wants the whole list):

1. **Cromwell server** — `ports.HealthChecker`; skipped when `--skip-server`.
2. **WDL syntax** — parse via `pkg/wdl`.
3. **Inputs** — `CheckInputs` findings.
4. **File paths** — `FileProvider.GetSize` for every `File`-typed value
   (metadata-only on GCS, no data transfer), concurrently with a small
   worker cap; skipped with `--skip-paths`.

Path check semantics matter: **not found is an error, anything else is a
warning**. A user without local GCP credentials must not be blocked from
submitting to a Cromwell that does have them — "could not verify" is not
"broken". This requires distinguishing the two, so `ports.ErrFileNotFound`
is (re)introduced as a sentinel and both storage backends wrap not-found
errors with it.

### Submit integration

`SubmitUseCase` gains the preflight use case as a dependency and runs it
(server check skipped — submit is about to contact the server anyway)
before submitting. On errors it returns a typed `PreflightFailedError`
carrying the report; the handler renders the same checklist as the
`preflight` command. `--skip-preflight` bypasses it.

This replaces the narrower `wdl.ValidateInputs` call: same spirit (fail
before Cromwell does), much wider coverage.

## CLI

```bash
pumbaa workflow scaffold -w main.wdl                 # template to stdout
pumbaa workflow scaffold -w main.wdl -o inputs.json  # write + explain
pumbaa workflow preflight -w main.wdl -i inputs.json # checklist, exit 1 on errors
pumbaa workflow submit -w main.wdl -i inputs.json    # preflight runs first
```

`scaffold` without `-o` prints only JSON, so `> inputs.json` works; with
`-o` it writes the file and prints the input table plus next steps (the
teaching moment). `--all` includes optional inputs; `--force` overwrites.

## Testing

- `pkg/wdl`: extraction (types, optionality, defaults, `parameter_meta`),
  scaffold (ordering, placeholders, `--all`, literal defaults), checks (each
  row of the table above, including the coercions that must *not* be flagged).
- Application: preflight with fake provider/health (all-green, missing file,
  not-found path vs unverifiable path, skip flags), submit blocked by
  preflight errors and bypassed with the flag.

## Follow-ups

- ~~**Agent actions** (`scaffold`/`preflight` in the chat tool)~~ — done. The
  chat agent can scaffold a template and preflight it, reading local WDL and
  inputs files sandboxed to the working directory (the shared helper
  `localfs.ResolveWorkingDirPath`). The agent's preflight skips the server
  check (it already reaches Cromwell for other actions) and reuses the same
  pure `wdl.CheckInputs`, so the meaningful logic cannot drift from the CLI.
- **Cost estimate** in preflight: `ast.Task.Runtime` is already parsed, so a
  price band and resource sanity checks ("512 GB of memory — typo?") fit
  naturally as another check. Separate design.
- **Scatter-aware path checks**: `Array[File]` inputs with thousands of
  entries are capped today; sampling strategy to be decided.
