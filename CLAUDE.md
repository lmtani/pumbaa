# Pumbaa - Guide for Claude

Go CLI/TUI for managing Cromwell workflows (bioinformatics), with an LLM
agent for analysis and debugging. Living technical backlog in
`docs/IMPROVEMENTS.md`.

## Repository rules

- **Lint is a CI gate and must stay at 0**: `make lint` (golangci-lint,
  same version pinned in CI). Run it before committing.
- **Commits/PRs**: Conventional Commits in English (`fix(cost): ...`), body
  explaining the why. **Never** use Co-Authored-By or mention AI.
- Code reachable only from tests is an injection seam, not dead code
  (e.g. `ResourceReportUseCase.Execute`, `storage.NewFileProviderWithBackends`).
- `pkg/wdl` is the library's public API — do not remove "unused" exports.

## Structure

```
cmd/cli/main.go              # Entry point - urfave/cli/v2
internal/
├── domain/workflow/         # Aggregate root + calculations (see below)
├── application/
│   ├── workflow/            # Use cases (Submit, Query, Compare, etc.)
│   └── ports/               # ALL interfaces (hexagonal) — includes
│                            #   ChatSessionStore, UpdateChecker, Telemetry,
│                            #   WorkflowMetadataFetcher, ResourceReportRenderer
├── infrastructure/
│   ├── cromwell/            # HTTP client + metadata mapper
│   ├── agents/              # LLM adapters (gemini/vertex/ollama) + agent tools
│   ├── recommendation/      # LLM-based resource recommendations
│   ├── session/             # SQLite (~/.pumbaa/sessions.db)
│   ├── wdlindexer/          # WDL index backing the agent tools
│   ├── storage/ cloudlogging/ metrics/ telemetry/ templates/ version/
├── interfaces/
│   ├── cli/handler/         # 14 handlers (pattern: useCase + presenter)
│   └── tui/                 # BubbleTea: app.go, common/, dashboard/, debug/, chat/
├── prompts/                 # Agent system instructions (dependency-free)
└── container/               # Composition root — ALL construction happens here
pkg/wdl/                     # WDL parser (ANTLR) — public library
```

**Layering (invariant enforced through imports):** domain → nothing;
application → domain; infrastructure implements ports (aliases +
compile-time checks); interfaces → ports/domain/application, **zero** infra.

## Domain: ready-made calculations

In `domain/workflow/`, all recurse into loaded subworkflows and anchor
Running tasks at `time.Now()`:

- `CalculateCostBreakdown()` — per-task cost; **real dollars
  (`ActualCost`) and the resource-hours estimate (`EstimatedCost`) are
  separate units, never sum them**. Shards are scoped per subworkflow
  instance.
- `CalculatePreemptionSummary()` — preemption efficiency (still blends
  units — item 2 of IMPROVEMENTS.md).
- `CalculateFailureSummary()` — root causes deduplicated by normalized
  signature; only the final attempt counts as failed.

## TUI (BubbleTea)

- `AppModel` (app.go) is the only handler of `Navigate*` messages
  (`common/nav.go`); each screen owns its own keys, including ESC (closes
  modal/search before emitting `NavigateBackMsg`). Async messages are
  broadcast; KeyMsg/spinner go only to the focused screen.
- The debug model is reused when returning from chat (tree/watch preserved).
- Screen packages decompose by file: `model.go`, `update.go`, `view*.go`,
  `types.go`, `styles.go`, `modal_*.go` (+ in chat: `stream.go` with the
  agent loop, `sessions.go` with lazy create/resume).

### Chat

- The chat drives its own agent loop (`chat/stream.go`): streaming with
  partials, tool calls, pushes via `tea.Program.Send` (the handler sets
  `deps.Program` right after `tea.NewProgram`). ESC cancels generation;
  enter sends, ctrl+j inserts a newline; ctrl+r resumes the task's previous
  session; ctrl+s lists/switches sessions.
- Composition (LLM + session store + tools) lives in the container:
  `Container.ChatDependencies` / `Container.SessionStore`. Extended session
  queries via assertion to `ports.ChatSessionStore`.
- Sessions: lazy creation on first send, task `context_label`, single scope
  `ports.DefaultChatAppName`.

### Agent tools (the "pumbaa" tool)

- The `builtinActions` table in `agents/tools/factory.go` is the single
  source of truth (registration + description + schema enum). Adding an
  action = 1 entry + a property in `GetParametersSchema` if it introduces a
  new parameter. Dependencies come through the `Deps{Repo, Fetcher, WDLRepo}`
  struct — actions with a missing dependency are simply not registered.
- Actions: query, status, metadata, outputs, logs, **failures** (root-cause
  summary — prefer over metadata for debugging), **read_log** (stderr/stdout
  tail), **cost**, **preemption**, gcs_download, write_file, wdl_*.
- No write actions (abort/submit) until the chat has a confirmation UX.
- Agent prompts live in `internal/prompts` (update them when actions change).
- Live E2E: `PUMBAA_TOOLS_E2E=1 CROMWELL_HOST=... go test ./internal/infrastructure/agents/tools/ -run TestToolsE2E`
- Cost vs API validation: `PUMBAA_COST_VALIDATE=<metadata.json> PUMBAA_COST_EXPECTED=<usd> go test ./internal/infrastructure/cromwell/ -run TestCostBreakdownMatchesAPI`

## Patterns

- CLI handler: struct with `useCase` + `presenter`; `Command()` returns
  `*cli.Command`. Use case: `Execute(ctx, XxxInput) (XxxOutput, error)`.
- New ports go in `application/ports/`; infrastructure implements them with
  `var _ ports.X = (*Impl)(nil)`; shared types via alias in the infra package.
- Key dependencies: bubbletea/bubbles/lipgloss (TUI), urfave/cli/v2,
  `google.golang.org/adk` (agent framework — session.Service, tool.Tool),
  `google.golang.org/genai`, modernc.org/sqlite (no CGO).

## Commands

```bash
make build / test / test-coverage / lint / fmt / docs-serve
```

## Configuration

Precedence: CLI flags > env vars > `~/.pumbaa/config.yaml` > defaults.
Main ones: `CROMWELL_HOST`, `PUMBAA_LLM_PROVIDER` (ollama|vertex|gemini),
`PUMBAA_WDL_DIR`.
