# Arquitetura de Microserviços — Rede Médica D'Excelência Vitalis

Documento de referência da arquitetura-alvo da plataforma Vitalis: opinião sobre o tamanho do projeto, mapa de todos os serviços, clientes (web/mobile), comunicação, dados e plano de migração a partir do backend monolítico atual.

| Campo | Valor |
|---|---|
| Versão | 1.1 |
| Data | 2026-09-09 |
| Escopo | Ecossistema Vitalis (backend + todos os frontends) |
| Estado atual | Monólito modular (Express + MySQL) + múltiplos clientes |
| Estado-alvo | Microserviços em **Go + PostgreSQL** (6 núcleos + gateway + comunicação) |
| Stack backend-alvo | **Golang**, **PostgreSQL**, Redis (cache/fila leve), Docker |

---

## 1. Opinião (tamanho do projeto × microserviços)

### Veredito curto

**Não reescreva tudo de uma vez em 12–15 microserviços.** Para o tamanho atual da Vitalis (TCC / time pequeno), com a decisão de stack **Go + PostgreSQL**, o caminho certo é:

1. **Curto prazo:** redesenhar em **Go** como **microserviços pragmáticos** (não copiar o monólito Node 1:1), reaproveitando os *bounded contexts* já mapeados.
2. **Médio prazo:** subir **6 serviços de domínio** + **API Gateway** + **Communication Hub**, todos com **Postgres próprio** (database-per-service).
3. **Destaque técnico:** o serviço de **Pagamentos (`vitalis-billing`)** como vitrine de engenharia (ledger, PIX, split, idempotência).
4. **Longo prazo:** só fatiar mais se houver dor real (escala, time paralelo, deploy independente, LGPD/isolamento).

### Por que não “microserviços completos” agora

| Fator | Situação Vitalis | Impacto |
|---|---|---|
| Time | Poucas pessoas / TCC | Cada serviço = mais CI, Docker, deploy, observabilidade |
| Backend atual | Domínios em pastas, **um MySQL**; alvo é **Go + Postgres por serviço** | Sem DB próprio por serviço vira “monólito distribuído” (pior) |
| Complexidade de negócio | Consulta ↔ pagamento ↔ notificação ↔ entrega | Transações distribuídas e consistência eventual |
| Frontends | Já são muitos clientes | O gargalo não é só o backend; é contrato de API estável |
| Defesa de TCC | Banca valoriza **clareza e decisão justificada** | “Microserviços porque é moderno” enfraquece; “extraímos X porque Y” fortalece |

### O que recomendo para a Vitalis

**Arquitetura-alvo pragmática: 6 serviços de domínio + 2 de plataforma.**

Isso casa com o que o backend já tem (`auth`, `user`, `medical`, `pharmacy`, `orders`, `payment`, `delivery`, `communication`, `support`, `system`, `search`, `validation`), mas **agrupa** o que ainda não merece processo/deploy separados.

```
                    ┌─────────────────────────┐
                    │      API Gateway        │
                    │  (authN, rate limit,    │
                    │   roteamento, BFF ops)  │
                    └───────────┬─────────────┘
                                │
        ┌───────────┬───────────┼───────────┬───────────┬───────────┐
        ▼           ▼           ▼           ▼           ▼           ▼
   Identity    Clinical     Commerce    Fulfillment  Billing   Platform
   Service     Service      Service      Service     Service   Support
        │           │           │           │           │           │
        └───────────┴───────────┴─────┬─────┴───────────┴───────────┘
                                      ▼
                         Communication Hub
                    (notificações, chat, eventos)
```

---

## 2. Visão do ecossistema atual (as-is)

### 2.1 Repositórios da organização

| Repositório | Papel | Stack típica |
|---|---|---|
| `Vitalis-Backend` | API única (monólito modular) | Node.js, Express, MySQL, Socket.IO, JWT, Stripe, Firebase |
| `Vitalis-web-paciente` | Portal do paciente | React, TypeScript, Vite, Zustand, Playwright |
| `Vitalis-web-profissional` | Portal do médico/profissional | React, TypeScript, Cypress/Playwright |
| `Vitalis-web-farmacias` | Portal da farmácia parceira | React, TypeScript |
| `Vitalis-web-administrador` | Painel administrativo | React, TypeScript |
| `Vitalis-mobile-paciente` | App paciente | React Native / Expo |
| `Vitalis-mobile-motoboy` | App entregador | React Native / Expo |

### 2.2 Módulos já existentes no backend (base dos bounded contexts)

