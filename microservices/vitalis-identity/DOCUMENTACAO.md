# Vitalis-identity — Documentação completa

> Microserviço de **identidade e autenticação** da plataforma Vitalis.  
> Este documento explica o **porquê** da arquitetura, o papel de cada pasta/arquivo e como o fluxo funciona de ponta a ponta.  
> Útil para live, TCC, onboarding e revisão de código.


| Campo                 | Valor                                                                       |
| --------------------- | --------------------------------------------------------------------------- |
| Repo / módulo Go      | `github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-identity`              |
| Porta HTTP (contrato) | `8081` (local pode estar em `8080` se `HTTP_ADDR` não for carregado)        |
| Banco                 | PostgreSQL `pg_identity` (database-per-service)                             |
| Stack                 | Go 1.27 · chi · pgx · goose · Redis · Argon2id · JWT RS256                  |
| ADRs                  | ADR-001 (polyrepo), ADR-002 (stack), ADR-003 (eventos), ADR-004 (auth/RBAC) |


---



## 1. Por que o Identity existe?

Na Vitalis em microserviços, **nenhum outro serviço deve “conhecer senha” nem emitir JWT**.

O **Identity** é a **fonte da verdade** de:

- cadastro de usuários
- autenticação (login / refresh / logout)
- papéis (`paciente`, `medico`, `farmacia`, `motoboy`, `admin`)
- perfil e endereços do usuário
- emissão de tokens **RS256** e publicação do **JWKS** (chave pública)

Os demais serviços (Clinical, Billing, Orders…) **não validam senha**. Em produção, o **Gateway**:

1. recebe o `Authorization: Bearer …`
2. valida a assinatura com o JWKS do Identity
3. injeta headers internos: `X-User-Id`, `X-Roles`, `X-Request-Id`
4. encaminha a requisição ao serviço de destino

Assim o Identity fica isolado, o Gateway é a porta pública, e cada bounded context permanece enxuto.

### Por que foi o primeiro serviço?

Sem Identity não há JWT confiável. Sem JWT o Gateway não autenticará o restante. A ordem natural é:

**Identity → Gateway → Billing → Clinical / Commerce / Orders…**

---



## 2. Princípios de construção



### 2.1 Hexagonal (ports & adapters)

```
                    ┌─────────────┐
   HTTP (chi)  ───► │    app/     │ ───► port.UserRepository
   JWT / Argon2 ──► │  casos de   │ ───► port.TokenIssuer
   Postgres    ◄─── │    uso      │ ───► port.PasswordHasher
   Outbox      ◄─── │             │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   domain/   │  (regras puras, sem framework)
                    └─────────────┘
```


| Camada      | Pasta                   | Responsabilidade                                      |
| ----------- | ----------------------- | ----------------------------------------------------- |
| Domínio     | `internal/domain`       | Entidades e regras (`PodeReceberToken`, `RoleValida`) |
| Portas      | `internal/port`         | Interfaces (contratos)                                |
| Aplicação   | `internal/app`          | Casos de uso (orquestração)                           |
| Adaptadores | `internal/adapter/*`    | HTTP, Postgres, Redis, JWT, Argon2                    |
| Entrypoints | `cmd/api`, `cmd/worker` | Composition root + processo                           |


**Por quê?** Trocar Postgres, chi ou a lib JWT não deve espalhar mudança no domínio. Testes de caso de uso podem mockar portas.

### 2.2 Database-per-service

Só o Identity acessa `pg_identity`. Outros serviços **não** fazem JOIN em `users`. Precisam de dado de usuário? Consomem evento (`user.registered`) ou chamam API via Gateway/rede interna.

### 2.3 Segurança alinhada ao ADR-004

- Senha **nunca** na tabela `users` → tabela `credentials` com **argon2id**
- JWT **RS256**: chave **privada** só no Identity; **JWKS** público para o Gateway
- Access token curto; refresh com **jti** persistido (revogação / rotação)
- Mensagens de login genéricas (não vazar se o e-mail existe)
- Endpoints `/me` confiam em `X-User-Id` injetado pelo Gateway (rede interna)



### 2.4 Outbox + Redis Streams (ADR-003)

Quando um usuário se registra ou muda role, o Identity grava um evento em `outbox_events` **no mesmo banco**. O `cmd/worker` publica no Redis Stream. Assim o evento não se perde se o Redis estiver fora no momento do commit.

