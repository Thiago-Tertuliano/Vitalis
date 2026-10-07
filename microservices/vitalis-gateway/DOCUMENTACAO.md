# Vitalis-gateway — Documentação completa

> **Porta única de entrada** da plataforma Vitalis.  
> Este documento explica o **porquê** de cada decisão, o papel de cada pasta/arquivo e como uma requisição atravessa o Gateway de ponta a ponta.  
> Útil para live, TCC, onboarding e revisão de código.


| Campo                 | Valor                                                                 |
| --------------------- | --------------------------------------------------------------------- |
| Repo / módulo Go      | `github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-gateway`         |
| Porta HTTP (contrato) | `8080`                                                                |
| Banco                 | nenhum (stateless)                                                    |
| Stack                 | Go 1.27 · chi · `httputil.ReverseProxy` · golang-jwt v5 · x/time/rate |
| Depende de            | Identity — `GET /v1/auth/jwks.json`                                   |
| ADRs                  | ADR-001 (polyrepo), ADR-003 (HTTP + eventos), ADR-004 (auth/RBAC)     |


---


## 1. Por que o Gateway existe?

Com 12 microserviços, cada um teria que repetir: validar JWT, CORS, rate limit, request id, log de acesso, checagem de papel. Repetir isso 12 vezes significa 12 lugares para errar.

O **Gateway** concentra essas preocupações transversais num único ponto:

- é o **único** endereço público (`:8080`); os serviços (8081–8091) ficam na rede interna
- valida o **access token** com as chaves públicas do Identity
- aplica a **autorização grossa** (papel por rota)
- **injeta** a identidade confiável (`X-User-Id`, `X-Roles`) para o serviço de destino
- **encaminha** a requisição (reverse proxy), inclusive WebSocket

O que ele **não** faz:

- não tem banco, não guarda sessão, não tem regra de negócio
- não emite token (só o Identity tem a chave privada)
- não decide *ownership* ("esse pedido é desse usuário?") — isso é do serviço dono do dado (ADR-004)

### Por que foi o segundo serviço?

Sem Gateway, cada serviço novo teria que validar JWT sozinho. Com o Gateway pronto, Billing, Clinical, Orders… nascem confiando em `X-User-Id` e só cuidam do próprio domínio.

**Identity → Gateway → Billing → Clinical / Commerce / Orders…**

---


## 2. Princípios de construção


### 2.1 Stateless

Nenhum estado precisa sobreviver a um restart: as chaves vêm do Identity, a tabela de rotas está no código, o rate limit é em memória. Isso permite subir N réplicas atrás de um balanceador sem coordenação (o rate limit distribuído em Redis fica para a fase 2).

### 2.2 Negar por padrão (fail closed)

| Situação                           | Resposta                                |
| ---------------------------------- | --------------------------------------- |
| Rota fora da tabela                | 404 (nem chega a nenhum serviço)        |
| Token ausente, inválido, expirado  | 401                                     |
| Papel insuficiente                 | 403                                     |
| Caminho com `..`                   | 400                                     |
| Serviço fora do ar                 | 502 (nunca "deixa passar")              |

Só **duas** rotas são públicas: `/v1/auth/*` e `/v1/webhooks/pagamentos`. Um teste (`TestTable_SoAuthEWebhookSaoPublicos`) quebra o build se alguém tornar outra rota pública sem querer.

### 2.3 Defesa em profundidade (ADR-004)

O Gateway faz **autenticação** e **autorização por papel**. O serviço de destino faz **ownership**. Se um dia alguém expuser um serviço interno por engano, ele ainda depende de `X-User-Id` — por isso a regra "não expor 8081–8091" é parte da arquitetura, não detalhe de deploy.

### 2.4 Por que não é hexagonal como o Identity?

O Identity tem domínio (usuário, senha, papéis), então separar `domain`/`port`/`app`/`adapter` paga o custo. O Gateway **não tem domínio**: é infraestrutura pura. Aqui a organização é por **responsabilidade técnica** (rotas, auth, proxy, middleware), cada pacote pequeno e testável isoladamente. Forçar portas e adaptadores sem domínio seria cerimônia sem benefício.