| Pasta em `src/routes` | Responsabilidade atual |
|---|---|
| `auth` | Login, tokens, sessão |
| `user` | Usuário, cadastro, endereço, avaliação |
| `validation` | Validação de documentos / dados |
| `medical` | Agenda, consulta, médico, paciente clínico, triagem, videochamada, prescrição, especialidades |
| `pharmacy` | Farmácia, produto, categoria |
| `orders` | Pedido, itens, chat do pedido |
| `payment` | Pagamento, assinaturas, carteira, comissão |
| `delivery` | Entrega, motoboy, rastreamento, veículo |
| `communication` | Chat, notificação |
| `support` | Tickets, artigos de ajuda |
| `search` | Busca transversal |
| `system` | Health, upload, PDF, logs, relatórios, integração |

**Conclusão as-is:** a Vitalis **já pensa em domínio**. O salto para microserviços é principalmente **deploy + banco + contratos**, não inventar módulos do zero.

---

## 3. Arquitetura-alvo (to-be)

### 3.1 Princípios

1. **Um domínio = um serviço Go = um database PostgreSQL próprio.**
2. **Comunicação síncrona** (REST) para leitura/comando imediato; **assíncrona** (eventos + outbox) para efeitos colaterais.
3. **API Gateway** é a única porta pública dos clientes.
4. **Sem shared database** entre serviços de domínio (anti-padrão).
5. **Identidade centralizada** (JWT RS256 emitido pelo Identity Service).
6. **LGPD por domínio:** cada serviço é dono do seu dado sensível, com trilha de auditoria.
7. **Billing como referência de qualidade:** idempotência, ledger e adapters de PSP em todos os fluxos de dinheiro.

### 3.2 Catálogo de serviços (plataforma)

| # | Serviço | Nome técnico | Prioridade | Extração sugerida |
|---|---|---|---|---|
| 0 | API Gateway | `vitalis-gateway` | P0 | Imediato no redesenho |
| 1 | Identity & Access | `vitalis-identity` | P0 | 1º a extrair |
| 2 | Clinical (Saúde) | `vitalis-clinical` | P0 | 2º |
| 3 | Commerce (Farmácia catálogo) | `vitalis-commerce` | P1 | 3º |
| 4 | Fulfillment (Pedido + Entrega) | `vitalis-fulfillment` | P1 | 4º |
| 5 | Billing (Pagamentos — **vitrine**) | `vitalis-billing` | P0 | PIX + ledger + split; priorizar cedo |
| 6 | Communication Hub | `vitalis-comms` | P1 | junto com chat/WS |
| 7 | Platform Support | `vitalis-support` | P2 | por último |
| — | Search (opcional) | `vitalis-search` | P3 | só se busca cruzada doer |

> **Nota de tamanho:** `orders` + `delivery` ficam juntos em **Fulfillment** no início. Separar só quando volume de entrega ou time de logística justificar.

---

## 4. Detalhamento de todos os serviços

### 4.0 `vitalis-gateway` — API Gateway

**Missão:** ponto único de entrada HTTPS para web/mobile.

**Responsabilidades**
- Roteamento para microserviços
- Terminação TLS
- Rate limiting / WAF básico
- Validação de JWT (assinatura/expiração) e repasse de claims
- CORS, request ID, observabilidade (trace)
- (Opcional) BFF por cliente: `/bff/paciente`, `/bff/profissional`

**Não faz**
- Regra de negócio
- Persistência de domínio

**Clientes que consomem**
- Todos os frontends e apps mobile

**Dependências**
- Identity (JWKS / chave pública)
- Demais serviços via rede interna

---

### 4.1 `vitalis-identity` — Identidade, usuários e acesso

**Missão:** quem é o usuário, como autentica, que papéis tem.

**Une os módulos atuais:** `auth` + `user` (cadastro/endereço) + parte de `validation` (documentos de identidade).

**Responsabilidades**
- Registro (paciente, profissional, farmácia, admin, motoboy)
- Login / logout / refresh token
- Perfis e papéis (`paciente`, `medico`, `farmacia`, `admin`, `motoboy`)
- Endereços do usuário
- Recuperação de senha
- Validação cadastral de documentos (CPF, CRM, etc. — validação **cadastral**, não clínica)
- Emissão de JWT / OIDC

**APIs (exemplos)**
- `POST /auth/login`
- `POST /auth/refresh`
- `POST /auth/logout`
- `POST /usuarios`
- `GET/PUT /usuarios/me`
- `POST /auth/esqueci-senha`
- `POST /validacao/documentos`

**Dados que possui**
- Credenciais (hash), perfil básico, papéis, endereços, status de verificação

**Eventos publicados**
- `user.registered`
- `user.updated`
- `user.role_changed`
- `user.verified`

**Consumidores típicos**
- Clinical (precisa saber paciente/médico)
- Billing (assinatura ligada ao usuário)
- Fulfillment (endereço de entrega)
- Comms (destino de notificação)

**Clientes**
- Todos (login é universal)

---

