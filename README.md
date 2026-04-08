# golang-boilerplate-api

> **Boilerplate** de API REST em Go para backends SaaS com autenticacao, planos/assinatura, pagamento e observabilidade completa.

---

## Stack

| Camada | Tecnologia |
|---|---|
| HTTP | [Fiber v2](https://github.com/gofiber/fiber) |
| DI / Lifecycle | [Uber fx](https://github.com/uber-go/fx) |
| ORM | [GORM](https://gorm.io) + PostgreSQL |
| Auth | [golang-jwt/v5](https://github.com/golang-jwt/jwt) + bcrypt |
| Cache | [go-redis/v9](https://github.com/redis/go-redis) |
| Migrations | [Flyway](https://flywaydb.org) |
| Logging | [Uber zap](https://github.com/uber-go/zap) |
| Tracing | OpenTelemetry → Tempo |
| Metricas | OpenTelemetry → Prometheus |
| Logs aggregation | OpenTelemetry → Loki |
| Dashboards | Grafana |
| Testes de integracao | [testcontainers-go](https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/) |
| Git hooks | [Lefthook](https://github.com/evilmartians/lefthook) |

---

## Arquitetura

**Clean Architecture** organizada por modulos de negocio. Cada modulo contem suas proprias camadas sem cruzar limites de dominio.

```
internal/
├── bootstrap/          # Composicao do app Fiber + wiring fx (root)
├── config/             # Configuracao via variaveis de ambiente
├── shared/             # Infraestrutura e abstracoes reutilizaveis
│   ├── domain/
│   │   ├── exceptions/ # DomainError + construtores tipados
│   │   ├── providers/  # Interfaces: LoggerProvider, CacheProvider, EmailProvider
│   │   └── repositories/ # GenericRepository[T, ID]
│   └── infra/
│       ├── http/middleware/  # ErrorHandler, RequestID, HTTPMetrics
│       ├── persistence/      # GORM + Redis connections
│       ├── providers/        # Logger (Zap), Cache (Redis), Email (SMTP)
│       └── telemetry/        # OpenTelemetry setup
├── modules/
│   ├── auth/           # Autenticacao (JWT, refresh tokens, middleware)
│   ├── health/         # Health checks (liveness + readiness)
│   ├── security/       # Monitoramento de seguranca (suspicious activities, blocks)
│   └── users/          # Gestao de usuarios
└── test/
    └── integration/    # Testes e2e com PostgreSQL + Redis via testcontainers
```

### Fluxo de uma requisicao

```
HTTP Request
  └── Fiber (CORS → OTel Span → HTTP Metrics → Request ID)
        └── Auth Middleware (JWT header OU cookie → valida → Redis blacklist check)
              └── Controller      (valida input, chama use case)
                    └── UseCase   (regras de negocio, abre span filho)
                          └── Repository  (GORM + span de DB via gorm-otel)
```

---

## Modulos implementados

### Auth (`/auth`)

| Metodo | Path | Descricao |
|---|---|---|
| `POST` | `/auth/register` | Registro com bcrypt hash |
| `POST` | `/auth/login` | Login retorna JWT access + refresh token |
| `POST` | `/auth/refresh` | Rotacao de refresh token com theft detection |
| `POST` | `/auth/logout` | Blacklist do token + revoga refresh tokens |

**Refresh tokens** armazenados no PostgreSQL com:
- `family_id` para deteccao de roubo (family-based rotation)
- `device_id`, `user_agent`, `ip_address` para rastreamento
- `token_hash` (SHA-256) — JWT raw nunca e armazenado

**Auth middleware** dual-mode:
- `Authorization: Bearer <token>` (mobile/API)
- Cookie `access_token` httpOnly (web)

### Health (`/healthz`, `/readyz`)

| Metodo | Path | Descricao |
|---|---|---|
| `GET` | `/healthz` | Liveness — sempre 200 |
| `GET` | `/readyz` | Readiness — verifica PostgreSQL + Redis |

### Users (`/api/users`)

| Metodo | Path | Descricao |
|---|---|---|
| `POST` | `/api/users` | Cria usuario |
| `GET` | `/api/users/:id` | Busca usuario por ID |

---

## Database Schema

6 migrations Flyway:

| Migration | Tabela | Descricao |
|---|---|---|
| V1 | `users` | Usuarios (bigserial, password, img_url, admin, active, source, metadata) |
| V2 | `refresh_tokens` | Refresh tokens com family rotation e device tracking |
| V3 | `suspicious_activities`, `user_security_blocks` | Monitoramento de seguranca |
| V4 | `email_verification_tokens` | Tokens de confirmacao de email |
| V5 | `password_reset_tokens` | Tokens de recuperacao de senha |
| V6 | `plans`, `subscriptions`, `payment_events` | Planos, assinaturas e eventos de pagamento |

---

## Pre-requisitos

- **Go 1.21+**
- **Docker** e **Docker Compose**
- _(opcional)_ [Lefthook](https://github.com/evilmartians/lefthook) para git hooks

---

## Configuracao

```bash
cp .env.example .env
```

| Variavel | Padrao | Descricao |
|---|---|---|
| `SERVICE_NAME` | `boilerplate-api` | Nome do servico nos traces/logs |
| `PORT` | `3000` | Porta HTTP |
| `APP_ENV` | `development` | Ambiente |
| `DATABASE_URL` | — | Connection string PostgreSQL |
| `REDIS_URL` | `redis://localhost:6379/0` | Connection string Redis |
| `JWT_ACCESS_SECRET` | — | Chave HMAC para access tokens |
| `JWT_REFRESH_SECRET` | — | Chave HMAC para refresh tokens |
| `SMTP_HOST` | `localhost` | Host SMTP para emails |
| `SMTP_PORT` | `1025` | Porta SMTP |
| `EMAIL_FROM` | `noreply@example.com` | Remetente padrao |

---

## Rodando localmente

### Stack completa (recomendado)

```bash
docker compose up
```

| Servico | URL |
|---|---|
| API | http://localhost:3000 |
| Grafana | http://localhost:3001 |
| MailHog | http://localhost:8025 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |

### Apenas a aplicacao

```bash
cp .env.example .env
make migrate
make run
```

---

## Comandos Make

```bash
make run              # Inicia a API
make build            # Compila para bin/api
make tidy             # go mod tidy

make test/unit        # Testes unitarios (sem Docker)
make test/integration # Testes de integracao (testcontainers)
make test             # Ambos

make migrate          # Aplica migrations via Flyway
make migrate-info     # Status das migrations
```

---

## Testes

### Unitarios

```bash
make test/unit
```

Testam logica de negocio isolada com mocks manuais. Sem Docker.

### Integracao

```bash
make test/integration
```

Sobem containers reais (PostgreSQL 17 + Redis 7 + MailHog) via testcontainers. Exercitam endpoints HTTP com `fiber.Test`.

---

## Git Hooks (Lefthook)

```bash
go install github.com/evilmartians/lefthook@latest
lefthook install
```

| Hook | Acao |
|---|---|
| `commit-msg` | Valida Conventional Commits + emoji |
| `pre-commit` | `gofmt`, `go vet`, `golangci-lint --fix` |
| `pre-push` | Valida nome do branch |

---

## CI/CD (GitHub Actions)

Workflow `.github/workflows/combined-analysis.yml` em PRs para `develop` e `main`:

- CodeQL (seguranca)
- golangci-lint
- Testes unitarios + integracao
- govulncheck

---

## Roadmap

Progresso atual do boilerplate:

| # | Phase | Status |
|---|-------|--------|
| 1 | Redis + Cache | ✓ |
| 2 | Email Service (adapter pattern) | ✓ |
| 3 | User Entity + Migrations | ✓ |
| 4 | Auth Foundation (JWT + family rotation) | ✓ |
| 5 | Account Lifecycle (confirm email, reset password, profile) | ○ |
| 6 | Security Monitoring | ○ |
| 7 | Plans + Subscriptions + Payments (Stripe) | ○ |
| 8 | Social Login (Google + Apple) | ○ |
