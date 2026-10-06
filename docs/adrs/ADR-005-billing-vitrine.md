# ADR-005 — Billing vitrine (PIX, cartão, carteira, ledger, split)

| Campo | Valor |
|---|---|
| Status | Aceito |
| Data | 2026-09-12 |
| Decisores | Equipe Vitalis |
| Relacionados | ADR-002 (stack), ADR-003 (webhooks/outbox), ADR-004 (auth) |

---

## Contexto

`Vitalis-billing` é o **serviço vitrine** do redesign: dinheiro, planos e rateio entre atores. Precisa ser:

- confiável (idempotência, ledger);
- Brasil-first (PIX + BRL);
- alinhado ao produto multi-ator (paciente, médico, farmácia, motoboy, plataforma);
- testável em Docker local **sem** depender de Stripe/PSP real no dia 1.

---

## Decisão

### 1. Meios de pagamento (entregável)

O Billing deve suportar **os três**:

| Meio | Uso típico |
|---|---|
| **PIX** | Assinaturas e pedidos (confirmação via webhook) |
| **Cartão** | Assinaturas recorrentes e checkout |
| **Carteira interna (Wallet)** | Saldo/créditos da Vitalis (top-up, abatimento, cashback/repasses internos) |

Moeda: **BRL**. Foco **Brasil-first**.

### 2. `* API Pagamentos` (adapter)

- **Começar com adapter fake/sandbox** que simula:
  - criação de cobrança PIX (QR / copia-cola / expiração);
  - cobrança cartão (aprovado/negado);
  - webhooks de confirmação/falha.
- Manter **porta (`PaymentProvider`)** estável para trocar por PSP real depois **sem** reescrever o domínio (orchestrator + ledger + split).
- Webhooks entram pelo **Gateway → Billing** (ADR-003).

### 3. Ledger (partidas dobradas)

- **Obrigatório em toda movimentação financeira** (não só assinaturas).
- Inclui: captura PIX/cartão, crédito/débito de carteira, split, settlement, estorno/refund.
- Dinheiro em `NUMERIC` (ADR-002); lançamentos append-oriented; saldo derivado do razão.
- Objetivo: auditabilidade e demo/TCC defensável (“cada centavo tem contrapartida”).

### 4. Split marketplace — regras de negócio

Há **dois contextos de split** (não misturar):

#### 4.1 Pedido farmácia / e-commerce (Orders + Delivery)

Destinatários do split:

- **Vitalis** (taxa plataforma)
- **Farmácia**
- **Motoboy**

**Médico NÃO participa** deste split.

#### 4.2 Consultas / telemedicina (Clinical + assinatura)

- O **médico recebe** rateio/comissão ligado à **assinatura/plano do paciente** e à realização da consulta (regra de negócio do Billing + eventos Clinical, ex. `appointment.completed`).
- Vitalis permanece com a parcela da plataforma.

Regras versionadas em `split_rules` (por produto, plano ou tipo de referência: `order` vs `appointment`/`subscription`).

### 5. Idempotência

- Header **`Idempotency-Key` obrigatório** em todo `POST` de:
  - cobrança (`payment-intents`);
  - assinatura / troca de plano;
  - operações sensíveis de carteira que geram movimento.
- Tabela `idempotency_keys` + dedupe de `webhook_events` (`provider + event_id`).

### 6. Assinaturas de plano

Opção **C** confirmada:

- Assinatura é **core** (gate do Clinical — plano ativo).
- Não é só pagamento avulso.
- Suportar: **assinar**, **upgrade**, **downgrade**, **cancelar** (e estados como `past_due` quando aplicável).

Pagamentos avulsos (ex.: pedido farmácia) coexistem com assinaturas.

### 7. State machines mínimas

- **PaymentIntent:** `created → pending → captured | failed | expired | cancelled` (+ `refunded`)
- **Subscription:** `active` (e correlatos) com troca de plano e cancelamento

(Detalhe visual já existe nos diagramas `billing-2.4-state`.)

---

## Critério de sucesso deste ADR

- [ ] PIX sandbox (fake) gera QR e confirma via webhook simulado.
- [ ] Cartão sandbox aprova/nega.
- [ ] Carteira interna credita/debita com lançamentos no ledger.
- [ ] Pedido farmácia gera split Vitalis + farmácia + motoboy (sem médico).
- [ ] Consulta/assinatura gera parcela ao médico conforme regra.
- [ ] `Idempotency-Key` + webhook duplicado não duplicam captura.
- [ ] Upgrade/downgrade/cancelamento de plano funcionam e refletem no gate Clinical.

---

## Consequências

### Positivas

- Billing rico o bastante para TCC/portfólio.
- Domínio desacoplado do PSP (porta + fake primeiro).
- Split alinhado ao produto real (médico ≠ delivery).

### Negativas / trade-offs

- Três meios + ledger + dois tipos de split = complexidade alta (aceita como vitrine).
- Adapter fake não prova integração bancária real (mitigar depois com PSP sandbox).

### Mitigações

- Testes de ledger (soma débitos = créditos).
- Contratos OpenAPI do Billing como primeiro contrato “duro”.
- Seeds de planos e percentuais de split no Compose.

---

## Alternativas consideradas

| Alternativa | Decisão |
|---|---|
| Só PIX | Rejeitado — cartão + carteira também |
| PSP real no dia 1 | Adiado — fake/sandbox primeiro, porta pronta |
| Ledger só em assinaturas | Rejeitado — toda movimentação |
| Médico no split do pedido farmácia | Rejeitado — médico só no fluxo clínico/assinatura |
| Sem upgrade/downgrade | Rejeitado — opção C |

---

## Encerramento do pacote ADR 001–005

Com este ADR, ficam fechadas as decisões estruturais de:

1. Microserviços + polyrepo  
2. Stack Go/Postgres/Redis/MinIO/Docker  
3. Comunicação HTTP + Redis Streams + Outbox + Gateway  
4. Auth JWT/RBAC  
5. Billing vitrine  

**Próximo pacote combinado:** OpenAPI/Eventos + Glossário + Matriz RBAC.