---



## 3. Mapa da árvore do projeto

```
vitalis-identity/
├── cmd/
│   ├── api/main.go              # API HTTP — composition root
│   └── worker/main.go           # Publica outbox → Redis Streams
├── internal/
│   ├── config/                  # Env → struct Config
│   ├── domain/                  # User, Address + regras
│   ├── port/                    # Interfaces (hexagonal)
│   ├── app/                     # Casos de uso
│   ├── adapter/
│   │   ├── http/                # Router chi + middleware
│   │   ├── postgres/            # Repositórios + pool
│   │   ├── redis/               # Cliente Redis
│   │   ├── crypto/              # Argon2id
│   │   └── jwt/                 # Emissor RS256 + JWKS
│   ├── outbox/                  # Writer + Publisher
│   └── platform/logger/         # slog JSON
├── migrations/                  # goose SQL
├── api/openapi.yaml             # Contrato HTTP
├── secrets/                     # PEM JWT (NÃO commitad)
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
├── DOCUMENTACAO.md              # Este arquivo
└── README.md
```

---



## 4. Processos: API vs Worker



### 4.1 `cmd/api/main.go` — Composition root

É o **único lugar** que “conhece tudo”. Ordem mental:

1. Carrega `.env` (`godotenv`) + `config.Load()`
2. Abre pool Postgres e cliente Redis
3. Instancia adapters (`UserRepo`, `Argon2Hasher`, `jwt.Issuer`, …)
4. Instancia casos de uso (`AuthService`, `ProfileService`, …)
5. Monta o router chi e sobe `http.Server`
6. No `SIGINT`/`SIGTERM`, faz shutdown gracioso

**Por quê no** `main` **e não no router?** O router só recebe dependências já prontas (`Dependencies`). Isso deixa o HTTP testável e sem acoplamento a Postgres.

### 4.2 `cmd/worker/main.go`

Loop periódico que:

1. Lê `outbox_events` com `published_at IS NULL`
2. Publica envelope JSON no Redis Stream (`EVENTS_STREAM`)
3. Marca `published_at`

**Por quê processo separado?** Falha/redeploy do worker não derruba a API de login. Escala de publicação é independente.

---



## 5. Camada a camada — arquivos e motivações



### 5.1 `internal/config/config.go`

Centraliza variáveis de ambiente: porta, `DATABASE_URL`, Redis, caminhos das chaves JWT, TTLs.

**Por quê?** Evita `os.Getenv` espalhado. Falha rápido se `DATABASE_URL` faltar.

Variáveis relevantes:


| Env                                            | Uso                                                              |
| ---------------------------------------------- | ---------------------------------------------------------------- |
| `HTTP_ADDR`                                    | Ex.: `:8081`                                                     |
| `DATABASE_URL`                                 | Postgres (`127.0.0.1:5433` se a 5432 estiver ocupada no Windows) |
| `JWT_PRIVATE_KEY_PATH` / `JWT_PUBLIC_KEY_PATH` | PEMs em `./secrets/`                                             |
| `JWT_ISSUER` / `JWT_AUDIENCE`                  | Claims padrão                                                    |
| `ACCESS_TTL_MIN` / `REFRESH_TTL_DAYS`          | TTL dos tokens (defaults no código se nomes divergirem)          |


> **Nota local Windows:** Postgres nativo costuma ocupar `5432`. O Compose do Identity mapeia `5433:5432` para evitar conflito de autenticação.

---



### 5.2 `internal/domain/`



#### `user.go`

- Aggregate `User` (id, email, nome, status, roles, …)
- `PodeReceberToken()` — só `active` recebe JWT
- `RoleValida()` — enum fechado alinhado ao RBAC

**Por quê no domínio?** Regra de negócio pura: não depende de HTTP nem SQL. Dá para testar sem banco.

#### `address.go`

Endereço de entrega/residência do usuário (CEP, logradouro, UF…). Pertence ao Identity porque é dado cadastral do usuário, não do pedido clínico.

---



### 5.3 `internal/port/ports.go`

Interfaces:


| Porta                     | Implementação típica                         |
| ------------------------- | -------------------------------------------- |
| `UserRepository`          | `adapter/postgres/user_repo.go`              |
| `RefreshTokenRepository`  | `refresh_repo.go`                            |
| `AddressRepository`       | `address_repo.go`                            |
| `PasswordResetRepository` | `reset_repo.go`                              |
| `PasswordHasher`          | `adapter/crypto/argon2.go`                   |
| `TokenIssuer`             | `adapter/jwt/issuer.go`                      |
| `Clock`                   | `RealClock` (facilita testes com tempo fixo) |


