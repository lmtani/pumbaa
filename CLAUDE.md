# CLAUDE.md — Pumbaa (Ultra-Resumo)

## O que é
Pumbaa é um **CLI/TUI em Go** para operar workflows **Cromwell + WDL**, seguindo **Clean / Hexagonal Architecture**.  
Inclui TUIs interativas e um **agente LLM** para debug e análise de workflows.

---

## Capacidades Principais
- Submit / query / abort / debug workflows Cromwell
- Dashboard e debug TUI (Bubble Tea)
- Indexação e busca de WDL
- Criação de bundles WDL (ZIP com dependências)
- Análise de uso de recursos
- Chat agent com LLMs (Gemini, Vertex, Ollama)
- Telemetria opcional (Sentry)

---

## Arquitetura (essencial)

**Fluxo de dependências (sempre para dentro):**

Interfaces → Application → Domain

Infrastructure → Domain (implementa ports)


- **Domain**: entidades + interfaces (ports), sem dependências externas
- **Application**: use cases (1 operação = 1 caso de uso)
- **Infrastructure**: Cromwell, LLMs, GCS, SQLite, WDL parser
- **Interfaces**: CLI e TUIs
- **Container**: injeta tudo

---

## Contratos-Chave (Domain Ports)
- `WorkflowRepository`: tudo sobre workflows Cromwell
- `FileProvider`: acesso a arquivos (local/GCS)
- `WDLRepository`: indexação e busca de WDL

Application depende **somente** desses ports.

---

## Use Cases (Application)
- `submit`, `metadata`, `abort`, `query`
- `debug` (metadata → árvore de execução)
- `monitoring` (CPU/mem/disk)
- `bundle` (resolver dependências WDL)

Erros são sempre encapsulados em erros de application layer.

---

## Infrastructure (Adapters)
- Cromwell REST client
- LLM providers + tool registry
- WDL indexer (ANTLR + cache JSON)
- Storage (local/GCS)
- Chat sessions (SQLite)
- Telemetry (Sentry / NoOp)

---

## Interfaces
- **CLI**: handlers chamam use cases, presenters formatam saída
- **TUI**: dashboard, debug, chat, config wizard

Nenhuma lógica de infraestrutura aqui.

---

## Princípios Importantes
- Domain não conhece frameworks
- Use cases não conhecem UI nem infra concreta
- Infra é substituível via ports
- Um `WorkflowRepository` unificado (contrato único)
- File access sempre via `FileProvider`

---

## CI/CD
- GitHub Actions
- CI: lint + tests + build
- Release: GoReleaser (Linux/macOS/Windows)