### 4.2 `vitalis-clinical` — Domínio clínico / telemedicina

**Missão:** jornada de saúde: triagem → agendamento → consulta → videochamada → prescrição → documentos.

**Une os módulos atuais:** `medical` (agenda, consulta, doctors, specialties, triagem, videochamada, prescrição, documento médico, paciente clínico).

**Responsabilidades**
- Especialidades e cadastro profissional clínico
- Disponibilidade e agenda
- Agendamento e ciclo de vida da consulta
- Triagem online
- Videochamada / sala WebRTC (sinalização; mídia pode ser provedor externo)
- Prescrições e documentos médicos
- Histórico clínico do paciente **no contexto Vitalis**
- Avaliações de atendimento (ou consome de Identity — preferir ficar aqui se for avaliação clínica)

**APIs (exemplos)**
- `GET /especialidades`
- `GET /medicos`
- `GET/POST /disponibilidade`
- `POST /consultas`
- `GET /consultas/:id`
- `POST /triagens`
- `POST /videochamadas/sala`
- `POST /prescricoes`
- `GET /pacientes/:id/historico`

**Dados que possui**
- Consultas, slots, triagens, prescrições, metadados de videochamada, vínculos paciente–profissional

**Eventos publicados**
- `appointment.created|confirmed|cancelled|completed`
- `triage.completed`
- `prescription.issued`
- `video.session.started|ended`

**Eventos consumidos**
- `user.registered` / `user.verified` (profissional liberado)
- `subscription.activated|cancelled` (gate de plano — ou consulta síncrona ao Billing)

**Clientes**
- `Vitalis-web-paciente`
- `Vitalis-web-profissional`
- `Vitalis-mobile-paciente`
- `Vitalis-web-administrador` (supervisão)

**LGPD:** dados sensíveis de saúde — criptografia em repouso, auditoria, minimização, retenção definida.

---

### 4.3 `vitalis-commerce` — Catálogo farmácia

**Missão:** farmácias parceiras, catálogo de produtos e categorias (antes do pedido).

**Une os módulos atuais:** `pharmacy`.

**Responsabilidades**
- CRUD de farmácia parceira
- Produtos, estoque lógico, categorias
- Busca de produtos **dentro do catálogo**
- Imagens / metadados de produto

**APIs (exemplos)**
- `GET/POST /farmacias`
- `GET /farmacias/:id/produtos`
- `GET/POST /produtos`
- `GET /categorias`

**Dados que possui**
- Farmácias, produtos, categorias, preços de vitrine

**Eventos publicados**
- `product.updated`
- `pharmacy.activated|suspended`
- `stock.changed` (se estoque for aqui)

**Eventos consumidos**
- `order.placed` (baixa de estoque reservado — ou Fulfillment reserva e Commerce confirma)

**Clientes**
- `Vitalis-web-paciente` / mobile paciente (vitrine)
- `Vitalis-web-farmacias`
- `Vitalis-web-administrador`

---

### 4.4 `vitalis-fulfillment` — Pedidos + entrega

**Missão:** do carrinho/checkout do medicamento até a entrega na porta.

**Une os módulos atuais:** `orders` + `delivery` (+ chat do pedido).

**Responsabilidades**
- Pedido e itens
- Status do pedido (`criado`, `pago`, `separando`, `saiu_entrega`, `entregue`, `cancelado`)
- Alocação de motoboy / veículo
- Rastreamento
- Chat operacional do pedido (paciente ↔ farmácia ↔ motoboy)
- Solicitações de entrega

**APIs (exemplos)**
- `POST /pedidos`
- `GET /pedidos/:id`
- `PATCH /pedidos/:id/status`
- `POST /entregas`
- `GET /rastreamento/:pedidoId`
- `GET/POST /motoboys/disponiveis`
- `POST /pedidos/:id/chat/mensagens`

**Dados que possui**
- Pedidos, itens, entregas, rastreamento, vínculo motoboy, mensagens do pedido

**Eventos publicados**
- `order.placed|paid|cancelled|delivered`
- `delivery.assigned|started|completed`
- `tracking.updated`

**Eventos consumidos**
- `payment.succeeded|failed`
- `product.updated` / `stock.changed`
- `user.updated` (endereço)

**Clientes**
- Paciente (web/mobile)
- `Vitalis-web-farmacias`
- `Vitalis-mobile-motoboy`
- Admin

**Evolução futura:** separar `vitalis-orders` e `vitalis-delivery` quando a operação logística crescer.

---

### 4.5 `vitalis-billing` — Pagamentos (serviço vitrine)

**Missão:** orquestrar **todo o dinheiro** da Vitalis com confiabilidade de sistema financeiro: cobranças avulsas, assinaturas de plano, carteira, split marketplace (plataforma / médico / farmácia / motoboy), estornos e auditoria imutável.