### 2.5 Sem segredo compartilhado

Com **RS256**, o Identity assina com a chave privada e o Gateway valida com a pública, baixada do JWKS. Se o Gateway for comprometido, o atacante **não** consegue emitir tokens. Com HS256 (segredo compartilhado) qualquer serviço que valida também poderia assinar.

---


## 3. Mapa da árvore do projeto

```
vitalis-gateway/
├── cmd/
│   └── api/main.go              # composition root — monta e sobe o servidor
├── internal/
│   ├── config/config.go         # env → struct Config
│   ├── routes/table.go          # prefixo → serviço + regra de acesso
│   ├── auth/
│   │   ├── jwks.go              # baixa e cacheia as chaves públicas do Identity
│   │   └── validator.go         # valida o JWT (alg, iss, aud, exp, typ)
│   ├── proxy/proxy.go           # um ReverseProxy por serviço interno
│   ├── gateway/handler.go       # coração: rota → auth → roles → headers → proxy
│   ├── middleware/
│   │   ├── requestid.go         # X-Request-Id
│   │   ├── logging.go           # log JSON de acesso
│   │   └── ratelimit.go         # token bucket por IP
│   ├── httpserver/router.go     # chi + pilha de middlewares + /health /ready
│   ├── httpx/respond.go         # JSON e erro padrão
│   └── platform/logger/         # slog JSON
├── api/openapi.yaml             # contrato + x-routing-table
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── .env.example
├── DOCUMENTACAO.md              # este arquivo
└── README.md
```

Testes ficam ao lado do código (`*_test.go`) em `gateway/`, `routes/`, `auth/` e `middleware/`.

---


## 4. A vida de uma requisição

Ordem exata em que uma requisição atravessa o Gateway:

```
Cliente
  │
  ▼
┌─────────────────────────── httpserver/router.go ───────────────────────────┐
│ 1. Recoverer      panic em qualquer handler vira 500, servidor não cai     │
│ 2. RequestID      gera/propaga X-Request-Id (antes do log, para logá-lo)   │
│ 3. Logging        mede duração e status (envolve todo o resto)            │
│ 4. CORS           responde preflight OPTIONS sem gastar rate limit/auth   │
│ 5. RateLimiter    429 se o IP estourou o balde                            │
│ 6. maxBody        limita o corpo a MAX_BODY_MB                            │
│                                                                            │
│   /health  /ready  → respondem direto                                      │
│   /v1/*            → gateway.Handler                                       │
└────────────────────────────────────────────────────────────────────────────┘
  │
  ▼
┌─────────────────────────── gateway/handler.go ─────────────────────────────┐
│ 1. apaga X-User-Id, X-Roles, X-Gateway vindos do cliente                   │
│ 2. path limpo? senão 400                                                   │
│ 3. routes.Match → senão 404                                                │
│ 4. rota autenticada? valida JWT (401) e papel (403)                        │
│ 5. injeta X-User-Id, X-Roles; marca X-Gateway: 1                           │
│ 6. proxy do serviço → ReverseProxy                                         │
└────────────────────────────────────────────────────────────────────────────┘
  │
  ▼
Serviço interno (Identity, Billing, …)
```

**Por que essa ordem?**

- `RequestID` vem antes de `Logging` para o log ter o id.
- `Logging` envolve tudo, então até um 429 ou um 401 aparece no log.
- `CORS` antes do rate limit: o navegador manda um `OPTIONS` antes de cada chamada cross-origin; contar isso no balde puniria o usuário em dobro.
- A autenticação fica **dentro** do handler (não num middleware global) porque depende da rota: só se sabe se precisa de token depois do `routes.Match`.

---


## 5. Arquivo a arquivo — o que faz e por quê


### 5.1 `internal/config/config.go`

Lê variáveis de ambiente com defaults seguros para desenvolvimento local.

