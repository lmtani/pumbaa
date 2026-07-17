# Oportunidades de melhoria (backlog técnico)

Segunda geração deste backlog (2026-07-17). A primeira (review de organização
do `internal/`, 2026-07-16) foi integralmente resolvida nos PRs #46–#51:
separação custo real/estimado, shards por instância de subworkflow, lint
zerado + CI, testes do tui/chat, telemetry port + prompts consolidados +
testes do mapper, e as ações failures/read_log/cost/preemption do agente.

Itens abaixo foram identificados durante esses trabalhos, em ordem
recomendada.

## 1. Extrair o loop de agente do chat para um componente testável

`generateResponse` (`tui/chat/stream.go`) é uma função de ~190 linhas que
dirige a conversa inteira: muta `*m.history` de dentro de goroutine (com
workaround explícito `pendingFlush` em `SetSession` para não correr race), e
o descarte de mensagens de conversas abandonadas depende de comparações de
ponteiro (`owner *[]ChatMessage`) espalhadas pelo `Update`. Os testes cobrem
as costuras, mas o loop em si não é testável de ponta a ponta por estar
soldado ao bubbletea.

**Proposta:** extrair um "conversation engine" em Go puro — dono do
histórico, emitindo eventos — testável com um LLM fake roteirizado
(tool-call no meio de streaming, cancelamento entre turnos, sessão anexando
no meio da geração). Maior impacto de confiabilidade do projeto.

## 2. Resumo de preempção ainda mistura unidades

O breakdown de custo separa dólares de resource-hours (#46), mas
`PreemptionSummary.TotalCost/WastedCost` seguem somando os dois com
`CostUnit: "resource-hours"` fixo. O "custo desperdiçado" do debug view tem
o defeito que o modal de custo corrigiu. `attemptCostParts` já existe —
aplicar o mesmo padrão.

## 3. Boilerplate do cliente Cromwell

`cromwell/client.go` tem 12 métodos com o mesmo esqueleto (montar request,
GET/POST, checar status, decodificar JSON, `defer resp.Body.Close()`). Um
helper único (`doJSON(ctx, method, path, &out)`) cortaria ~200 linhas e
centralizaria timeout/retry/tratamento de status.

## 4. Testes de integração reais adormecidos

Dois testes valiosos nunca rodam em CI: o E2E das tools
(`PUMBAA_TOOLS_E2E=1`, agora cobrindo as ações novas) e a validação de custo
contra a API (`PUMBAA_COST_VALIDATE`). Um workflow manual/agendado com
Cromwell em Docker cobriria o primeiro; o segundo pode rodar sempre com um
metadado expandido anonimizado commitado como fixture + valor esperado —
teria pego o bug do custo de tasks em execução antes de chegar à tela.

## 5. Falha de persistência de sessão é invisível

Se o SQLite falhar no meio de uma conversa (disco cheio, lock), cada
`AppendEvent` é descartado silenciosamente (`_ =` deliberado) e o usuário só
descobre ao tentar retomar a sessão. Um aviso único no transcript na
primeira falha resolve — o mecanismo de `notice` já existe.

## 6. Menores

- **Ações de escrita para o agente (abort/submit)** — bloqueadas pela falta
  de UX de confirmação no loop do chat (`noopToolContext` pula a confirmação
  do ADK). Desenhar a confirmação primeiro.
- **`ARCHITECTURE.md` desatualizado** — não acompanhou os refactors de
  2026-07 (ports novos, composition root, pacote prompts, wdlindexer).
- **Migrações do SQLite** — `ALTER TABLE` com erro ignorado funciona, mas
  `PRAGMA user_version` seria explícito.
- **Custo cacheado no modal congela para workflows running** — sem watch
  mode, o valor acumulado exibido fica no instante da abertura.
- **Cobertura da camada de interfaces (contínuo)** — handler 12.9%,
  dashboard 15.3%, common 14.4%, debug ~20%; o padrão dos testes do chat
  (#49) se aplica.