**Une os módulos atuais:** `payment` (pagamento, assinaturas, assinaturas-unificadas, carteira, comissão, pagamento automático).

**Por que este serviço é o diferencial técnico**
- Saúde + marketplace multi-ator exige **repasse (split)** e rastreabilidade — não basta “chamar Stripe e pronto”.
- No Brasil, **PIX** é obrigatório na prática; cartão e boleto entram como canais.
- Falhas de rede e webhooks duplicados exigem **idempotência** e **ledger de partidas dobradas**.
- Em Go + Postgres: concurrency, transactions e tipagem forte encaixam perfeitamente em domínio financeiro.

#### Responsabilidades

| Capacidade | Descrição |
|---|---|
| Payment Orchestrator | Fluxo único de cobrança independente do provedor (PIX, cartão, boleto) |
| Provider Adapters | Adaptadores pluggáveis: Stripe, Mercado Pago, Pagar.me, PSP PIX — troca sem reescrever domínio |
| Assinaturas | Planos Vitalis, upgrade/downgrade, pró-rata, retry de fatura, dunning |
| Carteira (Wallet) | Saldo interno do usuário/parceiro para créditos, cashback e saques |
| Split / Marketplace | Percentuais e valores fixos para Vitalis, profissional, farmácia, entregador |
| Ledger | Livro-razão append-only (débito/crédito); saldo derivado, nunca “UPDATE solto” |
| Webhooks | Recepção assinada, dedupe por `event_id`, processamento at-least-once |
| Estorno / chargeback | Estorno parcial/total com lançamentos compensatórios no ledger |
| Recibos | Comprovantes financeiros (não clínicos) e exportação para admin |

#### Arquitetura interna (Go)

```
vitalis-billing/
├── cmd/api/                 # HTTP API
├── cmd/worker/              # consumers de fila + outbox publisher
├── internal/
│   ├── domain/              # PaymentIntent, Subscription, LedgerEntry, SplitRule
│   ├── app/                 # use cases (Charge, ConfirmPIX, RenewSub, SettleSplit)
│   ├── port/                # interfaces: PaymentProvider, EventBus, Clock
│   ├── adapter/
│   │   ├── http/            # chi/fiber handlers + middleware idempotency
│   │   ├── postgres/        # repositories + advisory locks
│   │   ├── stripe/          # adapter cartão/internacional
│   │   ├── mercadopago/     # adapter PIX + BR
│   │   └── redis/           # idempotency cache / rate limit
│   └── outbox/              # transactional outbox
└── migrations/              # golang-migrate
```

**Padrões obrigatórios neste serviço**
1. **Idempotency-Key** em todo `POST` de cobrança (header); unique no Postgres.
2. **Transactional Outbox** na mesma TX do ledger → worker publica eventos.
3. **State machine** do `PaymentIntent`: `created → pending → authorized → captured | failed | expired → refunded`.
4. **Ledger de partidas dobradas** (toda movimentação gera ≥2 lançamentos; Σ débitos = Σ créditos).
5. **Saga/orquestração leve** com Fulfillment/Clinical via eventos (não 2PC).

#### Meios de pagamento (Brasil-first)

| Meio | Uso na Vitalis | Notas |
|---|---|---|
| **PIX** | Assinatura, consulta avulsa, pedido farmácia | QR/copia-cola, expiração curta, confirmação via webhook |
| **Cartão** | Assinaturas recorrentes, checkout | Tokenização no PSP; Vitalis guarda só `payment_method_token` |
| **Boleto** | Planos anuais / B2B farmácia | Prazo longo; status `pending` até compensar |
| **Carteira Vitalis** | Créditos, cashback, abatimento parcial | Débito no ledger interno antes/alongside PSP |

#### Modelo de split (exemplo)

Pedido farmácia R$ 100,00 com entrega:

| Destinatário | Tipo | Exemplo |
|---|---|---|
| Farmácia | seller | R$ 82,00 |
| Motoboy | delivery | R$ 10,00 |
| Vitalis | platform fee | R$ 8,00 |

Consulta telemedicina R$ 150,00:

| Destinatário | Tipo | Exemplo |
|---|---|---|
| Profissional | professional | R$ 120,00 |
| Vitalis | platform fee | R$ 30,00 |

Regras versionadas em `split_rules` (por produto, plano ou campanha). Liquidação (`settlement`) pode ser D+0 (carteira) ou D+N (payout PSP).

#### APIs (exemplos)

**Cobrança**
- `POST /v1/payment-intents` — cria intenção (valor, moeda BRL, meio, `reference_type`/`reference_id`)
- `GET /v1/payment-intents/:id`
- `POST /v1/payment-intents/:id/cancel`
- `POST /v1/payment-intents/:id/refund`