**Por quê?** O `app` depende de abstração. Na live: “invertemos a dependência”.

---



### 5.4 `internal/app/` — casos de uso


| Arquivo             | Responsabilidade                                                                                         |
| ------------------- | -------------------------------------------------------------------------------------------------------- |
| `errors.go`         | Erros de domínio de aplicação (`ErrCredenciaisInvalidas`, `ErrEmailJaExiste`, …) mapeados depois no HTTP |
| `auth.go`           | Register, Login, Refresh (com rotação), Logout, JWKS                                                     |
| `password_reset.go` | Esqueci / redefinir senha                                                                                |
| `profile.go`        | GET/PUT perfil + outbox `user.updated`                                                                   |
| `address.go`        | CRUD de endereços do usuário autenticado                                                                 |
| `roles.go`          | Admin altera papéis + outbox `user.role_changed`                                                         |




#### Fluxo Register (resumo)

1. Normaliza e-mail, valida role
2. `EmailExists` → 409 se duplicado
3. `hasher.Hash(senha)`
4. `users.Create` (TX: user + credentials + roles)
5. Outbox `user.registered`
6. Retorna DTO (sem senha)



#### Fluxo Login (resumo)

1. `FindByEmail` + hash
2. `Compare` — se falhar, **sempre** `ErrCredenciaisInvalidas`
3. `PodeReceberToken`
4. `IssueAccess` + `IssueRefresh` (jti no banco)



#### Fluxo Refresh (resumo)

1. Parse JWT refresh → `jti` + `userID`
2. `FindValid(jti)` no banco
3. **Revoga** jti antigo (rotação)
4. Emite novo access + novo refresh

**Por quê rotação?** Se um refresh vazar, o uso posterior do token antigo falha após a próxima renovação legítima.

---



### 5.5 Adaptadores de infraestrutura



#### `adapter/crypto/argon2.go`

Implementa `PasswordHasher` com **argon2id**, salt aleatório, formato PHC, comparação em **tempo constante** (`subtle.ConstantTimeCompare`).

**Por quê argon2id?** Resistente a GPU/ASIC melhor que MD5/SHA soltos; padrão moderno para senhas.

#### `adapter/jwt/issuer.go`

- Carrega PEM RSA  
- Emite access com claims `uid`, `email`, `roles`, `iss`, `aud`, `exp`  
- Emite refresh com `jti` persistido  
- Expõe `PublicJWKS()` para `GET /v1/auth/jwks.json`

**Por quê RS256 e não HS256?** Com 12 serviços, compartilhar um segredo HMAC é frágil. Com RS256 só o Identity assina; todos validam com a pública.

#### `adapter/postgres/`


| Arquivo           | Papel                                                  |
| ----------------- | ------------------------------------------------------ |
| `pool.go`         | Pool pgx + ping na conexão                             |
| `user_repo.go`    | Create em **transação**; find com roles; update perfil |
| `refresh_repo.go` | Save / FindValid / Revoke jti                          |
| `address_repo.go` | CRUD endereços com ownership (`user_id`)               |
| `reset_repo.go`   | Token de reset + update hash                           |


**Create em TX — por quê?** User sem credential (ou sem role) seria estado inconsistente. Tudo ou nada.

#### `adapter/redis/client.go`

Cliente Redis usado no `/ready` e pelo worker (streams).

#### `adapter/http/`


| Arquivo                    | Papel                                                 |
| -------------------------- | ----------------------------------------------------- |
| `middleware/middleware.go` | `X-Request-Id`; lê `X-User-Id` / `X-Roles` (Gateway)  |
| `router.go`                | Rotas OpenAPI; handlers “burros”; mapeia erros → HTTP |


Handlers **não** contêm regra de negócio: só decode JSON → chama `app` → status code.

---



### 5.6 `internal/outbox/`


| Arquivo        | Papel                              |
| -------------- | ---------------------------------- |
| `writer.go`    | `Enqueue` → `INSERT outbox_events` |
| `publisher.go` | Flush periódico → Redis `XADD`     |


Envelope (ADR-003): `id`, `type`, `producer`, `version`, `correlation_id`, `data`, `occurred_at`.