| Env                     | Default                                    | Uso                                          |
| ----------------------- | ------------------------------------------ | -------------------------------------------- |
| `HTTP_ADDR`             | `:8080`                                    | porta do Gateway                             |
| `APP_ENV`               | `local`                                    | `local`/`dev` liga log em nível debug        |
| `JWKS_URL`              | `http://127.0.0.1:8081/v1/auth/jwks.json`  | de onde vêm as chaves públicas               |
| `JWKS_REFRESH_MIN`      | `10`                                       | idade máxima do cache de chaves              |
| `JWT_ISSUER`            | `vitalis-identity`                         | claim `iss` exigido                          |
| `JWT_AUDIENCE`          | `vitalis`                                  | claim `aud` exigido                          |
| `CORS_ALLOWED_ORIGINS`  | `http://localhost:3000,http://localhost:5173` | allowlist (CSV)                           |
| `RATE_LIMIT_RPS`        | `20`                                       | requisições/segundo por IP                   |
| `RATE_LIMIT_BURST`      | `40`                                       | rajada permitida                             |
| `MAX_BODY_MB`           | `10`                                       | tamanho máximo do corpo                      |
| `UPSTREAM_TIMEOUT_SEC`  | `30`                                       | tempo máximo esperando o serviço responder   |
| `SHUTDOWN_TIMEOUT_SEC`  | `10`                                       | tempo para terminar requisições no desligamento |
| `UPSTREAM_<SERVIÇO>`    | `http://127.0.0.1:8081` … `:8091`          | endereço de cada serviço interno             |

**Por que tudo por env?** O mesmo binário roda local, em Docker (`host.docker.internal`) e em produção (`http://identity:8081`), mudando só a configuração.

> **Windows:** salve o `.env` em UTF-8 **sem BOM**. Com BOM, o `godotenv` falha na primeira linha e ignora o arquivo inteiro.

---


### 5.2 `internal/routes/table.go`

A tabela de rotas é uma lista de `Route{Prefix, Upstream, Access, Roles}`, espelho do `x-routing-table` do OpenAPI e da matriz do `docs/RBAC.md`.

```go
{Prefix: "/v1/admin/usuarios", Upstream: "identity", Access: Authenticated, Roles: []string{"admin"}},
```

`Match` casa o prefixo **exato** ou **seguido de `/`**:

| Path                  | Casa com `/v1/medicos`? |
| --------------------- | ----------------------- |
| `/v1/medicos`         | sim                     |
| `/v1/medicos/123`     | sim                     |
| `/v1/medicosX`        | **não**                 |

**Por que não `strings.HasPrefix` puro?** Porque `/v1/auth` casaria com `/v1/authAdmin`, transformando uma rota futura em pública sem ninguém perceber.

**Por que uma lista e não um mapa?** A ordem importa (o primeiro que casa vence) e uma lista lida de cima para baixo é fácil de revisar numa live ou num PR.

---


### 5.3 `internal/auth/jwks.go`

Baixa `GET /v1/auth/jwks.json` do Identity e guarda as chaves RSA em memória, indexadas por **`kid`** (key id).

| Comportamento                         | Por quê                                                                        |
| ------------------------------------- | ------------------------------------------------------------------------------ |
| Cache por `kid`                       | Permite **rotação de chave**: Identity publica a nova junto com a antiga       |
| Rebusca após `JWKS_REFRESH_MIN`       | Chave nova aparece sem reiniciar o Gateway                                     |
| `kid` desconhecido → rebusca, no máximo 1x a cada 30 s (`minRefetch`) | Um atacante mandando tokens com `kid` aleatório não transforma o Gateway num gerador de flood contra o Identity |
| Identity fora do ar → usa a chave em cache | Queda do Identity não derruba o login de quem já tem token válido        |
| `Ready()`                             | Usado pelo `/ready`: sem chave carregada, o Gateway não consegue validar nada  |

`rsaFromJWK` monta a `rsa.PublicKey` a partir de `n` (módulo) e `e` (expoente), ambos em base64url, como define a RFC 7517.

---


### 5.4 `internal/auth/validator.go`

Valida o token com um `jwt.Parser` configurado para:

| Checagem                          | Ataque que bloqueia                                                       |
| --------------------------------- | ------------------------------------------------------------------------- |
| `WithValidMethods(["RS256"])`     | **Algorithm confusion**: trocar `alg` para `HS256` e assinar usando a chave pública como segredo; ou `alg: none` |
| `WithIssuer` / `WithAudience`     | Token emitido por outro sistema ou para outro público                     |
| `WithExpirationRequired`          | Token sem `exp` (eterno)                                                  |
| `WithLeeway(30s)`                 | Tolera relógios levemente dessincronizados entre máquinas                 |
| `typ == "access"`                 | **Refresh usado como access**: o refresh vale 7 dias e tem a mesma assinatura |
| `uid` não vazio                   | Token sem dono                                                            |

A chave de verificação vem do `kid` no header, via `JWKS.Key`.

**Por que o `typ`?** Antes, Identity emitia access e refresh com a mesma chave e claims parecidos. Sem essa checagem, quem roubasse um refresh token teria acesso à API por 7 dias em vez de 15 minutos. O Identity agora marca `typ: access` / `typ: refresh` e o Gateway exige `access`.

---


### 5.5 `internal/proxy/proxy.go`

Cria um `httputil.ReverseProxy` por serviço, todos compartilhando um único `http.Transport`.

```go
Rewrite: func(pr *httputil.ProxyRequest) {
    pr.SetURL(target)  // troca host/scheme, mantém path e query
    pr.SetXForwarded() // X-Forwarded-For/Host/Proto
},
```

| Decisão                                  | Por quê                                                                                         |
| ---------------------------------------- | ----------------------------------------------------------------------------------------------- |
| `Rewrite` em vez do antigo `Director`    | Com `Rewrite`, o Go **remove** os `X-Forwarded-*` enviados pelo cliente antes de escrever os verdadeiros; com `Director`, um cliente podia forjar o próprio IP |
| Transport compartilhado com keep-alive   | Reaproveita conexões TCP com os serviços (menos latência, menos portas abertas)                 |
| `ResponseHeaderTimeout = UPSTREAM_TIMEOUT_SEC` | Serviço travado não segura a conexão do cliente para sempre                               |
| `ErrorHandler` → 502 `BAD_GATEWAY`       | Erro de rede vira resposta JSON no padrão Vitalis, com `request_id`, e um log `upstream indisponível` |

O `ReverseProxy` também remove *hop-by-hop headers* (`Connection`, `Keep-Alive`…) e trata `Upgrade: websocket` sozinho.

---


### 5.6 `internal/gateway/handler.go` — o coração

São seis passos, numerados nos comentários do código (o melhor arquivo para explicar na live):

1. **Anti-spoofing** — `r.Header.Del` em `X-User-Id`, `X-Roles`, `X-Gateway`. Sem isso, `curl -H "X-User-Id: <id da vítima>"` numa rota pública chegaria ao serviço.
2. **Path limpo** — `path.Clean(p) == p`. Bloqueia `/v1/auth/../admin/usuarios`, que casaria com a rota **pública** `/v1/auth` mas pode ser normalizado por um proxy ou framework no caminho para `/v1/admin/usuarios`, pulando a exigência de `admin`.
3. **Roteamento** — `routes.Match`; fora da tabela → 404.
4. **Autenticação + papel** — só em rotas `Authenticated`. Token ausente/inválido → 401; papel insuficiente → 403.
5. **Injeção de identidade** — `X-User-Id` e `X-Roles` vêm **do token validado**, nunca do cliente. `X-Gateway: 1` marca que passou pelo Gateway.
6. **Proxy** — entrega ao `ReverseProxy` do serviço.

**Token por query só para WebSocket.** O navegador não deixa enviar `Authorization` no handshake de WebSocket, então `/v1/realtime?access_token=…` é aceito **apenas** com `Upgrade: websocket`. Depois de validado, `stripQueryToken` remove o token da URL para ele não aparecer nos logs do serviço de destino. Numa requisição comum, token na query é ignorado (401).