**PIX**
- `POST /v1/payment-intents/:id/pix` — retorna QR / EMV / `expires_at`
- `GET /v1/payment-intents/:id/pix/status`

**Assinaturas**
- `POST /v1/subscriptions` — assinar / trocar plano (`upgrade|downgrade|troca`)
- `GET /v1/subscriptions/me`
- `POST /v1/subscriptions/:id/cancel`
- `GET /v1/invoices` / `GET /v1/invoices/:id`

**Carteira & split**
- `GET /v1/wallets/me`
- `POST /v1/wallets/topup` (opcional)
- `GET /v1/settlements`
- `GET /v1/commissions`

**Provedor**
- `POST /v1/webhooks/{provider}` — Stripe / MP / etc. (assinatura HMAC)

Headers críticos: `Idempotency-Key`, `Authorization`, `X-Request-Id`.

#### Dados (PostgreSQL)

| Tabela | Papel |
|---|---|
| `payment_intents` | Intenção de pagamento + estado |
| `payment_attempts` | Tentativas / retentativas por provedor |
| `payment_methods` | Tokens/métodos salvos (sem PAN) |
| `subscriptions` / `subscription_items` | Planos ativos |
| `invoices` / `invoice_lines` | Faturas |
| `wallets` / `wallet_holds` | Saldo e reservas |
| `ledger_accounts` | Contas do razão (user, platform, tax…) |
| `ledger_entries` / `ledger_transactions` | Lançamentos imutáveis |
| `split_rules` / `split_allocations` | Regras e rateios |
| `settlements` / `payouts` | Liquidação a parceiros |
| `webhook_events` | Inbox deduplicada |
| `outbox_events` | Publicação confiável |
| `idempotency_keys` | Dedupe de comandos |

**Convenções Postgres**
- `NUMERIC(12,2)` para dinheiro (nunca `float`)
- `timestamptz` em UTC
- Soft constraints + `CHECK` de status
- Índices únicos: `(idempotency_key)`, `(provider, provider_event_id)`
- Particionamento futuro de `ledger_entries` por mês (se volume crescer)

#### Eventos publicados

- `payment.intent.created`
- `payment.succeeded` / `payment.failed` / `payment.expired`
- `payment.refunded`
- `subscription.activated` / `subscription.changed` / `subscription.cancelled` / `subscription.past_due`
- `wallet.credited` / `wallet.debited`
- `split.allocated`
- `settlement.completed`
- `commission.calculated`

#### Eventos consumidos

- `order.placed` → cria `PaymentIntent` (farmácia)
- `appointment.completed` → comissão / split clínico (se pós-pago)
- `user.registered` → provisiona contas no ledger / wallet
- `delivery.completed` → libera hold do motoboy no settlement

#### Fluxos de destaque (demo TCC)

1. **Assinar plano via PIX**  
   Paciente → `POST /subscriptions` → `PaymentIntent` PIX → QR → webhook `approved` → ledger + `subscription.activated` → Clinical libera gates.

2. **Pedido farmácia com split**  
   Fulfillment `order.placed` → Billing cobra → sucesso → `split.allocated` → carteiras farmácia/motoboy/plataforma → Fulfillment avança status.

3. **Falha e retentativa idempotente**  
   Mesmo `Idempotency-Key` + webhook duplicado → uma única captura no ledger.

#### Segurança financeira

- Nunca armazenar PAN/CVV — só token do PSP
- Webhooks com verificação de assinatura + IP allowlist quando possível
- Secrets em env/vault; rotação de chaves de webhook
- Auditoria: quem iniciou estorno (admin) com motivo
- Isolamento de rede: Billing não expõe DB; só API interna + webhooks públicos no Gateway

#### SLOs sugeridos (para monografia)

| Métrica | Alvo |
|---|---|
| Confirmação PIX (p95 após webhook) | < 2s processamento interno |
| Cobrança cartão síncrona (p95) | < 3s (excluindo latência PSP) |
| Duplicidade de captura | 0 (garantia por idempotência + unique) |
| Outbox lag | < 5s em operação normal |

**Clientes:** paciente (planos/checkout), profissional/farmácia/motoboy (repasse), admin (estornos/settlements).

---

### 4.6 `vitalis-comms` — Communication Hub

**Missão:** tempo real e fan-out de notificações.

**Une os módulos atuais:** `communication` (chat genérico + notificação) e a parte WS/Socket.IO do monólito.

**Responsabilidades**
- Push / e-mail / in-app notifications
- Chat clínico (paciente ↔ médico) — *diferente* do chat de pedido (que pode ficar no Fulfillment ou ser canal deste hub)
- Presença online
- Fan-out de eventos de domínio para clientes conectados
- Templates de mensagem