Eventos emitidos pelo Identity:


| Tipo                | Quando             |
| ------------------- | ------------------ |
| `user.registered`   | Cadastro           |
| `user.updated`      | Perfil alterado    |
| `user.role_changed` | Admin troca papéis |


---



### 5.7 `internal/platform/logger/logger.go`

Logger `slog` JSON com campos `service` e `env`. Facilita grep em logs na live/demo.

---



## 6. Modelo de dados (migration)

Arquivo: `migrations/00001_init.sql` (goose).

```
users 1──1 credentials
  │
  ├──* user_roles
  ├──* refresh_tokens
  ├──* addresses
  └──* password_reset_tokens

outbox_events (independente, mesma base)
```


| Tabela                  | Motivo                                 |
| ----------------------- | -------------------------------------- |
| `users`                 | Dados cadastrais / status              |
| `credentials`           | Hash isolado (LGPD / menor superfície) |
| `user_roles`            | N:N simples; roles no JWT              |
| `refresh_tokens`        | Revogação e rotação por `jti`          |
| `addresses`             | Cadastro de endereços do usuário       |
| `password_reset_tokens` | Fluxo esqueci senha                    |
| `outbox_events`         | Entrega confiável de eventos           |


Ferramenta: **goose** (`goose -dir ./migrations postgres "$DATABASE_URL" up`).

---



## 7. API HTTP (contrato)

Espelho: `api/openapi.yaml` (canônico também em `docs/contracts/openapi/identity.yaml` no monorepo de docs).

### Públicos (sem Bearer / sem X-User-Id)


| Método | Path                       | Uso               |
| ------ | -------------------------- | ----------------- |
| GET    | `/health`                  | Liveness          |
| GET    | `/ready`                   | Postgres + Redis  |
| POST   | `/v1/auth/registro`        | Cadastro          |
| POST   | `/v1/auth/login`           | Tokens            |
| POST   | `/v1/auth/refresh`         | Renova (rota jti) |
| POST   | `/v1/auth/logout`          | Revoga refresh    |
| GET    | `/v1/auth/jwks.json`       | Chaves públicas   |
| POST   | `/v1/auth/esqueci-senha`   | Gera token reset  |
| POST   | `/v1/auth/redefinir-senha` | Consome token     |




### Autenticados via Gateway (headers)


| Método  | Path                            | Header                                   |
| ------- | ------------------------------- | ---------------------------------------- |
| GET/PUT | `/v1/usuarios/me`               | `X-User-Id`                              |
| CRUD    | `/v1/usuarios/me/enderecos`     | `X-User-Id`                              |
| PUT     | `/v1/admin/usuarios/{id}/roles` | `X-User-Id` + `X-Roles` contendo `admin` |


Modelo de erro padrão:

```json
{
  "erro": "mensagem",
  "codigo": "UNAUTHORIZED",
  "request_id": "…"
}
```

---



## 8. Sequências importantes



### 8.1 Login

```
Cliente → POST /v1/auth/login {email, senha}
       → AuthService.Login
       → UserRepo.FindByEmail + Argon2.Compare
       → TokenIssuer.IssueAccess (RS256)
       → TokenIssuer.IssueRefresh (jti → refresh_tokens)
Cliente ← { access_token, refresh_token, user }
```



### 8.2 Chamada autenticada (futuro Gateway)

```
Cliente → Gateway (Bearer access)
Gateway → valida JWKS do Identity
Gateway → Identity GET /v1/usuarios/me
           Headers: X-User-Id, X-Roles, X-Request-Id
Identity → ProfileService.Me(userID)
```



### 8.3 Registro + evento

```
Register → TX users/credentials/roles
        → INSERT outbox_events (user.registered)
Worker  → XADD vitalis.events
Outros serviços (futuro) consomem e projetam leitura local
```

---



## 9. Secrets e o que NÃO versionar


| Item              | Onde       | Commit?                                               |
| ----------------- | ---------- | ----------------------------------------------------- |
| `jwt_private.pem` | `secrets/` | **NÃO**                                               |
| `jwt_public.pem`  | `secrets/` | Evitar em repo público; ok em lab privado com cuidado |
| `.env`            | raiz       | **NÃO** (usar `.env.example`)                         |


Geração local (OpenSSL do Git no Windows):