**401 vs 403.** 401 = "não sei quem você é" (token ruim). 403 = "sei quem você é, mas você não pode" (papel). A mensagem do 401 é genérica (`token inválido ou expirado`); o motivo real vai só para o log em nível debug.

---


### 5.7 `internal/middleware/`

#### `requestid.go`

Usa o `X-Request-Id` do cliente se existir (até 128 caracteres) ou gera um UUID. Escreve no request (vai para o serviço) e na resposta (volta para o cliente).

**Por quê?** Um único id liga o log do Gateway, o log do Identity e o erro que o usuário viu. O limite de 128 impede que alguém encha os logs com um header gigante.

#### `logging.go`

Uma linha JSON por requisição: `method`, `path`, `status`, `bytes`, `duration_ms`, `request_id`, `user_id`. **Nunca** loga o header `Authorization` nem a query.

O `statusRecorder` envolve o `ResponseWriter` para capturar o status. O método `Unwrap()` é essencial: sem ele o `ReverseProxy` não consegue fazer `Hijack` da conexão e o **WebSocket quebraria**.

#### `ratelimit.go`

Token bucket por IP (`golang.org/x/time/rate`): `RATE_LIMIT_RPS` fichas por segundo, até `RATE_LIMIT_BURST` acumuladas. Estourou → 429 com `Retry-After: 1`. Uma goroutine remove IPs parados há mais de 3 minutos para o mapa não crescer para sempre.

**Por que o IP da conexão e não `X-Forwarded-For`?** O Gateway é a borda; qualquer cliente pode mandar `X-Forwarded-For: 1.2.3.4` diferente a cada requisição e nunca ser limitado. Pelo mesmo motivo o router **não** usa `chimw.RealIP` (que o próprio chi depreciou por isso). Há um teste (`TestRateLimiter_IgnoraXForwardedFor`) garantindo esse comportamento.

---


### 5.8 `internal/httpserver/router.go`

Monta o chi com a pilha da seção 4 e três rotas:

| Rota      | Comportamento                                                                 |
| --------- | ----------------------------------------------------------------------------- |
| `/health` | Sempre 200 se o processo está de pé (liveness)                                |
| `/ready`  | 200 se já tem chave do Identity; senão tenta buscar (3 s) e responde 503 `degraded` (readiness) |
| `/v1/*`   | `gateway.Handler`                                                             |

**CORS:** `AllowCredentials: true` porque o refresh token vai morar num cookie `httpOnly` (ADR-004); por isso as origens precisam ser uma **allowlist** explícita — o navegador recusa `*` com credenciais.

**`maxBody`:** `http.MaxBytesReader` corta o corpo em `MAX_BODY_MB`, protegendo os serviços de uploads gigantes que nunca deveriam chegar até eles.

---


### 5.9 `internal/httpx/respond.go`

`WriteJSON` e `WriteError`. Todo erro do Gateway segue o modelo `Error` do OpenAPI:

```json
{
  "erro": "token inválido ou expirado",
  "codigo": "UNAUTHORIZED",
  "request_id": "218cd8b8-62b4-441a-b65a-ccc00da71357"
}
```

**Por que um pacote separado?** `middleware`, `proxy`, `gateway` e `httpserver` usam a mesma função. Se ela ficasse em `httpserver`, haveria import circular.

---


### 5.10 `cmd/api/main.go` — composition root

1. Carrega `.env` e `config.Load()`
2. Cria o logger
3. Cria o `JWKS` e tenta a primeira busca (5 s). Se falhar, **só avisa**: o Identity pode subir depois
4. Cria o `Validator`, o `Registry` de proxies e o `RateLimiter`
5. Monta o router e sobe o `http.Server`
6. No `SIGINT`/`SIGTERM`, `Shutdown` gracioso (termina as requisições em andamento)

| Configuração do servidor | Por quê                                                                     |
| ------------------------ | --------------------------------------------------------------------------- |
| `ReadHeaderTimeout: 10s` | Mitiga **Slowloris** (cliente que manda headers byte a byte para segurar conexões) |
| **Sem** `WriteTimeout`   | Derrubaria conexões WebSocket longas de `/v1/realtime`                       |

---