**APIs / canais**
- `GET /notificacoes`
- `PATCH /notificacoes/:id/lida`
- `WS /realtime` (Gorilla WebSocket / nhooyr — Go nativo)
- `POST /chat/salas` / `POST /chat/mensagens` (se chat clínico centralizado aqui)

**Dados que possui**
- Notificações, salas/mensagens (se ownership for deste serviço), preferências de canal

**Eventos consumidos (quase todos)**
- `appointment.*`, `order.*`, `payment.*`, `delivery.*`, `prescription.issued`

**Eventos publicados**
- `notification.sent`
- `chat.message.created`

**Clientes**
- Todos os apps com sininho / chat / tempo real

---

### 4.7 `vitalis-support` — Suporte e conteúdo de ajuda

**Missão:** atendimento operacional e base de conhecimento.

**Une os módulos atuais:** `support` (+ parte de artigos do paciente).

**Responsabilidades**
- Tickets e comentários
- Artigos da central de ajuda
- SLA básico / filas para admin

**APIs (exemplos)**
- `POST /tickets`
- `GET /tickets/:id`
- `POST /tickets/:id/comentarios`
- `GET /artigos-ajuda`

**Dados que possui**
- Tickets, comentários, artigos

**Eventos publicados**
- `ticket.created|resolved`

**Clientes**
- Paciente, profissional, farmácia, admin

**Nota TCC:** pode começar como módulo interno do Gateway e só virar serviço depois.

---

### 4.8 Serviços de plataforma transversais (opcionais)

| Serviço | Quando criar | Função |
|---|---|---|
| `vitalis-search` | Busca unificada médico+produto+artigo ficar cara | Indexação (OpenSearch/Elastic) |
| `vitalis-files` | Upload/PDF crescer | S3-compatible, URLs assinadas (sai de `system/upload` e `pdf`) |
| `vitalis-audit` | Exigência forte de LGPD/compliance | Log imutável de acesso a dado clínico |
| `vitalis-reporting` | BI/admin pesado | Relatórios assíncronos (sai de `system/relatorio`) |

Enquanto forem pequenos, permanecem como **bibliotecas/módulos** dentro de Clinical/Billing/Support — não como microserviços.

---

## 5. Clientes da plataforma (não são microserviços)

| Cliente | Consome principalmente |
|---|---|
| Web Paciente | Identity, Clinical, Commerce, Fulfillment, Billing, Comms, Support |
| Web Profissional | Identity, Clinical, Billing, Comms, Support |
| Web Farmácias | Identity, Commerce, Fulfillment, Billing, Comms |
| Web Administrador | Todos (leitura/administração) |
| Mobile Paciente | Mesmo do web paciente (BFF recomendado) |
| Mobile Motoboy | Identity, Fulfillment, Comms |

### Papéis (RBAC)

| Papel | Serviços críticos |
|---|---|
| `paciente` | Clinical, Commerce, Fulfillment, Billing |
| `medico` | Clinical, Billing (comissões), Comms |
| `farmacia` | Commerce, Fulfillment |
| `motoboy` | Fulfillment |
| `admin` | Support + admin APIs em todos |

---

## 6. Comunicação entre serviços

### 6.1 Síncrona (REST interno)

Usar quando a resposta é necessária na hora:

- Gateway → qualquer serviço
- Clinical → Billing (`GET assinatura ativa?`) *ou* cache + evento
- Fulfillment → Commerce (`reservar estoque?`)
- Fulfillment → Billing (`cobrar pedido`)

Preferir **timeout curto + circuit breaker**. Evitar cadeias longas (A→B→C→D).

### 6.2 Assíncrona (eventos)

Barramento sugerido (escolha uma e mantenha simples no TCC):

- **Fase 1:** fila leve (Redis Streams / RabbitMQ)
- **Fase 2:** Kafka se volume/auditoria exigir

**Contrato de evento (exemplo)**

```json
{
  "id": "evt_01J...",
  "type": "appointment.confirmed",
  "occurred_at": "2026-09-09T22:00:00Z",
  "producer": "vitalis-clinical",
  "version": 1,
  "data": {
    "appointment_id": "c_123",
    "patient_id": "u_1",
    "doctor_id": "u_9"
  }
}
```

### 6.3 Tempo real

- Clientes conectam só no **Comms** (ou Gateway faz proxy WS → Comms).
- Domínios **não** abrem WebSocket direto para o browser.

---

## 7. Dados e ownership

