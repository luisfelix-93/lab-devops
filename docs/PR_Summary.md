# PR Summary — `75e6906` (v4) — Feature: Health Check API

> **Branch:** `v4`
> **Commit:** `75e690687d7f` — `20260217 - #11 implementação de health-check`
> **Author:** luisfelix-93
> **Date:** 2026-02-17

---

## 🎯 Objetivo

Implementar um endpoint de **Health Check** (`/api/v1/health`) para monitorização do estado da aplicação e das suas dependências críticas (base de dados e sistema de ficheiros). Este endpoint permite que sistemas externos (load balancers, K8s probes, dashboards de status) verifiquem se a API está operacional.

---

## 📁 Arquivos Alterados (5 ficheiros)

| Arquivo | Tipo | Impacto | Detalhe |
|---------|------|---------|---------|
| `internal/api/health_handler.go` | [NEW] Apresentação | 🟢 Baixo | Handler HTTP para o endpoint `/health`. |
| `internal/api/routes.go` | [MODIFY] Config | 🟢 Baixo | Registo da rota GET `/api/v1/health`. |
| `internal/service/health_service.go` | [NEW] Domínio | 🟢 Baixo | Lógica de verificação (DB Ping, Disk Write). |
| `internal/service/ports.go` | [MODIFY] Contrato | 🟡 Médio | Adição do método `Ping()` à interface `WorkspaceRepository`. |
| `internal/repository/sqlite_repo.go` | [MODIFY] Dados | 🟢 Baixo | Implementação de `Ping()` usando `sql.DB.PingContext`. |

---

## 🛠️ Detalhes Técnicos

### 1. Novo Endpoint: `GET /api/v1/health`

O endpoint retorna um status agregado HTTP 200 (OK) ou 503 (Service Unavailable) e um payload JSON detalhado:

```json
{
  "status": "ok",      // "ok", "degraded", "unavailable"
  "checks": {
    "database": "ok",
    "disk": "ok"
  },
  "timestamp": "2026-02-17T10:00:00Z"
}
```

### 2. Service Layer (`HealthService`)

O serviço `HealthService` orquestra as verificações:
- **Base de Dados:** Chama `repo.Ping(ctx)`. Se falhar, o status global torna-se `unavailable`.
- **Disco:** Verifica se é possível criar e remover um ficheiro temporário (`checkDiskWritable`). Se falhar, o status torna-se `degraded` (assumindo que a app ainda pode ler, mas não gravar logs/estados).

### 3. Repository Layer (`Ping`)

A interface `WorkspaceRepository` foi expandida para incluir o método `Ping(ctx context.Context) error`.
No `SQLiteRepository`, isto é implementado delegando para o driver SQL nativo (`r.db.PingContext(ctx)`), garantindo que a conexão à base de dados está viva.

---

## 📊 Análise de Impacto

### Riscos e Pontos de Atenção

| Risco | Severidade | Mitigação |
|-------|-----------|-----------|
| **Disk Check I/O** | 🟢 Baixo | O teste de disco e/s criar um ficheiro vazio e remove-o imediatamente. É rápido e de baixo impacto, mas executado a cada request. Em high-load pode gerar noise de I/O (considerar cache futura se necessário). |
| **Exposure** | 🟢 Baixo | O endpoint é público. Não expõe detalhes sensíveis do sistema (apenas "ok" ou erro genérico). |

---

## ✅ Checklist de Revisão

- [x] Endpoint responde 200 OK quando tudo está saudável.
- [x] Endpoint responde 503 quando o DB está em baixo (simulado).
- [x] Verificado que o ficheiro temporário de teste de disco é removido (não deixa lixo).
- [x] Interface `WorkspaceRepository` atualizada corretamente em todos os consumidores.
