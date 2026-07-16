# Oportunidades de melhoria (backlog técnico)

Levantamento feito em 2026-07-16, após o refactor de organização do `internal/`
(branch `refactor/internal-organization`: ports para chat/report/version,
composition root unificado no container, split do `tui/chat`, fix do custo de
tasks em execução). Estas são as pendências identificadas na segunda varredura,
em ordem recomendada de ataque.

## 1. Separar custo real de custo estimado no CostBreakdown

**Problema:** tasks sem `vmCostPerHour` caem no estimador `cpu × mem × horas`
(`calculateAttemptCost` em `internal/domain/workflow/preemption.go`), e esse
valor — que é "resource-hours", não dinheiro — é somado no mesmo
`CostBreakdown.TotalCost` exibido com `$` no modal de custo. Num backend sem
custo reportado (ex.: local), o total inteiro vira ficção com cifrão. O
disclaimer "(some values estimated from resources)" não resolve a mistura de
unidades.

**Proposta:** `TaskCost.FromActual` já existe por task. Fazer o
`CostBreakdown` acumular `ActualCost` e `EstimatedCost` separados e o modal
(`internal/interfaces/tui/debug/modal_cost.go`) exibir os dois sem somá-los.

**Esforço:** pequeno. **Impacto:** confiança nos números exibidos.

## 2. Agregação de shards colapsa subworkflows diferentes

**Problema:** o `walk` de `CalculateCostBreakdown`
(`internal/domain/workflow/cost.go`) agrega por `ShardIndex` global, então
instâncias da mesma task em subworkflows distintos são fundidas: no metadado
real de exemplo, Bwamem2Align aparecia como "shards=2, attempts=59". O mesmo
padrão existe na análise de preempção (`taskKey{name, shard}` em
`preemption.go`, `CalculatePreemptionSummary`).

**Proposta:** incluir o caminho do subworkflow na chave de agregação de shard
durante a recursão (ex.: `subPath + shardIndex`).

**Esforço:** pequeno. Casar com o item 1 (mesmos arquivos).

## 3. Zerar o lint e passar a rodá-lo no CI

**Problema:** `make lint` falha com 71 issues (50 errcheck, 19 staticcheck,
2 unused) e o `.github/workflows/ci.yml` roda apenas testes e build — o lint
está permanentemente vermelho e não protege contra regressões.

Concentrações: `interfaces/cli/presenter/presenter.go` (11),
`infrastructure/session/sqlite.go` (9), `pkg/wdl/bundle.go` (5), deprecations
`.Copy()` do lipgloss espalhadas na TUI.

**Proposta:** (a) zerar as 71 issues — maioria mecânica; (b) adicionar
golangci-lint como step do CI para travar no zero.

**Esforço:** médio-mecânico. **Impacto:** CI passa a segurar qualidade.

## 4. Cobertura de testes na camada de interfaces

**Números (go test -cover, 2026-07-16):** domain 89–100%, application
63–100%, mas `interfaces/tui/chat` tem **zero testes** (~2.100 linhas,
incluindo o loop de agente/streaming), `cli/handler` 12.9%,
`tui/dashboard` 15.3%, `tui/common` 14.4%, `tui/debug` 20.4%.

**Proposta:** começar pelo `tui/chat` — o split em arquivos
(`stream.go`, `update.go`, `sessions.go`) e o port `ports.ChatSessionStore`
(mockável) tornaram isso barato. Prioridade: fluxo de streaming/tool-call
(`stream.go`) e transições de estado do `update.go`. Depois, handlers via
mocks de ports.

**Esforço:** contínuo. **Impacto:** rede de proteção onde hoje não há nenhuma.

## 5. Menores

- **`telemetry.Service`** — último edge `interfaces → infrastructure`. Mover a
  interface (+ `Event`/`CommandContext`) para `application/ports` com alias na
  infra, mesmo padrão aplicado em `session` (`ports.ChatSessionStore`) e
  `version` (`ports.UpdateChecker`).
- **Prompts de sistema espalhados** — ~100 linhas inline em
  `interfaces/cli/handler/chat.go` (`systemInstruction`), outro em
  `interfaces/tui/debug/chat_context.go` (`taskDebugSystemInstruction`), outros
  em `infrastructure/recommendation/prompts.go`. Consolidar num pacote de
  prompts para facilitar iteração.
- **`infrastructure/cromwell` a 46% de cobertura** — o cliente é wrapper de
  API, mas o `mapper.go` é lógica pura de parsing de metadado (onde bugs como
  o do custo nascem) e merece mais casos: timestamps ausentes, attempts
  Running, subworkflows aninhados.
- **`application/errors.go`** — avaliado e mantido de propósito: é a taxonomia
  de erros de use case (distinta de `ports/errors.go`, que cobre erros de
  ports). Renomear o pacote geraria churn sem ganho real. Registrado aqui para
  não rediscutir.

## Ordem recomendada

1 → 2 (fecham a história de custo, pequenos e nos mesmos arquivos), depois 3
(habilita o CI a segurar o resto), e 4 como esforço contínuo começando pelo
chat. Item 5 conforme oportunidade.