| Serviço | Store | Observação |
|---|---|---|
| Identity | **PostgreSQL** | Credenciais isoladas; `pgcrypto` / hash Argon2id no app |
| Clinical | **PostgreSQL** | Dados clínicos + auditoria; schemas rígidos |
| Commerce | **PostgreSQL** | Catálogo e estoque lógico |
| Fulfillment | **PostgreSQL** + Redis | Redis para tracking/presença de entrega |
| Billing | **PostgreSQL** (+ Redis opcional) | Ledger, idempotência, outbox; Redis só cache/lock leve |
| Comms | **PostgreSQL** + Redis | Redis pub/sub / presença; Postgres histórico |
| Support | **PostgreSQL** | Tickets e artigos |

**Proibido no alvo:** todos os serviços escrevendo nas mesmas tabelas do monólito legado.

**Durante a migração:** um cluster Postgres com **database por serviço** (`vitalis_identity`, `vitalis_billing`, …) é o padrão. Schema compartilhado só como etapa temporária e documentada.

**Convenção Go:** acesso via `database/sql` + `pgx`, migrations com `golang-migrate` ou `goose`.

---

## 8. Segurança e LGPD (obrigatório na Vitalis)

1. **Classificação de dados:** clínico > financeiro > cadastral > operacional.
2. **Clinical** e **Billing** com acesso restrito e auditoria.
3. Tokens: preferir **httpOnly cookie** (web) ou storage seguro (mobile); evitar JWT longo em `localStorage` em produção.
4. Segredos só em vault/CI; **nunca** commit de `.env` com credenciais.
5. Consentimento e base legal documentados por fluxo (consulta, receita, entrega).
6. Direito ao esquecimento: processo cross-service (orquestração), não `DELETE` cego em um banco só.

---

## 9. Plano de migração (strangler / rewrite em Go)

O legado Node/MySQL permanece como referência de regras até os serviços Go assumirem o tráfego. Novos serviços **nascem em Go + Postgres**, não como cópia linha a linha do Express.

### Fase 0 — Fundação (1–2 sprints)

- **Polyrepo:** um repositório Go por serviço (`Vitalis-identity`, `Vitalis-billing`, …) com layout `cmd/` + `internal/`
- Docker Compose da plataforma em `Vitalis-infra`; contratos em `Vitalis-contracts`
- Contratos OpenAPI por serviço
- Observabilidade: `otel` + logs estruturados (`slog`)
- Remover secrets do git legado; padronizar `.env.example`

### Fase 1 — Gateway + Identity (Go)

- `vitalis-gateway` (Go: chi/echo + proxy) ou Traefik/Caddy na frente
- `vitalis-identity` com JWT/JWKS
- Frontends autenticam no Identity novo
- Legado valida JWT do Identity (ponte)

### Fase 2 — Billing (vitrine — priorizar cedo)

- Implementar `vitalis-billing` completo (orchestrator + PIX + ledger + split)
- Clinical/Fulfillment (ainda legado ou stubs) consomem `subscription.*` / `payment.*`
- Demo TCC: assinar plano via PIX + pedido com split

### Fase 3 — Clinical

- `vitalis-clinical` em Go (agenda, consulta, triagem, video sinalização, prescrição)
- Gates de plano consultam Billing (sync curto) ou cache de eventos

### Fase 4 — Commerce + Fulfillment

- Catálogo e pedidos/entrega em Go
- App motoboy e portal farmácia estabilizam contratos

### Fase 5 — Comms + Support

- WebSocket Go + notificações
- Tickets no Support

### Fase 6 — Desligar monólito Node

- Adapter legado removido; só serviços Go em produção acadêmica/demo

```
[Clientes] → [Gateway Go] → [Identity]
                          → [Billing]   ★ vitrine
                          → [Clinical]
                          → [Commerce]
                          → [Fulfillment]
                          → [Comms]
                          → [Support]
               ↘ (temporário) [Monólito Node legado]
```

---

## 10. Mapa rápido: módulo legado → serviço novo

| Módulo atual (`Vitalis-Backend`) | Serviço-alvo |
|---|---|
| `auth` | `vitalis-identity` |
| `user` | `vitalis-identity` |
| `validation` | `vitalis-identity` (docs) / Clinical se for validação clínica |
| `medical` | `vitalis-clinical` |
| `pharmacy` | `vitalis-commerce` |
| `orders` | `vitalis-fulfillment` |
| `delivery` | `vitalis-fulfillment` |
| `payment` | `vitalis-billing` |
| `communication` | `vitalis-comms` |
| `support` | `vitalis-support` |
| `search` | `vitalis-search` (opcional) ou query nos donos |
| `system` (health) | cada serviço + gateway |
| `system` (upload/pdf) | `vitalis-files` (opcional) |
| `system` (relatório/log) | `vitalis-reporting` / `vitalis-audit` (opcional) |

---

## 11. Stack oficial (Go + PostgreSQL)

### 11.1 Padrão por serviço

