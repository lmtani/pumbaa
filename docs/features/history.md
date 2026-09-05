# Local Run History

What was submitted from this machine, and what it was for.

<div class="grid cards" markdown>

-   :material-note-text: **A sentence, not a tag**

    `--describe` says what a run is for, in prose

-   :material-database-off: **Outlives the server**

    Records survive a Cromwell restart, and answer offline

-   :material-pencil: **Any run, any time**

    Notes attach to runs someone else started too

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa workflow submit -w pipeline.wdl -i inputs.json \
  -D "rerun after the reference panel was rebuilt"

pumbaa history                    # what has been submitted, newest first
pumbaa history show <workflow-id> # everything remembered about one run
```

## :material-help-circle: What is kept, and why

A Cromwell server already answers what a run *did*. The local history keeps
what it cannot answer later:

- the **description** given at submission time — Cromwell labels are
  constrained and meant to be tags, so free text has nowhere else to live;
- the **local files** a submission was assembled from (WDL, inputs, options,
  dependency zip), stored as absolute paths so they are still findable a month
  later;
- the last known **status**, which is what a listing shows when the server is
  unreachable — or has forgotten the run entirely, as an in-memory Cromwell
  does on every restart.

Records are keyed by `(host, workflow id)`, since an ID is only unique within a
server. Writing to the history never fails a submission: the workflow is
already running by then, so a failure is reported as a warning.

!!! info "Where it lives"
    `~/.pumbaa/history.db` (SQLite), separate from the chat sessions database.
    Override with `PUMBAA_HISTORY_DB`.

## :material-console: Commands

| Command | Description |
|---------|-------------|
| `pumbaa history` | List remembered runs of the active host |
| `pumbaa history show <id>` | Everything remembered about one run |
| `pumbaa history note <id> <text>` | Write or rewrite a run's description |
| `pumbaa history forget <id>` | Drop one run from the history |
| `pumbaa history prune --older-than 90d` | Drop everything older than a cutoff (`--yes` to confirm) |

### List flags

| Flag | Alias | Description |
|------|:-----:|-------------|
| `--all` | `-a` | Every host, not just the active one |
| `--search` | `-s` | Keywords; every one must appear somewhere in the record |
| `--since` | | `7d`, `24h` or `2026-07-01` |
| `--limit` | `-l` | Maximum rows (default 20) |
| `--no-refresh` | | Do not ask the server for current statuses |
| `--json` | | Print the records as JSON |

```
$ pumbaa history

Run history
─────────────────────────────────────────
 STATUS      ID                                    NAME          SUBMITTED         DESCRIPTION
 Succeeded   fbfd8cce-90f5-4d07-81b0-3504bd0ef98c  HelloHistory  26-09-05 09:53    smoke test after the docker bump
 Succeeded*  deadbeef-0000-0000-0000-000000000001  GhostRun      25-03-01 07:00    the server restarted after this

ℹ * the server no longer knows this run — only the local record remains
```

Statuses are refreshed in a single query filtered by run ID. When the server
cannot be reached the listing still works, from the last known statuses, and
says so.

!!! note "Why a run is only marked `*` after a while"
    Cromwell answers `/query` from a summary table filled by a background job,
    so a run submitted moments ago is routinely missing from it while very much
    existing. A run is only reported as forgotten once it is old enough for
    that lag to be ruled out.

## :material-magnify: Searching

`--search` takes keywords, not a phrase: the terms are matched independently,
in any order, and every one of them must appear somewhere in the record —
description, workflow name, ID, labels, or the paths of the WDL, inputs,
options and dependency files.

```bash
pumbaa history --search "tso500 referência"   # both words, not necessarily adjacent
pumbaa history --search hg38                  # runs whose inputs path mentions hg38
pumbaa history --search S001                  # a label value
```

Matching is case-insensitive, and `%` and `_` are searched for as text rather
than treated as wildcards.

## :material-pencil-plus: Annotating someone else's run

A note can be attached to any run on the server, not only to ones submitted
from this machine:

```bash
pumbaa history note 25704b54-4ae6-408d-9c9c-f3b5ed021d2f \
  "the run that produced the reference panel"
```

The run is looked up on the server first, so its real name and submission time
are recorded rather than invented. A run that does not exist there is refused.

Clearing the note of a run that was only remembered *for* that note drops the
record; a run submitted from here keeps its record, because the files it ran
with are worth remembering on their own.

## :material-monitor-dashboard: In the dashboard

| Key | Action |
|-----|--------|
| `n` | Write or edit the note of the selected run |
| `H` | Open the local history, including runs the server forgot |

Rows say whether this machine remembers them — `✎` when the run carries a
note, `•` when it was submitted from here without one — and the note itself is
shown where the row has room for it.

The `H` modal is the only place runs the server no longer has can be seen: the
dashboard table is a view of the server. From it, `enter` opens a run that
still exists and `n` writes a note about any of them.

## :material-robot: In the chat agent

The `history` action gives the agent the same memory, so it can answer what a
run was for and what has been submitted recently — neither of which the
Cromwell server knows:

```
> what was workflow abc-123 for?
> what have I been running this week?
```

Besides `workflow_id` and a keyword `query`, the action takes `status` and
`since_days`, so "what failed last week" is one call rather than a scan. It
reads only what this machine remembers — no Cromwell server is contacted, so
it keeps working offline; live state still comes from the Cromwell actions.

## :material-lightbulb: See also

- [Submit workflows](submit.md) — where `--describe` is given
- [Cromwell hosts](hosts.md) — history is scoped per host
