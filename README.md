# golang-boilerplate-api

> **Boilerplate** de API REST em Go com arquitetura modular, observabilidade completa (tracing · métricas · logs) e testes de integração baseados em containers.

---

## Stack

| Camada | Tecnologia |
|---|---|
| HTTP | [Fiber v2](https://github.com/gofiber/fiber) |
| DI / Lifecycle | [Uber fx](https://github.com/uber-go/fx) |
| ORM | [GORM](https://gorm.io) + PostgreSQL |
| Migrations | [Flyway](https://flywaydb.org) |
| Logging | [Uber zap](https://github.com/uber-go/zap) |
| Tracing | OpenTelemetry → Tempo |
| Métricas | OpenTelemetry → Prometheus |
| Logs aggregation | OpenTelemetry → Loki |
| Dashboards | Grafana |
| Testes de integração | [testcontainers-go](https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/) |
| Git hooks | [Lefthook](https://github.com/evilmartians/lefthook) |

---

## Arquitetura

O projeto segue **Clean Architecture** organizada por módulos de negócio. Cada módulo contém suas próprias camadas sem cruzar limites de domínio.

```
internal/
├── bootstrap/          # Composição do app Fiber + wiring fx (root)
├── config/             # Configuração via variáveis de ambiente
├── shared/             # Infraestrutura e abstrações reutilizáveis
│   ├── domain/
│   │   ├── exceptions/ # DomainError + construtores tipados
│   │   ├── providers/  # Interface LoggerProvider
│   │   └── repositories/ # GenericRepository[T, ID]
│   └── infra/
│       ├── http/middleware/  # ErrorHandler, RequestID, HTTPMetrics
│       ├── observability/    # Helpers de span (RecordError, LoggerWithTrace)
│       ├── persistence/      # Conexão GORM + GormGenericRepository
│       ├── providers/logger/ # ZapLoggerProvider
│       └── telemetry/        # Setup OpenTelemetry (tracer, meter, logger)
├── modules/
│   ├── health/
│   │   ├── application/usecases/  # CheckHealthUseCase, CheckReadinessUseCase
│   │   ├── domain/                # HealthStatus, HealthRepository interface
│   │   └── infra/
│   │       ├── http/              # HealthController, routes
│   │       └── persistence/       # GormHealthRepository
│   └── users/
│       ├── application/usecases/  # CreateUserUseCase, GetUserUseCase
│       ├── domain/                # User entity, UserRepository interface
│       └── infra/
│           ├── http/              # UserController, routes
│           └── persistence/       # GormUserRepository
└── test/
    └── integration/               # Testes e2e com PostgreSQL via testcontainers
```

### Fluxo de uma requisição

```
HTTP Request
  └── Fiber (CORS → OTel Span → HTTP Metrics → Request ID)
        └── Controller      (valida input, chama use case)
              └── UseCase   (regras de negócio, abre span filho)
                    └── Repository  (GORM + span de DB via gorm-otel)
```

---

## Pré-requisitos

- **Go 1.21+**
- **Docker** e **Docker Compose**
- _(opcional)_ [Lefthook](https://github.com/evilmartians/lefthook) para git hooks

---

## Configuração

Copie o arquivo de exemplo e ajuste conforme necessário:

```bash
cp .env.example .env
```

| Variável | Padrão | Descrição |
|---|---|---|
| `SERVICE_NAME` | `boilerplate-api` | Nome do serviço nos traces/logs |
| `PORT` | `3000` | Porta HTTP |
| `APP_ENV` | `development` | Ambiente (`development`, `production`, `test`) |
| `LOG_LEVEL` | `debug` | Nível de log (`debug`, `info`, `warn`, `error`) |
| `DATABASE_URL` | — | Connection string PostgreSQL |
| `DATABASE_MAX_CONNECTIONS` | `10` | Pool máximo de conexões |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | Endpoint OTLP HTTP (vazio = desativado) |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | Protocolo OTLP |

---

## Rodando localmente

### Apenas a aplicação (requer PostgreSQL externo)

```bash
# 1. Configure o .env com DATABASE_URL apontando para seu PostgreSQL
cp .env.example .env

# 2. Aplique as migrations
make migrate

# 3. Starte a API
make run
```

### Stack completa com observabilidade (recomendado)

```bash
docker compose up
```

Sobe todos os serviços:

| Serviço | URL |
|---|---|
| API | http://localhost:3000 |
| Grafana | http://localhost:3001 |
| PostgreSQL | localhost:5432 |
| OTLP HTTP | http://localhost:4318 |

---

## Endpoints

### Health

| Método | Path | Descrição |
|---|---|---|
| `GET` | `/healthz` | Liveness — responde `200 { "status": "healthy" }` sempre |
| `GET` | `/readyz` | Readiness — verifica conexão com o banco |

```jsonc
// GET /readyz — exemplo de resposta
{
  "status": "healthy",
  "components": {
    "database": { "status": "healthy" }
  }
}
```

### Users

| Método | Path | Descrição |
|---|---|---|
| `POST` | `/api/users` | Cria um novo usuário |
| `GET` | `/api/users/:id` | Busca usuário por ID |

```jsonc
// POST /api/users
{ "name": "João Silva", "email": "joao@example.com" }

// 201 Created
{ "id": 1, "name": "João Silva", "email": "joao@example.com" }
```

**Códigos de erro:**

| Código HTTP | Quando |
|---|---|
| `400` | Body malformado ou campos obrigatórios ausentes |
| `404` | Usuário não encontrado |
| `422` | E-mail já cadastrado |
| `503` | Banco indisponível (apenas `/readyz`) |

---

## Comandos Make

```bash
make run              # Inicia a API (carrega .env automaticamente)
make build            # Compila para bin/api
make tidy             # Sincroniza go.mod e go.sum

make test/unit        # Testes unitários dos use cases (sem Docker, rápidos)
make test/integration # Testes de integração com PostgreSQL via testcontainers
make test             # Executa unit → integration em sequência

make migrate          # Aplica as migrations pendentes via Flyway
make migrate-info     # Exibe o status das migrations
```

---

## Testes

### Unitários (use cases)

Testam a lógica de negócio isolada através de mocks manuais. **Não precisam de Docker.**

```bash
make test/unit
```

Cobre:

- `CreateUserUseCase` — sucesso, campos ausentes, e-mail duplicado, erro de repositório
- `GetUserUseCase` — sucesso, not found, erro de repositório
- `CheckHealthUseCase` — sempre retorna `healthy`
- `CheckReadinessUseCase` — banco saudável, banco unhealthy, ping retorna `false`

### Integração (end-to-end)

Sobem um container **PostgreSQL 17** real via testcontainers, aplicam as migrations inline e exercitam os endpoints HTTP usando `fiber.Test` (sem abrir porta de rede).

```bash
make test/integration
```

Cobre:

- `GET /healthz`, `GET /readyz` — liveness e readiness
- `X-Request-ID` — propagação e geração automática
- `POST /api/users` — sucesso, e-mail duplicado, campos ausentes
- `GET /api/users/:id` — sucesso, not found, ID inválido

---

## Observabilidade

A stack é provisionada automaticamente pelo `docker compose up`. Todos os sinais convergem no **OpenTelemetry Collector** antes de serem roteados.

```
API (OTLP HTTP :4318)
  └── OTel Collector
        ├── Traces  → Tempo  (visualização: Grafana → Explore → Tempo)
        ├── Métricas → Prometheus (visualização: Grafana → Explore → Prometheus)
        └── Logs   → Loki   (visualização: Grafana → Explore → Loki)
```

**Configurações em** `monitoring/`:

| Arquivo | Propósito |
|---|---|
| `otel-collector-config.yaml` | Pipeline receivers / processors / exporters |
| `tempo-config.yaml` | Armazenamento de traces |
| `loki-config.yaml` | Armazenamento de logs |
| `prometheus.yaml` | Scrape configs |
| `grafana-datasources.yaml` | Provisionamento automático das datasources |

---

## Git Hooks (Lefthook)

Instale o Lefthook e ative os hooks:

```bash
# Instalar Lefthook (uma vez)
go install github.com/evilmartians/lefthook@latest

# Ativar os hooks no repositório
lefthook install
```

| Hook | Ação |
|---|---|
| `commit-msg` | Valida formato Conventional Commits e adiciona emoji |
| `pre-commit` | `gofmt`, `go vet`, `golangci-lint --fix` |
| `pre-push` | Valida nome do branch (`feature/`, `fix/`, `hotfix/`, `docs/`, `refactor/`, `test/`, `build/`) |

### Conventional Commits

```
feat(users): add create user endpoint   →  ✨ feat(users): ...
fix(health): correct readiness check    →  🐛 fix(health): ...
chore: update dependencies              →  🔧 chore: ...
```

---

## CI/CD (GitHub Actions)

O workflow `.github/workflows/combined-analysis.yml` é ativado em Pull Requests para `develop` e `main`.

```
detect-changes
├── CodeQL          (análise estática de segurança)
├── Lint            (golangci-lint)
├── Test-Unit       (use cases, sem Docker) ──→ Test-Integration (testcontainers)
└── Security        (govulncheck)
```

O **Dependabot** (`weekly`) monitora atualizações em `go.mod`, `Dockerfile` e GitHub Actions.

---

## Migrations

As migrations ficam em `migrations/` no formato Flyway (`V1__description.sql`).

```bash
# Aplicar migrations (requer PostgreSQL rodando)
make migrate

# Ver status
make migrate-info
```

No `docker compose up`, o Flyway roda automaticamente antes da API subir.

---

## Estrutura de arquivos raiz

```
.
├── cmd/api/main.go          # Entrypoint
├── internal/                # Todo o código da aplicação
├── migrations/              # Scripts SQL (Flyway)
├── monitoring/              # Configs OTel Collector, Tempo, Loki, Prometheus, Grafana
├── Dockerfile               # Multi-stage build (builder + alpine runtime)
├── docker-compose.yaml      # Stack completa com observabilidade
├── Makefile                 # Comandos de desenvolvimento
├── lefthook.yml             # Git hooks
├── go.mod / go.sum          # Dependências Go
└── .env.example             # Template de variáveis de ambiente
```