| Camada | Escolha | Motivo |
|---|---|---|
| Linguagem | **Go 1.22+** | Performance, deploy simples, ótimo para APIs e workers |
| HTTP | **chi** ou **echo** (padrão do time) | Leve, middleware claro; Gateway e APIs |
| DB | **PostgreSQL 16** (1 database / serviço) | ACID, JSONB, constraints, ledger financeiro |
| Driver | **pgx** + `database/sql` | Performance e pool |
| Migrations | **golang-migrate** ou **goose** | Versionamento de schema |
| Cache / fila leve | **Redis** | Idempotency TTL, presença WS, streams |
| Auth | JWT (RS256) emitido pelo Identity | JWKS no Gateway |
| Tempo real | **gorilla/websocket** ou **nhooyr/websocket** | Comms em Go (sem Socket.IO no backend novo) |
| Pagamentos | **Orchestrator + adapters** (Stripe e/ou Mercado Pago / PIX) | Domínio Vitalis dono do fluxo; PSP só execução |
| Push | Firebase Admin (via Comms) | Já conhecido no ecossistema |
| Observabilidade | OpenTelemetry + `slog` + Prometheus | Padrão cloud-native |
| Containers | Docker Compose (TCC) → K8s depois | Demo reprodutível |
| Contratos | OpenAPI 3 + AsyncAPI (eventos) | Contrato estável com frontends |

### 11.2 Stack específica do `vitalis-billing`

| Peça | Tecnologia |
|---|---|
| API + Worker | Dois binários Go (`cmd/api`, `cmd/worker`) |
| Dinheiro | `shopspring/decimal` ou `NUMERIC` no Postgres (evitar `float64`) |
| Idempotência | Header + tabela `idempotency_keys` |
| Outbox | Tabela `outbox_events` + worker |
| PIX / BR | Adapter Mercado Pago (ou PSP PIX escolhido) |
| Cartão | Adapter Stripe e/ou MP |
| Testes | `testify` + testcontainers (Postgres) + contratos de webhook gravados |

### 11.3 Frontends (inalterados na essência)

| Cliente | Stack |
|---|---|
| Webs | React + TypeScript + Vite |
| Mobiles | React Native / Expo |
| Integração | REST via Gateway; WS só no Comms |

### 11.4 Por que Go + Postgres nesta reescrita

- **Go:** concurrency nativa para webhooks, workers e WebSocket; binário único por serviço.
- **Postgres:** encaixa em Identity (relacional), Clinical (integridade), e especialmente **Billing** (ledger, constraints, transações).
- **Corte limpo com o legado Node/MySQL:** o monólito vira especificação viva, não a base do código novo.

---

## 12. Critérios de sucesso (para TCC e para o produto)

- [ ] Diagrama C4 (contexto + containers) na monografia — stack **Go + Postgres** explícita
- [ ] Cada serviço com responsabilidade e dono de dados explícitos
- [ ] Pelo menos **Identity + Billing** (e de preferência Clinical) rodando em Go
- [ ] **Billing vitrine:** PIX + ledger + split + idempotência demonstráveis
- [ ] Fluxo E2E: login → assinatura PIX → agendar → teleconsulta → receita
- [ ] Fluxo E2E farmácia: catálogo → pedido → pagamento com split → entrega
- [ ] Decisão arquitetural escrita: *por que N serviços, e por que Go/Postgres*
- [ ] Riscos: consistência eventual, webhooks duplicados, operação — com mitigações

---

## 13. Resumo executivo

A Vitalis **já é uma plataforma multi-ator** (paciente, médico, farmácia, admin, motoboy). Isso **justifica** serviços por domínio.

A reescrita-alvo usa **Golang + PostgreSQL**, com microserviços pragmáticos:

> **Gateway + Identity + Clinical + Commerce + Fulfillment + Billing + Comms (+ Support)**  
> sendo **`vitalis-billing` o serviço mais rico**: orquestração de pagamento, PIX, cartão, carteira, ledger de partidas dobradas, split marketplace e outbox.

O monólito Node/MySQL permanece como mapa de regras até ser desligado — não como stack do futuro.

---

## 14. Próximos passos sugeridos

1. Congelar esta stack (Go + Postgres) em ADR na monografia.
2. Desenhar diagramas: contexto C4 + sequência PIX assinatura + sequência split pedido.
3. Começar pela **Fase 2 antecipável:** esqueleto Go do `vitalis-billing` (PaymentIntent + ledger + webhook fake).
4. Em paralelo: `vitalis-identity` (JWT) + Gateway.
5. Definir OpenAPI do Billing como contrato-piloto com o web paciente.

---

*Documento v1.1 — alinhado a Go + PostgreSQL, com `vitalis-billing` como serviço vitrine. Baseado na organização `Rede-Medica-D-Excelencia-Vitalis` e nos módulos reais do `Vitalis-Backend`.*