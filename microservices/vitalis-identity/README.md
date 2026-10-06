# Vitalis-identity

Microserviço de **identidade e autenticação** da plataforma [Vitalis](https://github.com/Rede-Medica-D-Excelencia-Vitalis).

É a **fonte da verdade** de usuários, papéis, senhas e tokens. Os demais serviços **não** validam senha: o **Gateway** valida o JWT via **JWKS** e injeta `X-User-Id` / `X-Roles`.

| | |
|---|---|
| Módulo Go | `github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity` |
| Porta (contrato) | `8081` |
| Banco | PostgreSQL `pg_identity` (porta host **5433** no Compose) |
| Stack | Go 1.27 · chi · pgx · goose · Redis · Argon2id · JWT RS256 |
| ADRs | ADR-001 … ADR-004 |

> **Documentação completa (arquitetura, cada arquivo, fluxos, ER):**  
> **[DOCUMENTACAO.md](./DOCUMENTACAO.md)**

---

## O que este serviço faz

- Cadastro e login (`/v1/auth/registro`, `/v1/auth/login`)
- Refresh com **rotação** de `jti` e logout
- Emissão de access JWT **RS256** + `GET /v1/auth/jwks.json`
- Perfil e endereços (`/v1/usuarios/me…`) via headers do Gateway
- Admin de papéis (`paciente` \| `medico` \| `farmacia` \| `motoboy` \| `admin`)
- Reset de senha (MVP)
- Eventos via **outbox** → Redis Streams (`user.registered`, `user.updated`, `user.role_changed`)

---

## Layout

```
cmd/api                 # HTTP API (composition root)
cmd/worker              # publica outbox → Redis Streams
internal/
  domain/               # User, Address + regras
  port/                 # interfaces hexagonais
  app/                  # casos de uso
  adapter/http|postgres|redis|jwt|crypto
  outbox/               # writer + publisher
  config/  platform/logger/
migrations/             # goose
api/openapi.yaml        # contrato HTTP
secrets/                # PEM JWT (não versionar a privada)
```

---

## Pré-requisitos

- Go **1.27+**
- Docker Desktop
- [goose](https://github.com/pressly/goose): `go install github.com/pressly/goose/v3/cmd/goose@latest`
- OpenSSL (no Windows: `C:\Program Files\Git\usr\bin\openssl.exe`)

Garanta `%\USERPROFILE%\go\bin` no `PATH`.

---

## Setup local

### 1. Ambiente

```powershell
cd microservices/vitalis-identity
copy .env.example .env
```

Ajuste no `.env` (importante no Windows se já existir Postgres na 5432):

```env
HTTP_ADDR=:8081
DATABASE_URL=postgres://vitalis:vitalis@127.0.0.1:5433/pg_identity?sslmode=disable
```

### 2. Chaves JWT (uma vez)

```powershell
mkdir secrets -Force
& "C:\Program Files\Git\usr\bin\openssl.exe" genrsa -out secrets/jwt_private.pem 2048
& "C:\Program Files\Git\usr\bin\openssl.exe" rsa -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem
```

Não commitar `secrets/jwt_private.pem` nem `.env`.

### 3. Infra + migration + API

```powershell
docker compose up postgres redis -d

$env:Path = "$env:USERPROFILE\go\bin;$env:Path"
$env:DATABASE_URL = "postgres://vitalis:vitalis@127.0.0.1:5433/pg_identity?sslmode=disable"
goose -dir ./migrations postgres $env:DATABASE_URL up

go mod tidy
go run ./cmd/api
```

Worker (outro terminal, eventos):

```powershell
go run ./cmd/worker
```

Makefile (opcional): `make docker-up`, `make migrate-up`, `make run-api`, `make run-worker`.

---

## Smoke test

```powershell
curl http://127.0.0.1:8081/health

curl -X POST http://127.0.0.1:8081/v1/auth/registro `
  -H "Content-Type: application/json" `
  -d "{\"nome\":\"Ana\",\"email\":\"ana@test.com\",\"senha\":\"Senha@123\",\"tipo_usuario\":\"paciente\"}"

curl -X POST http://127.0.0.1:8081/v1/auth/login `
  -H "Content-Type: application/json" `
  -d "{\"email\":\"ana@test.com\",\"senha\":\"Senha@123\"}"

curl http://127.0.0.1:8081/v1/auth/jwks.json

# simula Gateway
curl http://127.0.0.1:8081/v1/usuarios/me -H "X-User-Id: <ID_DO_USER>"
```

---

## Endpoints principais

| Método | Path | Auth |
|---|---|---|
| GET | `/health`, `/ready` | público |
| POST | `/v1/auth/registro`, `/login`, `/refresh`, `/logout` | público / body |
| GET | `/v1/auth/jwks.json` | público |
| GET/PUT | `/v1/usuarios/me` | `X-User-Id` |
| CRUD | `/v1/usuarios/me/enderecos` | `X-User-Id` |
| PUT | `/v1/admin/usuarios/{id}/roles` | `X-Roles: admin` |

Contrato: [`api/openapi.yaml`](./api/openapi.yaml).

---

## Segurança (resumo)

- Senha em `credentials` com **argon2id** (nunca na tabela `users`)
- JWT **RS256** — privada só no Identity; Gateway usa JWKS
- Refresh com **jti** no banco (revogação / rotação)
- Headers `X-User-Id` / `X-Roles` só na rede interna (após o Gateway)

---

## Referências Vitalis

- [DOCUMENTACAO.md](./DOCUMENTACAO.md) — guia completo deste serviço
- `docs/ARQUITETURA_MICROSERVICOS_VITALIS.md`
- `docs/adrs/` (ADR-001 … ADR-004)
- `docs/RBAC.md`
- `docs/contracts/openapi/identity.yaml`
- `docs/contracts/events/CATALOGO_EVENTOS.md`

---

## Próximo serviço

**Vitalis-gateway** — valida Bearer via JWKS deste Identity e roteia para os demais microserviços.