### 5.11 `internal/platform/logger/logger.go`

Igual ao do Identity: `slog` JSON com `service` e `env` em toda linha. Em `local`/`dev`, nível debug (mostra o motivo dos JWT rejeitados).

---


## 6. Tabela de rotas

| Prefixo                                                                                     | Serviço         | Acesso                       |
| ------------------------------------------------------------------------------------------- | --------------- | ---------------------------- |
| `/v1/auth`                                                                                  | identity:8081   | público                      |
| `/v1/webhooks/pagamentos`                                                                   | billing:8086    | público (Billing valida HMAC) |
| `/v1/usuarios`                                                                              | identity:8081   | autenticado                  |
| `/v1/admin/usuarios`                                                                        | identity:8081   | `admin`                      |
| `/v1/payment-intents` `/subscriptions` `/wallets` `/invoices` `/settlements` `/commissions` | billing:8086    | autenticado                  |
| `/v1/especialidades` `/medicos` `/disponibilidade` `/consultas` `/triagens` `/videochamadas` `/prescricoes` | clinical:8082 | autenticado        |
| `/v1/farmacias` `/categorias` `/produtos` `/estoque`                                        | commerce:8083   | autenticado                  |
| `/v1/pedidos`                                                                               | orders:8084     | autenticado                  |
| `/v1/entregas` `/motoboys`                                                                  | delivery:8085   | autenticado                  |
| `/v1/notificacoes` `/chat` `/realtime` (WebSocket)                                          | comms:8087      | autenticado                  |
| `/v1/tickets` `/artigos-ajuda`                                                              | support:8088    | autenticado                  |
| `/v1/objetos` `/pdf`                                                                        | files:8089      | autenticado                  |
| `/v1/search`                                                                                | search:8090     | autenticado                  |
| `/v1/audit`                                                                                 | audit:8091      | `admin`                      |

Rotas `/v1/internal/*` (chamadas serviço→serviço) **não** estão na tabela, então nunca são alcançáveis de fora.

---


## 7. Modelo de ameaças

Cada linha tem um teste automatizado em `internal/gateway/handler_test.go` (ou no pacote indicado).

| Ataque                                             | Defesa                                         | Resultado |
| -------------------------------------------------- | ---------------------------------------------- | --------- |
| Chamar rota protegida sem token                    | `bearerToken` vazio                            | 401       |
| Forjar `X-User-Id` / `X-Roles`                     | headers apagados no passo 1                    | 401 (ou o uid real do token) |
| Usar o refresh token (7 dias) como access          | `typ != "access"`                              | 401       |
| Token assinado com outra chave                     | assinatura não confere com o JWKS              | 401       |
| Trocar `alg` para `HS256` usando a chave pública   | `WithValidMethods(["RS256"])`                  | 401       |
| Token expirado / de outro issuer                   | `exp` / `iss` obrigatórios                     | 401       |
| Paciente acessando `/v1/admin` ou `/v1/audit`      | `Roles: ["admin"]` na rota                     | 403       |
| `/v1/auth/../admin/usuarios`                       | `isCleanPath`                                  | 400       |
| Alcançar `/v1/internal/*`                          | fora da tabela                                 | 404       |
| `/v1/usuariosX` para escapar do prefixo            | `Match` exige `/` após o prefixo               | 404       |
| Token de WebSocket vazar para logs internos        | `stripQueryToken`                              | query limpa |
| `kid` aleatório para inundar o Identity            | `minRefetch` 30 s (`auth/jwks_test.go`)        | 1 busca   |
| Burlar rate limit com `X-Forwarded-For`            | IP da conexão (`middleware/middleware_test.go`) | 429      |
| Slowloris                                          | `ReadHeaderTimeout`                            | conexão fechada |

**Ameaça residual principal:** expor um serviço interno (8081–8091) diretamente. Ele confia em `X-User-Id` e qualquer um poderia se passar por outro usuário. Mitigação: rede privada no Compose/cluster e, no futuro, mTLS entre Gateway e serviços.

---


## 8. Sequências importantes


### 8.1 Login (rota pública)

