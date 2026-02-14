# Release Notes — v4.20260214

> **Data:** 2026-02-14
> **Branch:** `v4`
> **Commit:** `21e12ed`

---

## 🌟 Destaques

### Novo Motor de Execução — Session Manager

A forma como os laboratórios são executados foi **completamente reformulada**. O sistema agora usa um modelo de **"Session Manager"** onde um container Docker de longa duração é criado e reutilizado para os passos de execução e validação — em vez de criar e destruir um container para cada operação.

**O que muda para o utilizador:**
- ⚡ **Execução mais robusta** — retry automático (3 tentativas) em caso de falhas transitórias na criação do container.
- 🎯 **Validação separada** — o resultado de execução e validação são reportados como eventos independentes no WebSocket, com mensagens de sucesso/falha distintas.
- 🔄 **K8s com retry** — validação de labs Kubernetes agora aguarda até 30 segundos para que os recursos fiquem prontos, com tentativas a cada 2 segundos.

### Simplificação do Fluxo WebSocket

O handler WebSocket foi simplificado. O fluxo de validação automática após execução bem-sucedida — anteriormente orquestrado no handler com flags de controle — agora é gerido internamente pelo executor. O handler é um consumidor passivo que apenas reporta os resultados ao cliente.

---

## 🚀 Melhorias e Alterações

### Backend — Executor (`internal/executor/docker_executor.go`)

- **Novo padrão Session Manager**: containers criados com `tail -f /dev/null` como entrypoint, passos executados via `docker exec`.
- **Retry na criação de containers**: 3 tentativas com delay crescente (1.5s, 3s) — resolve race conditions do Docker Desktop + WSL2.
- **Stream de logs melhorado**: demultiplexação com `stdcopy.StdCopy` → leitura linha-a-linha via `bufio.Scanner`(mais fiável que leitura por buffer).
- **Validação K8s com retry**: método `runWithRetry` com timeout de 30s e polling de 2s.
- **Removidos métodos obsoletos**: `getContainerConfig`, `streamLogs`, `buildCommand`, `streamPipe`.
- **Removida dependência** de `os/exec` — toda interação Docker é agora via SDK.

### Backend — Handler (`internal/api/handler.go`)

- **Eliminação do fluxo de duas fases**: removidas variáveis `isValidation` e `shouldValidateAfter`.
- **Handler simplificado**: de 333 para 306 linhas — responsabilidade única (streaming + feedback).
- **Inspeção direta de resultados**: o handler verifica `state.ValidationResult.ExitCode` e `state.ExecutionResult` sem necessidade de re-invocar serviços.

### Backend — Contrato de Domínio (`internal/service/ports.go`)

- **`ExecutionFinalState` expandido**: novos campos `ExecutionResult` e `ValidationResult` (tipo `domain.StepResult`), permitindo inspeção granular de cada fase.

### Infraestrutura (`.gitignore`)

- Adicionada exclusão para diretório `.agent/`, ficheiros `TODO.md` e `*.spec.md`.
- Padrão de logs expandido: `log_*.txt` (antes apenas `log_execução.txt`).
- Preservação do diretório `data/temp-exec/` via `.gitkeep`.

---

## 🐛 Correções

- **Container mount race condition**: adicionados delays de sincronização para ambientes Docker Desktop + WSL2.
- **Containers órfãos**: lifecycle gerido explicitamente com `startContainer` / `stopContainer` + `Force: true` na remoção.
- **Ansible validação encadeada**: anteriormente executada inline no mesmo comando shell (`&& ansible-playbook validation.yml`), agora como passo separado via `docker exec` — isolamento e reporting independente.

---

## ⚠️ Breaking Changes

- Nenhum breaking change na API pública (WebSocket + REST permanecem iguais).
- A estrutura interna de `ExecutionFinalState` foi alterada (adição de campos) — afeta apenas código que consuma diretamente este struct.

---

## 📋 Tipos de Lab Suportados

| Tipo | Status | Notas |
|------|--------|-------|
| Terraform | ✅ | Execução + state persistence |
| Ansible | ✅ | Validação agora em passo separado |
| Linux | ✅ | Sem alteração funcional |
| Docker | ✅ | Sem alteração funcional |
| Kubernetes | ✅ | **Novo:** retry na validação |
| GitHub Actions | ✅ | Sem alteração funcional |

---

*Release gerada em 2026-02-14 a partir do commit `21e12ed` (branch `v4`).*
