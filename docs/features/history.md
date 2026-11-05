# Local Run History

What was submitted from this machine, and what it was for.

<div class="grid cards" markdown>

-   :material-note-text: **A sentence, not a tag**

    `--describe` says what a run is for, in prose

-   :material-database-off: **Outlives the server**

    Records survive a Cromwell restart, and answer offline

-   :material-pencil: **Any run, any time**

    Notes attach to runs someone else started too

-   :material-package-variant-closed: **Run it again**

    The submitted files are kept, and `export` writes them back out

</div>

## :material-rocket-launch: Quick Start

```bash
pumbaa workflow submit -w pipeline.wdl -i inputs.json \
  -D "rerun after the reference panel was rebuilt"

pumbaa history                    # what has been submitted, newest first
pumbaa history show <workflow-id> # everything remembered about one run
pumbaa history export <workflow-id> ./rerun   # the files, ready to submit again
```

## :material-help-circle: What is kept, and why

A Cromwell server already answers what a run *did*. The local history keeps
what it cannot answer later:

- the **description** given at submission time — Cromwell labels are
  constrained and meant to be tags, so free text has nowhere else to live;
- the **files** a submission was assembled from — WDL, inputs, options and
  the dependency zip — kept verbatim, as the very bytes the server received,
  alongside the absolute paths they were read from. Paths rot: a WDL gets
  edited, a branch is switched, a directory is deleted. The bytes do not;
- the last known **status**, which is what a listing shows when the server is
  unreachable — or has forgotten the run entirely, as an in-memory Cromwell
  does on every restart.

Records are keyed by `(host, workflow id)`, since an ID is only unique within a
server. Writing to the history never fails a submission: the workflow is
already running by then, so a failure is reported as a warning.

!!! info "Where it lives"
    `~/.pumbaa/history.db` (SQLite), separate from the chat sessions database.
    Override with `PUMBAA_HISTORY_DB`. The kept files live in the same
    database, so forgetting a run forgets its files with it and nothing is
    left behind on disk. A WDL bundle is tens of kilobytes; a submission
    whose files exceed 50 MB is recorded without them, with a warning.
    `PUMBAA_HISTORY_KEEP_FILES=false` (or `history_keep_files: false` in the
    config file) records paths only.

## :material-console: Commands

| Command | Description |
|---------|-------------|
| `pumbaa history` | List remembered runs of the active host |
| `pumbaa history show <id>` | Everything remembered about one run |
| `pumbaa history note <id> <text>` | Write or rewrite a run's description |
| `pumbaa history export <id> [dir]` | Write the run's files to a directory, ready to submit again |
| `pumbaa history forget <id>` | Drop one run from the history, kept files included |
| `pumbaa history prune --older-than 90d` | Drop everything older than a cutoff, kept files included (`--yes` to confirm) |

### List flags

| Flag | Alias | Description |
|------|:-----:|-------------|
| `--all` | `-a` | Every host, not just the active one |
| `--search` | `-s` | Keywords; every one must appear somewhere in the record |
| `--since` | | `7d`, `24h` or `2026-07-01` |
| `--limit` | `-l` | Maximum rows (default 20) |
| `--no-refresh` | | Do not ask the server for current statuses |
| `--json` | | Print the records as JSON; `Exportable` says the files were kept |

```
$ pumbaa history

Run history
─────────────────────────────────────────
 STATUS      ID                                    NAME          SUBMITTED         FILES  DESCRIPTION
 Succeeded   fbfd8cce-90f5-4d07-81b0-3504bd0ef98c  HelloHistory  26-09-05 09:53    ✓      smoke test after the docker bump
 Succeeded*  deadbeef-0000-0000-0000-000000000001  GhostRun      25-03-01 07:00           the server restarted after this

ℹ * the server no longer knows this run — only the local record remains
ℹ ✓ files kept: pumbaa history export <workflow-id> writes them out, ready to submit again
```

Statuses are refreshed in a single query filtered by run ID. When the server
cannot be reached the listing still works, from the last known statuses, and
says so.

`pumbaa history show --json <id>` prints one record the same way, and exits 1 when there is none.

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

## :material-package-variant-closed: Running it again, or handing it over

`export` writes a run back out as the files it was submitted with:

```
$ pumbaa history export fbfd8cce-90f5-4d07-81b0-3504bd0ef98c ./rerun

✓ Exported fbfd8cce-90f5-4d07-81b0-3504bd0ef98c to /home/me/rerun
 FILE              SOURCE          SIZE      NOTE
 workflow.wdl      kept at submit  1.2 KiB
 inputs.json       kept at submit  412 B
 options.json      kept at submit  96 B
 dependencies.zip  kept at submit  54.1 KiB
 labels.json       kept at submit  38 B
 manifest.json     pumbaa                    where each file came from

ℹ Submit it again with:
  cd /home/me/rerun && pumbaa workflow submit -w workflow.wdl -i inputs.json -o options.json -d dependencies.zip -l sample=S001
```

The names are fixed so the printed command is always right; the basenames the
files were read from are in `manifest.json`, with the host, the run, the
description and the origin and checksum of every file. The directory is the
unit to share: a colleague with access to the same server runs the command and
gets the same submission.

| Flag | Alias | Description |
|------|:-----:|-------------|
| `--out` | `-o` | Directory to write to; the second argument does the same (default: `./<workflow-id>`) |
| `--force` | | Replace files already in the directory |
| `--strip-secrets` | | Drop credential options (`user_service_account_json`, tokens) before sharing |
| `--json` | | Print the manifest instead of the summary, with `dir` and `complete` added |

The exit status is 1 when a file could not be produced. Whatever could be written is still in the directory, and the manifest says what is missing, so a script can tell a partial export from a failed one.

!!! warning "What the server cannot supply"
    A run nothing was kept for — submitted before files were kept, through
    another client, or with keeping turned off — is read from the server's
    `submittedFiles` instead: exact for the workflow and the inputs,
    reformatted for the options. Cromwell discards the **dependencies zip**
    once a run leaves its workflow store, so for a workflow with imports the
    export says what is missing and which imports need it. When the history
    recorded the zip's path and the file is still there, it is used, marked
    as read now rather than kept then.

Credentials in the options — a service-account JSON, a refresh token — are
reported, and `--strip-secrets` removes them: whoever receives the directory
submits with their own.

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
Runs whose files are kept come back marked `exportable`.

The `export` action is the same as the command: it writes a run's files into
a directory under the one pumbaa was launched from, and answers with the
submit command:

```
> export run abc-123 so I can send it to Ana
> give me the inputs that run used
```

## :material-lightbulb: See also

- [Submit workflows](submit.md) — where `--describe` is given
- [Cromwell hosts](hosts.md) — history is scoped per host