```
Cliente → POST :8080/v1/auth/login {email, senha}
Gateway → apaga X-User-Id/X-Roles, path ok, rota pública (sem JWT)
Gateway → POST identity:8081/v1/auth/login  (+ X-Request-Id, X-Gateway, X-Forwarded-For)
Identity → valida senha, emite access (typ=access, kid) + refresh (typ=refresh)
Cliente ← { access_token, refresh_token, user }
```


### 8.2 Chamada autenticada

```
Cliente → GET :8080/v1/usuarios/me   Authorization: Bearer <access>
Gateway → JWKS.Key(kid)  (cache; busca no Identity se expirou)
Gateway → valida RS256, iss, aud, exp, typ=access
Gateway → GET identity:8081/v1/usuarios/me
           X-User-Id: <uid do token>   X-Roles: paciente   X-Request-Id: …
Identity → ProfileService.Me(uid)
Cliente ← 200 { perfil }
```


### 8.3 WebSocket

```
Navegador → GET :8080/v1/realtime?access_token=<access>&sala=1
            Upgrade: websocket
Gateway  → valida o token da query, remove access_token
Gateway  → comms:8087/v1/realtime?sala=1  (+ X-User-Id)  → 101 Switching Protocols
```


### 8.4 Identity fora do ar

```
Gateway já tem a chave em cache  → continua validando tokens normalmente
Rotas /v1/auth/* (login)         → 502 "serviço identity indisponível"
Gateway sem chave (subiu sozinho) → rotas autenticadas 401, /ready 503
```


### 8.5 Rotação de chave (futuro)

```
Identity passa a publicar [kid-2 (nova), kid-1 (antiga)] e assina com kid-2
Gateway recebe token com kid-2 desconhecido → rebusca o JWKS → valida
Tokens antigos (kid-1) seguem válidos até expirar; depois o Identity remove kid-1
```

---


## 9. Como subir localmente

```powershell
# Terminal 1 — Identity
cd microservices/vitalis-identity
docker compose up postgres redis -d
go run ./cmd/api                      # :8081

# Terminal 2 — Gateway
cd microservices/vitalis-gateway
copy .env.example .env
go run ./cmd/api                      # :8080
```

Smoke:

```powershell
curl.exe http://127.0.0.1:8080/health
curl.exe http://127.0.0.1:8080/ready

$login = Invoke-RestMethod -Method Post -Uri http://127.0.0.1:8080/v1/auth/login `
  -ContentType "application/json" -Body '{"email":"ana@test.com","senha":"Senha@123"}'
Invoke-RestMethod -Uri http://127.0.0.1:8080/v1/usuarios/me `
  -Headers @{ Authorization = "Bearer $($login.access_token)" }