```powershell
& "C:\Program Files\Git\usr\bin\openssl.exe" genrsa -out secrets/jwt_private.pem 2048
& "C:\Program Files\Git\usr\bin\openssl.exe" rsa -in secrets/jwt_private.pem -pubout -out secrets/jwt_public.pem
```

---



## 10. Como subir localmente

```powershell
cd microservices/vitalis-identity
copy .env.example .env
# Ajuste DATABASE_URL para 127.0.0.1:5433 se necessário
# HTTP_ADDR=:8081

docker compose up postgres redis -d
$env:Path = "$env:USERPROFILE\go\bin;$env:Path"
$env:DATABASE_URL = "postgres://vitalis:vitalis@127.0.0.1:5433/pg_identity?sslmode=disable"
goose -dir ./migrations postgres $env:DATABASE_URL up

go run ./cmd/api
# outro terminal:
go run ./cmd/worker
```

Smoke:

```powershell
curl http://127.0.0.1:8081/health   # ou :8080 conforme log

curl -X POST http://127.0.0.1:8081/v1/auth/registro `
  -H "Content-Type: application/json" `
  -d "{\"nome\":\"Ana\",\"email\":\"ana@test.com\",\"senha\":\"Senha@123\",\"tipo_usuario\":\"paciente\"}"

curl -X POST http://127.0.0.1:8081/v1/auth/login `
  -H "Content-Type: application/json" `
  -d "{\"email\":\"ana@test.com\",\"senha\":\"Senha@123\"}"

curl http://127.0.0.1:8081/v1/auth/jwks.json

curl http://127.0.0.1:8081/v1/usuarios/me -H "X-User-Id: <ID>"
```

---



## 11. Docker / CI / Makefile


| Arquivo                    | Função                                                               |
| -------------------------- | -------------------------------------------------------------------- |
| `docker-compose.yml`       | Postgres `pg_identity` (porta host 5433), Redis, opcional api/worker |
| `Dockerfile`               | Multi-stage: build `api` + `worker`                                  |
| `Makefile`                 | `tidy`, `run-api`, `run-worker`, `migrate-up`, `docker-up`           |
| `.github/workflows/ci.yml` | CI básico do serviço                                                 |
| `api/openapi.yaml`         | Contrato versionado junto do código                                  |


---



## 12. Decisões conscientes (MVP) e próximos passos



### Já feito no MVP

- [x] Register / Login / Refresh / Logout  
- [x] JWKS RS256  
- [x] Perfil e endereços via `X-User-Id`  
- [x] Admin roles  
- [x] Reset de senha (token devolvido em modo dev no JSON)  
- [x] Outbox writer + worker  



### Melhorias naturais depois

- [ ] Refresh em cookie `httpOnly` (front guarda access só em memória)  
- [ ] Rate limit no login  
- [ ] Outbox na **mesma TX** do domínio (passar `pgx.Tx` ao writer)  
- [ ] Remover `reset_token_dev` da resposta pública (só Comms/e-mail)  
- [ ] Testes unitários de `app` com fakes das portas  
- [ ] Alinhar nomes de env (`ACCESS_TTL_MIN` vs `JWT_ACCESS_TTL_MIN`) e porta padrão 8081  
- [ ] Gateway consumindo este JWKS  



### Próximo serviço sugerido

**Vitalis-gateway** — proxy reverso autenticado, validação JWT, injeção de headers, roteamento para Identity e demais serviços.

---



## 13. Glossário rápido (Identity)


| Termo            | Significado aqui                                              |
| ---------------- | ------------------------------------------------------------- |
| JWKS             | JSON Web Key Set — conjunto de chaves públicas                |
| jti              | JWT ID — identificador único do refresh no banco              |
| Outbox           | Tabela de eventos a publicar (padrão transactional outbox)    |
| Composition root | `main` que monta o grafo de dependências                      |
| Bounded context  | Limite do Identity: só identidade/auth, não clínica/pagamento |


---



## 14. Referências no repositório Vitalis

- `docs/ARQUITETURA_MICROSERVICOS_VITALIS.md`
- `docs/adrs/ADR-001` … `ADR-004`
- `docs/RBAC.md`
- `docs/contracts/openapi/identity.yaml`
- `docs/contracts/events/CATALOGO_EVENTOS.md`
- `docs/diagramas/servicos/identity.md`

---

*Documento gerado para acompanhar o desenvolvimento hands-on do Vitalis-identity (template Go → domínio real → demo login).*