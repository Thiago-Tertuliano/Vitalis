# ADR-004 — AuthN / AuthZ (JWT RS256 + RBAC + checagens no serviço)

| Campo | Valor |
|---|---|
| Status | Aceito |
| Data | 2026-09-12 |
| Decisores | Equipe Vitalis |
| Relacionados | ADR-001, ADR-003 (Gateway único), ADR-005 |

---

## Contexto

Os clientes só acessam o backend pelo Gateway (ADR-003). É necessário definir:

- quem **emite** credenciais;
- como o token é **assinado e validado**;
- onde o **web** guarda tokens;
- quais **papéis** existem;
- como autorizar de forma **profissional** (não só “tem role X”).

---

## Decisão

### 1. Emissor (AuthN)

- **Somente `Vitalis-identity` emite** access token e refresh token.
- Outros serviços **não** assinam JWT de usuário.
- O Gateway **valida** o access token; não emite.

### 2. Algoritmo JWT

- **RS256** (par de chaves assimétrico).
- Identity guarda a **chave privada** (só ele assina).
- Gateway (e, se necessário, serviços) validam via **JWKS** público do Identity.
- **HS256 rejeitado** como padrão (segredo compartilhado entre muitos repos é frágil em polyrepo).

### 3. Armazenamento no front (web)

Padrão alvo (mais seguro entre as opções práticas):

| Token | Onde |
|---|---|
| **Access token** | **Memória** da aplicação (não `localStorage`) |
| **Refresh token** | **Cookie httpOnly** (Secure, SameSite adequado) |

Mobile: secure storage da plataforma (Keychain/Keystore), sem espelhar `localStorage` web.

> Nota: detalhes de cookie domain/path no Docker local serão fixados no template do Gateway/Identity; a política acima é a decisão de segurança.

### 4. Papéis oficiais (RBAC)

Papéis do sistema (nomes canônicos em PT no token claim `roles`):

| Role | Equivalente falado |
|---|---|
| `admin` | admin |
| `paciente` | patient |
| `medico` | doctor |
| `farmacia` | farmácia (cliente web farmácias) |
| `motoboy` | motoboy / entrega |

`farmacia` e `motoboy` permanecem porque o ADR-001 define clientes dedicados para esses atores. Se quiserem reduzir o conjunto depois, abre-se ADR de alteração de papéis.

### 5. Autorização (o que é mais profissional) — decisão

**Roles no JWT (RBAC) + checagens no serviço (ownership / tenant).**

| Camada | O que faz |
|---|---|
| **Gateway** | Autentica JWT; pode exigir role mínima em algumas rotas |
| **JWT claim `roles`** | Autorização **grosseira** (ex.: só `medico` acessa rotas clínicas de profissional) |
| **Serviço de domínio** | Autorização **fina**: recurso pertence ao usuário / farmácia / corrida do motoboy |

**Por quê essa combinação é a mais profissional:**

- Só RBAC no JWT **não basta** (“é paciente” ≠ “pode ver o pedido do outro paciente”).
- Só checagem no serviço sem roles vira bagunça de `if` e dificulta o Gateway.
- Padrão de mercado em APIs multi-tenant: **coarse RBAC + fine-grained resource checks**.

### 6. Refresh token

- **Sim**: refresh com **rotação** e **revogação** no Identity.
- Refresh tokens persistidos (ex.: `refresh_tokens` com jti, expiry, revoked).
- Access token de vida curta; refresh de vida maior, rotacionado a cada uso.

### 7. Serviço → serviço (Gateway → Clinical, etc.)

**Padrão profissional adotado: A + B**

1. Cliente envia `Authorization: Bearer <access>`.
2. **Gateway valida** JWT (assinatura JWKS, exp, issuer).
3. Gateway **propaga** `Authorization` (opcional, útil para auditoria) **e injeta headers internos confiáveis**:
   - `X-User-Id`
   - `X-Roles`
   - `X-Request-Id`
   - (opcional) `X-Gateway: 1`
4. Serviços de domínio:
   - **confiam** nesses headers **somente** na rede interna (não expor portas 8081–8091);
   - aplicam RBAC + ownership;
   - **não** aceitam `X-User-Id` vindo da internet pública.

Sem mTLS no momento (Docker local); a fronteira de confiança é a rede `vitalis_net` + Gateway como único ingresso.

---

## Critério de sucesso deste ADR

- [ ] Login/refresh só via Identity (através do Gateway).
- [ ] Gateway rejeita access token inválido/expirado.
- [ ] Web: access em memória; refresh em cookie httpOnly.
- [ ] Endpoint de exemplo com role + ownership (ex.: pedido só do próprio `paciente`).
- [ ] Refresh rotaciona e pode ser revogado (logout).

---

## Consequências

### Positivas

- Modelo alinhado a APIs profissionais multi-ator.
- Chaves assimétricas cabem bem em polyrepo.
- Menos risco XSS no access token (memória vs localStorage).

### Negativas / trade-offs

- Cookie refresh exige cuidado com CSRF (SameSite / anti-CSRF onde aplicável).
- Headers internos exigem disciplina de rede (não publicar serviços).
- Ownership checks são código a mais em cada serviço (necessário).

### Mitigações

- Documentar matriz RBAC em `docs/` (próximo pacote Glossário/RBAC).
- Testes de autorização negativos (usuário A não lê recurso de B).
- Rate limit no login/refresh no Gateway/Identity.

---

## Alternativas consideradas

| Alternativa | Decisão |
|---|---|
| HS256 compartilhado | Rejeitado |
| Access token só em localStorage | Rejeitado como padrão (XSS) |
| Só roles no JWT, sem ownership | Rejeitado — inseguro multi-tenant |
| Cada serviço revalida JWKS sempre + sem headers | Possível, mas A+B é mais prático no Gateway único |
| Sem refresh | Rejeitado — UX ruim |

---

## Próximo

- Detalhes financeiros / Billing vitrine → **ADR-005**.