```

Em container (Identity continua na máquina):

```powershell
docker compose up --build -d          # usa host.docker.internal:8081
```

---


## 10. Testes

```powershell
go test -cover ./...
```

| Arquivo                                  | Cenários                                                                                   |
| ---------------------------------------- | ------------------------------------------------------------------------------------------ |
| `internal/gateway/handler_test.go`       | 19 status (público, válido, sem token, forjado, refresh, issuer, expirado, outra chave, HS256, malformado, admin/audit, 404, traversal, 503, 502) + injeção de identidade + rota pública sem identidade forjada + WebSocket |
| `internal/routes/table_test.go`          | casamento de prefixo; só auth e webhook públicos                                           |
| `internal/auth/jwks_test.go`             | reconstrução da chave e cache; `kid` forjado não vira flood; cache com Identity fora       |
| `internal/middleware/middleware_test.go` | rate limit por IP e imune a `X-Forwarded-For`; request id gerado, propagado e limitado      |

Os testes sobem servidores `httptest` no lugar do Identity (JWKS) e dos serviços (eco dos headers recebidos), geram chaves RSA na hora e assinam tokens de verdade — nenhum segredo fica no repositório.

---


## 11. Docker / CI / Makefile

| Arquivo                                | Função                                                                     |
| -------------------------------------- | -------------------------------------------------------------------------- |
| `Dockerfile`                           | Multi-stage: build estático (`CGO_ENABLED=0`) → Alpine, roda como `nobody` |
| `docker-compose.yml`                   | Gateway em container apontando para o Identity no host                     |
| `Makefile`                             | `tidy`, `run`, `test`, `build`, `docker-up`, `docker-down`                 |
| `.github/workflows/ci.yml`             | CI do serviço isolado (para quando virar repositório próprio, ADR-001)     |
| `/.github/workflows/quality.yml` (raiz) | Pipeline que roda hoje: `golangci-lint` (staticcheck, gosec), `go test -race -cover`, `govulncheck`, Trivy, Semgrep, Gitleaks |
| `api/openapi.yaml`                     | Contrato e `x-routing-table`                                               |

---


## 12. Decisões conscientes (MVP) e próximos passos


### Já feito no MVP

- [x] Reverse proxy para os 11 serviços com tabela de rotas
- [x] Validação JWT RS256 via JWKS com cache por `kid`
- [x] Autorização por papel (`admin`)
- [x] Anti-spoofing de headers, path traversal, rotas internas bloqueadas
- [x] Rate limit por IP, CORS allowlist, limite de corpo, request id, log JSON
- [x] Proxy de WebSocket com token por query
- [x] Testes automatizados dos cenários de segurança


### Melhorias naturais depois

- [ ] Rate limit distribuído em Redis (várias réplicas do Gateway)
- [ ] Limite mais rígido em `/v1/auth/login` (brute force)
- [ ] `/metrics` Prometheus (latência e status por rota/serviço), previsto no diagrama 2.1
- [ ] Responder 413 explícito quando o `Content-Length` passar de `MAX_BODY_MB`
- [ ] Lista de proxies confiáveis, se houver um load balancer na frente (aí sim ler `X-Forwarded-For` dele)
- [ ] Circuit breaker / timeout por rota (ex.: PDF pode demorar mais que login)
- [ ] mTLS entre Gateway e serviços
- [ ] TLS terminado no Gateway ou no load balancer em produção


### Próximo serviço sugerido

**Vitalis-billing** — pagamentos, assinaturas e carteiras. O Gateway já encaminha `/v1/payment-intents`, `/v1/subscriptions` e o webhook público `/v1/webhooks/pagamentos`.

---


## 13. Glossário rápido (Gateway)

| Termo              | Significado aqui                                                            |
| ------------------ | --------------------------------------------------------------------------- |
| Reverse proxy      | Recebe a requisição do cliente e a reenvia ao serviço interno               |
| JWKS               | JSON Web Key Set — chaves públicas publicadas pelo Identity                 |
| `kid`              | Key ID — diz qual chave do JWKS assinou o token                             |
| `typ`              | Claim que diferencia access de refresh                                      |
| Algorithm confusion | Ataque que troca o `alg` do token para enganar o validador                 |
| Token bucket       | Algoritmo de rate limit: fichas repostas a taxa fixa, consumidas por requisição |
| Hop-by-hop headers | Headers que valem só para uma conexão (`Connection`, `Upgrade`) e não devem ser repassados |
| Liveness / readiness | `/health` = processo vivo; `/ready` = apto a atender (tem chaves)         |
| Fail closed        | Na dúvida, negar                                                            |

---


## 14. Referências no repositório Vitalis

- `docs/ARQUITETURA_MICROSERVICOS_VITALIS.md`
- `docs/adrs/ADR-001-microservicos-polyrepo.md`
- `docs/adrs/ADR-003-comunicacao-http-eventos.md`
- `docs/adrs/ADR-004-auth-rbac.md`
- `docs/RBAC.md`
- `docs/contracts/openapi/gateway.yaml`
- `docs/diagramas/servicos/gateway.md`
- [`../vitalis-identity/DOCUMENTACAO.md`](../vitalis-identity/DOCUMENTACAO.md) — quem emite os tokens que este serviço valida

---

*Documento gerado para acompanhar o desenvolvimento hands-on do Vitalis-gateway (proxy autenticado na frente do Identity e dos próximos serviços).*
