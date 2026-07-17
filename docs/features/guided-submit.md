# Prepare a Submission

Start from the workflow's own declarations instead of a blank inputs file, and
find out whether a run will work **before** spending time and money on it.

<div class="grid cards" markdown>

-   :material-file-document-edit: **Scaffold**

    Generate an inputs template from the WDL, with types and documentation

-   :material-airplane-check: **Preflight**

    Check server, WDL, inputs and file paths before submitting

-   :material-shield-check: **Automatic**

    `submit` runs the checks for you

</div>

## :material-rocket-launch: The flow

```bash
pumbaa workflow scaffold  -w main.wdl -o inputs.json   # 1. generate
# ... fill in the placeholders ...
pumbaa workflow preflight -w main.wdl -i inputs.json   # 2. check
pumbaa workflow submit    -w main.wdl -i inputs.json   # 3. run (checks again)
```

## :material-file-document-edit: Scaffold

```bash
pumbaa workflow scaffold --workflow FILE [OPTIONS]
```

Alias: `pumbaa wf template`

| Flag | Alias | Required | Description |
|------|:-----:|:--------:|-------------|
| `--workflow` | `-w` | :material-check: | WDL workflow file |
| `--output` | `-o` | | Write to this file instead of stdout |
| `--all` | | | Include optional inputs, with their defaults |
| `--force` | | | Overwrite an existing output file |

Required inputs come first, each with a placeholder carrying its type and —
when the WDL documents it in `parameter_meta` — its description:

```json
{
  "AlignReads.reads_fastq": "<FILL: File — Sequencing reads in FASTQ format>",
  "AlignReads.reference_files": "<FILL: Array[File]+ — Reference genome and its index files>",
  "AlignReads.sample_name": "<FILL: String — Identifier used to name outputs>"
}
```

Optional inputs are left out by default, so the file is minimal and ready to
submit once filled. Add `--all` to include them with their default values.

With `--output`, the command also prints every input — including the optional
ones — and the next steps:

```text
✓ Wrote inputs.json for workflow AlignReads

 INPUT                       TYPE          REQUIRED  DEFAULT  DESCRIPTION
 AlignReads.reads_fastq      File          yes                Sequencing reads in FASTQ format
 AlignReads.reference_files  Array[File]+  yes                Reference genome and its index files
 AlignReads.sample_name      String        yes                Identifier used to name outputs
 AlignReads.threads          Int           no        4
 AlignReads.skip_qc          Boolean       no        false
```

!!! tip "Piping"
    Without `--output` only the JSON is printed, so
    `pumbaa workflow scaffold -w main.wdl > inputs.json` works.

## :material-airplane-check: Preflight

```bash
pumbaa workflow preflight --workflow FILE [OPTIONS]
```

Alias: `pumbaa wf check`

| Flag | Alias | Required | Description |
|------|:-----:|:--------:|-------------|
| `--workflow` | `-w` | :material-check: | WDL workflow file |
| `--inputs` | `-i` | | Inputs JSON file |
| `--skip-paths` | | | Do not check that input files exist |
| `--skip-server` | | | Do not check that Cromwell is reachable |

```text
Preflight — workflow AlignReads
─────────────────────────────────────────
  ✓ Cromwell server  reachable
  ✓ WDL syntax       workflow AlignReads
  ⚠ Inputs           worth a look
      ⚠ AlignReads.threads: is the quoted number "8" where Int is expected
      ⚠ AlignReads.sampleName: not declared by this workflow — check for a typo
  ✗ Input files      missing files
      ✗ AlignReads.reference_files[1]: file does not exist: gs://bucket/ref.fai

✗ 1 problem(s) must be fixed before this run can start (2 warning(s) too)
```

The command exits non-zero when there are errors, so it can gate a script or
CI job.

### What is checked

| Check | Blocks? | Notes |
|---|:---:|---|
| Cromwell reachable | :material-check: | Skipped with `--skip-server` |
| WDL parses | | A parse failure is only a warning — Cromwell is the authority |
| Required inputs present | :material-check: | |
| Placeholders replaced | :material-check: | Catches "scaffolded and submitted unedited" |
| Types match declarations | :material-check: | Only clear mismatches; coercible values warn instead |
| Keys declared by the workflow | | Usually a typo, so a warning |
| `File` inputs exist | :material-check: | Missing is an error; **unverifiable** (no credentials) is only a warning |

!!! note "Missing vs. unverifiable"
    If a path cannot be checked — no local cloud credentials, for instance —
    preflight warns instead of blocking: Cromwell may well have access this
    machine does not.

## :material-shield-check: Submit runs it for you

`pumbaa workflow submit` runs the same checks before sending anything, so a
broken submission fails in seconds instead of minutes:

```text
✗ 1 problem(s) must be fixed before this run can start

ℹ Nothing was submitted. Fix the problems above, or use --skip-preflight to submit anyway.
```

The server check is skipped there (submitting contacts the server anyway).
Use `--skip-preflight` to bypass the checks entirely.

## :material-book-open-variant: See Also

- [:material-upload: Submit Workflow](submit.md)
- [:material-package: Bundle Creation](bundle.md)
