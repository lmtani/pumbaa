# Cache Forecast

Find out what a submission would actually run — and what it would reuse — before starting it.

<div class="grid cards" markdown>

-   :material-crystal-ball: **Before, not after**

    Answers "is this run worth starting?" while it still costs nothing

-   :material-source-branch: **Root cause, not symptoms**

    One changed task and forty downstream ones read as one finding

-   :material-scale-balance: **Says what it does not know**

    Reuse is claimed only where it was established

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa workflow cache-forecast \
  --workflow analysis.wdl \
  --inputs inputs.json \
  --dependencies imports.zip \
  --against <previous-run-id>
```

Alias: `pumbaa workflow forecast`

## :material-flag: Flags

| Flag | Description |
|------|-------------|
| `--workflow`, `-w` | [required] Path to the WDL workflow file |
| `--inputs`, `-i` | Path to the inputs JSON file |
| `--dependencies`, `-d` | Imports ZIP; without it, imports are read from files beside the workflow |
| `--against`, `-a` | The previous run to compare against |
| `--json` | Emit the forecast as JSON |

## :material-help-circle: Choosing what to compare against

The forecast works by comparing your submission against **one previous run**. Which run
that is decides the answer, so Pumbaa will not pick for you when there is a choice to
make:

- **One previous successful run** — it is used automatically. There is nothing to choose.
- **Several** — Pumbaa stops and lists them, ordered by how close each one's inputs are
  to yours:

```
Which run should this be compared against?
─────────────────────────────────────────
  30 previous runs of AnalysisWorkflow could serve as the reference. Pick the one whose
  inputs match what you are about to submit — comparing against a different
  one reports work as new when it would in fact be reused.

  → 87c1b0d9-e694-4358-acaa-d40b7c22418a  2026-07-17 20:08  same parameters
    c003a473-7c9d-4de3-bfc4-058e410345d2  2026-07-11 01:24  3 of 16 parameters differ
    …and 24 older run(s), all differing more than those above

  Re-run with:  --against <id>
```

Comparing against an unrelated run is not dangerous, but it is useless: every input
looks new, so the forecast reports work that would in fact be served from cache. The
list exists so you do not have to guess from dates alone.

## :material-format-list-checks: Reading the result

Every call gets one of four verdicts.

| Verdict | Meaning |
|---------|---------|
| **would be served from cache** | Every input was checked and matches. This is the claim the tool stakes its usefulness on. |
| **will run again** | Something in this call's own definition or inputs changed. This is a root cause — the thing you can act on. |
| **may run again** | The call itself is unchanged, but something upstream will rerun. **Not a prediction that it will.** |
| **could not determine** | Something resisted checking. Never treat this as "will be reused". |

```
ℹ 14 of 23 call(s) would be served from cache; 1 will run again and 6 more may,
  depending on whether their inputs really change

  Will run again (1):
    ✗ AlignReads              docker image changed (aligner:1.2 → aligner:1.3)

  May run again (6) — downstream, only if the rerun changes their inputs:
    ↓ SortAlignments          after AlignReads
    ↓ CallVariants            after AlignReads
```

### Why "may run again" is not "will"

A task that reruns can still produce **byte-identical output**. When it does, everything
downstream matches the cache and costs nothing. This has been measured: a tool upgrade
made one task rerun, and both of its consumers were served from cache anyway because the
new version wrote the same bytes.

That is why certain and possible reruns are counted separately. Reading "6 may run again"
as "6 will run again" overstates the cost of the run, sometimes by a lot.

### Partly reused tasks

A scattered task is cached shard by shard. When some shards match and others do not, the
call is reported as partly reused with the split, rather than being forced into a single
verdict that would be wrong either way:

```
  Partly reused (1) — a fan-out whose instances differ:
    ◑ AlignReads              18 of 20 instances reused
```

## :material-shield-check: What it will not claim

The forecast is advisory. **Cromwell decides**, and Pumbaa is built so that being wrong
lands on the safe side: it will report work as new when it was unsure, and never the
reverse.

Concretely, it declines to answer rather than guess when:

- **The backend is not local or GCP.** How a file is fingerprinted depends on the backend
  and its configuration, so a confident answer elsewhere would be a wrong one.
- **A task's definition cannot be read** — an import that was not supplied. Pass
  `--dependencies` and it becomes readable.
- **An input's value cannot be worked out** from the WDL and the inputs file.
- **The workflow was rewired** so that an input now comes from somewhere the previous run
  did not read.

Anything it could not check is listed under "could not determine", with the reason.

## :material-alert: Known limits

- **Deleted outputs.** If the run being reused from had its outputs removed, Cromwell
  finds the entry but cannot copy from it and reruns. The forecast does not check that
  the outputs still exist, so it would report reuse.
- **Values computed by the WDL** — a disk size derived from an input's size — are assumed
  to follow the inputs they derive from. They are listed at the end of the output.
- **Fixed text inside an interpolated string** is not compared, so changing it goes
  unnoticed.

## :material-lightbulb: When to use it

- **Before an expensive rerun.** "14 of 23 reused" and "nothing reused" are different
  decisions, and today you only learn which after paying.
- **After changing a container image or a command.** See how far the change reaches
  before it reaches your bill.
- **When a run you expected to be free was not.** Compare against the run you thought it
  would reuse from, and the root cause is named.

## :material-connection: Related

- [Prepare a Submission](guided-submit.md) — check a submission is valid before sending it
- [Diff Two Runs](diff.md) — compare two runs that already happened
