# ADR-003 — Comunicação entre serviços (HTTP + eventos + Gateway)

| Campo | Valor |
|---|---|
| Status | Aceito |
| Data | 2026-09-12 |
| Decisores | Equipe Vitalis |
| Relacionados | ADR-001, ADR-002, ADR-004, `Vitalis-contracts` |

---

## Contexto

Com microserviços em polyrepo (ADR-001) e stack Go/Postgres/Redis (ADR-002), é preciso definir **como** os serviços e os clientes se comunicam, para:

- evitar acoplamento caótico (chamadas cruzadas sem padrão);
- garantir que frontends não “bypassem” o Gateway;
- tornar eventos confiáveis (sem perder mensagem após commit no banco);
- manter contratos claros para estudo, TCC e evolução futura.

O time pediu **recomendação explícita** sobre HTTP vs gRPC no sync, com espaço para estudar o tema depois.

---

## Decisão

### 1. Comunicação síncrona (serviço → serviço)

**Padrão oficial: HTTP/REST + JSON** na rede interna (`vitalis_net`).

**Por que não gRPC agora (recomendação):**

| Critério | HTTP/REST + JSON | gRPC |
|---|---|---|
| Curva de aprendizado | Mais baixa (você já conhece APIs REST) | Mais alta (protobuf, codegen, tooling) |
| Contratos com front | OpenAPI natural | Front continua REST via Gateway de qualquer forma |
| Debug local | curl / Insomnia / logs texto | precisa tooling binário |
| Performance | Suficiente para demo, TCC e 10+ usuários | Melhor em alto throughput interno |
| Fit atual (chi + OpenAPI) | Direto | Camada extra |

**Regra:** sync interno = REST.  
**Futuro:** gRPC pode ser reavaliado (ADR novo) se houver gargalo real entre serviços — não é bloqueio do desenho atual.

Chamadas sync devem ser **curtas**, com timeout, e evitadas em cadeias longas (preferir evento quando for efeito colateral).

### 2. Comunicação assíncrona (eventos)

- Barramento oficial: **Redis Streams** (Redis já é padrão de plataforma no ADR-002).
- Todo serviço que **publica** evento usa **Transactional Outbox** obrigatório:
  1. grava estado de domínio + linha na `outbox_events` na **mesma** transação Postgres;
  2. worker publica no Redis Streams;
  3. marca/publica com at-least-once + consumidores idempotentes.

### 3. Clientes (web/mobile) ↔ backend

- Clientes **só** falam com o **`Vitalis-gateway`**.
- **Proibido** (no ambiente Docker/demo “oficial”): cliente chamar direto Identity, Billing, Clinical, etc.
- Inclui HTTP e **WebSocket**: o Gateway faz **upgrade/proxy** para o `Vitalis-comms`.

### 4. Webhooks de `* API Pagamentos`

- Entrada pública (ou exposta no Compose): **Gateway**.
- Gateway encaminha para **Billing** (rota interna).
- Billing valida assinatura HMAC / autenticação do provedor, aplica idempotência e atualiza ledger.

### 5. Envelope padrão de evento

Manter o padrão mínimo (pode evoluir depois com ADR de versionamento):

```json
{
  "id": "evt_...",
  "type": "payment.succeeded",
  "occurred_at": "2026-09-12T12:00:00Z",
  "producer": "Vitalis-billing",
  "version": 1,
  "correlation_id": "...",
  "data": {}
}
```

Campos obrigatórios: `id`, `type`, `occurred_at`, `producer`, `version`, `correlation_id`, `data`.

### 6. Contratos

- Fonte da verdade: repositório **`Vitalis-contracts`**
  - OpenAPI por serviço (HTTP)
  - Catálogo de eventos (AsyncAPI ou equivalente)
- Mudança de contrato = PR em contracts antes (ou junto) da implementação nos serviços.

---

## Critério de sucesso deste ADR

- [ ] Nenhum cliente aponta para porta de serviço de domínio (só Gateway `:8080`).
- [ ] WS do app passa pelo Gateway → Comms.
- [ ] Webhook de pagamento: Gateway → Billing com validação + idempotência.
- [ ] Serviços publicadores têm outbox + worker.
- [ ] Eventos publicados seguem o envelope padrão.
- [ ] Sync interno documentado como REST (sem gRPC obrigatório).

---

## Consequências

### Positivas

- Modelo simples de aprender e de documentar (OpenAPI + eventos).
- Gateway como único ponto de política (JWT, rate limit, CORS, WS).
- Outbox reduz “sumiço” de eventos após commit.

### Negativas / trade-offs

- REST interno não é o máximo de performance (aceitável no escopo).
- Redis Streams exige disciplina de consumer group / retry / DLQ simples.
- Gateway vira ponto crítico (mitigar com health, timeout, observação).

### Mitigações

- Timeouts e circuit breakers leves nas chamadas sync.
- Idempotência nos consumers (`event.id` único processado).
- Estudo posterior de gRPC sem reescrever o desenho agora.

---

## Alternativas consideradas

| Alternativa | Decisão |
|---|---|
| gRPC como sync padrão | Adiado — REST escolhido por clareza/OpenAPI/aprendizado |
| Cliente → Comms direto (WS) | Rejeitado — tudo pelo Gateway |
| Webhook direto no Billing | Rejeitado — entra pelo Gateway |
| Publicar evento sem outbox | Rejeitado — outbox obrigatório |
| Kafka desde o dia 1 | Desnecessário — Redis Streams suficiente no Docker local |

---

## Notas de estudo (para você)

- **HTTP sync:** “pergunto e espero resposta agora” (ex.: Clinical pergunta se assinatura está ativa).
- **Evento async:** “aviso que algo aconteceu” (ex.: `payment.succeeded` → Orders marca pago).
- **Outbox:** garante que o aviso só sai se o banco commitou o fato.
- **gRPC:** outro “idioma” de sync (binário/contratos `.proto`); útil depois, não bloqueia o Vitalis agora.

---

## Próximo

- AuthN/AuthZ (JWT, papéis, claims) → **ADR-004**.
