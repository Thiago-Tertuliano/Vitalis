# Vitalis-gateway

**Porta única de entrada** da plataforma [Vitalis](https://github.com/Rede-Medica-D-Excelencia-Vitalis).

Todo cliente (web, mobile, webhook) fala só com o Gateway. Ele **não tem banco nem regra de negócio**: valida o JWT emitido pelo **Identity** (via **JWKS**), aplica as regras de acesso por papel, injeta `X-User-Id` / `X-Roles` e encaminha a requisição para o microserviço certo.

| | |
|---|---|
| Módulo Go | `github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway` |
| Porta (contrato) | `8080` |
| Banco | nenhum (stateless) |
| Stack | Go 1.27 · chi · httputil.ReverseProxy · golang-jwt · x/time/rate |
| Depende de | Identity (`/v1/auth/jwks.json`) |
| ADRs | ADR-001, ADR-003, ADR-004 |

> **Documentação completa (arquitetura, cada arquivo, modelo de ameaças, sequências):**  
> **[DOCUMENTACAO.md](./DOCUMENTACAO.md)**

---

## O que este serviço faz

- Roteia `/v1/*` para 11 serviços internos pela tabela de rotas (`internal/routes/table.go`)
- Valida access token **RS256** com as chaves do Identity (cache por `kid`, sem segredo compartilhado)
- Rejeita refresh token usado como access (claim `typ`)
- Autorização por papel: `/v1/admin/usuarios` e `/v1/audit` exigem `admin`
- Anti-spoofing: descarta `X-User-Id`, `X-Roles` e `X-Gateway` vindos do cliente
- Bloqueia path traversal (`/v1/auth/../admin`) e rotas fora da tabela (ex.: `/v1/internal/*` → 404)
- Rate limit por IP, CORS com allowlist, limite de tamanho do corpo
- `X-Request-Id` em toda requisição (gerado ou propagado) e log JSON de acesso
- Proxy de WebSocket (`/v1/realtime`) com token por query, removido antes de chegar ao serviço

---

## Layout

```
cmd/api                 # composition root: monta tudo e sobe o servidor
internal/
  config/               # variáveis de ambiente → Config
  routes/               # tabela prefixo → serviço + regra de acesso
  auth/                 # JWKS (cache de chaves) + validador JWT
  proxy/                # um reverse proxy por serviço interno
  gateway/              # handler: rota → auth → roles → headers → proxy
  middleware/           # request id, log de acesso, rate limit
  httpserver/           # router chi + /health /ready
  httpx/                # resposta JSON / erro padrão
  platform/logger/      # slog JSON
api/openapi.yaml        # contrato HTTP + x-routing-table
```

---

## Pré-requisitos

- Go **1.27+**
- **Identity rodando** em `http://127.0.0.1:8081` (veja o [README do Identity](../vitalis-identity/README.md))
- Docker Desktop (opcional, só para rodar o Gateway em container)

---

## Setup local

### 1. Ambiente

```powershell
cd microservices/vitalis-gateway
copy .env.example .env
```

Os defaults já apontam para o Identity local:

```env
HTTP_ADDR=:8080
JWKS_URL=http://127.0.0.1:8081/v1/auth/jwks.json
UPSTREAM_IDENTITY=http://127.0.0.1:8081
```

> No Windows, salve o `.env` em **UTF-8 sem BOM**. Com BOM o `godotenv` ignora o arquivo inteiro.

### 2. Subir

Terminal 1, o Identity (porta 8081):

```powershell
cd microservices/vitalis-identity
docker compose up postgres redis -d
go run ./cmd/api
```

Terminal 2, o Gateway (porta 8080):

```powershell
cd microservices/vitalis-gateway
go mod tidy
go run ./cmd/api
```

Se o Gateway subir antes do Identity, ele só avisa (`JWKS indisponível no boot`) e busca as chaves na primeira requisição autenticada.

Makefile (opcional): `make run`, `make test`, `make build`, `make docker-up`.

---

## Smoke test

Use `curl.exe`, e não `curl`: no PowerShell, `curl` é um alias de outro comando.

```powershell
curl.exe http://127.0.0.1:8080/health
curl.exe http://127.0.0.1:8080/ready          # 200 = já tem as chaves do Identity

# Login passando pelo Gateway (rota pública)
$login = Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8080/v1/auth/login `
  -ContentType "application/json" -Body '{"email":"ana@test.com","senha":"Senha@123"}'
$token = $login.access_token

# 200: o Gateway validou o JWT e injetou X-User-Id para o Identity
Invoke-RestMethod -Uri http://127.0.0.1:8080/v1/usuarios/me -Headers @{ Authorization = "Bearer $token" }
```

Provas de segurança:

```powershell
curl.exe -i http://127.0.0.1:8080/v1/usuarios/me                                                    # 401 sem token
curl.exe -i http://127.0.0.1:8080/v1/usuarios/me -H "X-User-Id: qualquer"                            # 401 header forjado
curl.exe -i http://127.0.0.1:8080/v1/usuarios/me -H "Authorization: Bearer $($login.refresh_token)"  # 401 refresh não é access
curl.exe -i -X PUT http://127.0.0.1:8080/v1/admin/usuarios/x/roles -H "Authorization: Bearer $token" # 403 paciente não é admin
curl.exe -i --path-as-is http://127.0.0.1:8080/v1/auth/../admin/usuarios                             # 400 path traversal
curl.exe -i http://127.0.0.1:8080/v1/internal/users -H "Authorization: Bearer $token"                # 404 rota interna
curl.exe -i http://127.0.0.1:8080/v1/pedidos -H "Authorization: Bearer $token"                       # 502 Orders ainda não existe
```

---

## Testes

```powershell
go test -cover ./...
```

| Pacote | O que cobre | Cobertura |
|---|---|---|
| `internal/gateway` | 19 cenários de status (401/403/404/400/502/503), injeção de identidade, anti-spoofing, token de WebSocket | 100% |
| `internal/routes` | casamento de prefixo e "só auth e webhook são públicos" | 100% |
| `internal/auth` | cache do JWKS, kid forjado não vira flood, cache com Identity fora | 75% |
| `internal/middleware` | rate limit por IP (e imune a `X-Forwarded-For`), request id | 58% |

No CI (`.github/workflows/quality.yml`, na raiz do repositório) ainda rodam `golangci-lint` (com `staticcheck` e `gosec`), `go test -race`, `govulncheck`, Trivy, Semgrep e Gitleaks.

---

## Rotas

| Prefixo | Serviço | Acesso |
|---|---|---|
| `/v1/auth/*` | identity:8081 | público |
| `/v1/webhooks/pagamentos` | billing:8086 | público (Billing valida HMAC) |
| `/v1/usuarios/*` | identity:8081 | autenticado |
| `/v1/admin/usuarios/*` | identity:8081 | `admin` |
| `/v1/audit/*` | audit:8091 | `admin` |
| `/v1/payment-intents`, `/subscriptions`, `/wallets`, `/invoices`, `/settlements`, `/commissions` | billing:8086 | autenticado |
| `/v1/especialidades`, `/medicos`, `/disponibilidade`, `/consultas`, `/triagens`, `/videochamadas`, `/prescricoes` | clinical:8082 | autenticado |
| `/v1/farmacias`, `/categorias`, `/produtos`, `/estoque` | commerce:8083 | autenticado |
| `/v1/pedidos` | orders:8084 | autenticado |
| `/v1/entregas`, `/motoboys` | delivery:8085 | autenticado |
| `/v1/notificacoes`, `/chat`, `/realtime` (WS) | comms:8087 | autenticado |
| `/v1/tickets`, `/artigos-ajuda` | support:8088 | autenticado |
| `/v1/objetos`, `/pdf` | files:8089 | autenticado |
| `/v1/search` | search:8090 | autenticado |

Fora da tabela → **404**. Contrato: [`api/openapi.yaml`](./api/openapi.yaml).

---

## Segurança (resumo)

- Só **RS256** é aceito (bloqueia troca de `alg` para `HS256`/`none`); `iss`, `aud`, `exp` e `typ=access` obrigatórios
- Chave privada **nunca** sai do Identity; o Gateway só conhece as públicas (JWKS)
- Headers de identidade são **sempre** reescritos pelo Gateway
- Rate limit usa o IP da conexão, não `X-Forwarded-For` (que o cliente pode forjar)
- Token nunca é logado; token de WebSocket sai da query antes do proxy
- Os serviços internos (8081–8091) **não** devem ser expostos fora da rede interna

---

## Referências Vitalis

- [DOCUMENTACAO.md](./DOCUMENTACAO.md) — guia completo deste serviço
- `docs/ARQUITETURA_MICROSERVICOS_VITALIS.md`
- `docs/adrs/ADR-004-auth-rbac.md`
- `docs/RBAC.md`
- `docs/contracts/openapi/gateway.yaml`
- `docs/diagramas/servicos/gateway.md`

---

## Próximo serviço

**Vitalis-billing** — pagamentos, assinaturas e carteiras; recebe o webhook `/v1/webhooks/pagamentos` que o Gateway já encaminha.
