# Guia draw.io — Visão macro Vitalis (1.1 → 1.6)

Você desenha no **diagrams.net / draw.io**. Este arquivo é o roteiro de texto para copiar/colar.

Arquivo sugerido: `docs/diagramas/Vitalis-Visao-Macro.drawio`  
Abas: `01-Contexto` · `02-Containers` · `03-Rede-Compose` · `04a-Seq-PIX` · `04b-Seq-Pedido` · `05-Matriz-Clientes` · `06-Matriz-Eventos`

Canvas de apoio (Cursor): abra ao lado do chat enquanto desenha.

---

## Setup visual (todas as abas)

| Elemento | Forma no draw.io | Cor sugerida |
|---|---|---|
| Ator humano | Person / stickman (C4) ou retângulo arredondado | Cinza claro |
| Sistema Vitalis (contexto) | Retângulo grande | Azul |
| Microserviço | Container / retângulo | Ver domínio abaixo |
| Banco Postgres | Cylinder | Cinza/azul escuro |
| Redis / bus | Hexágono ou rounded | Vermelho suave |
| MinIO | Rounded | Laranja suave |
| Sistema externo | Retângulo tracejado | Branco + borda tracejada |

**Cores por domínio**
- Plataforma (Gateway, Identity, Support, Files, Search, Audit): azul
- Saúde (Clinical): verde
- Marketplace (Commerce, Orders, Delivery): laranja
- Dinheiro (Billing): roxo
- Realtime (Comms): teal

**Em toda aba:** título no topo (`Vitalis — C4 Contexto | v1.0 | 2026-09-10`), legenda de cores no canto, autor.

Biblioteca útil no draw.io: *More shapes* → **Software** → **C4**.

---

## 1.1 C4 — Contexto

### Objetivo
Mostrar **o sistema como uma caixa só** e quem interage com ele. Sem microserviços.

### Caixa central
```
Rede Médica D'Excelência Vitalis
[Software System]
Plataforma de saúde digital: telemedicina,
planos, farmácia parceira e entrega.
```

### Atores (caixas ao redor)

| Nome | Descrição (subtítulo) |
|---|---|
| Paciente | Usuário final da rede |
| Médico / Profissional | Atendimento e teleconsulta |
| Farmácia parceira | Catálogo e separação de pedidos |
| Motoboy | Coleta e entrega |
| Administrador Vitalis | Operação, suporte, auditoria |

### Sistemas externos

| Nome | Descrição |
|---|---|
| PSP (Stripe / Mercado Pago) | Processa PIX, cartão e webhooks |
| Firebase | Push notifications |

### Setas (rótulos)

| De | Para | Rótulo |
|---|---|---|
| Paciente | Vitalis | Agenda, teleconsulta, compra e acompanha entrega |
| Médico | Vitalis | Atende, prescreve e gerencia agenda |
| Farmácia | Vitalis | Gerencia catálogo e pedidos |
| Motoboy | Vitalis | Recebe corridas e atualiza status |
| Admin | Vitalis | Opera plataforma, suporte e auditoria |
| Vitalis | PSP | Solicita cobranças; recebe confirmações |
| Vitalis | Firebase | Dispara notificações push |

### Layout
```
     Paciente          Médico
          \              /
           \            /
            [  VITALIS  ] ---- PSP
           /            \ ---- Firebase
          /              \
   Farmácia            Motoboy
           \
            Admin
```

---

## 1.2 C4 — Containers (v2 — referência oficial)

**Arquivo de imagem:** `vitalis-02-c4-containers-v2.png`  
**Objetivo:** diagrama sólido, sem DB compartilhado, Gateway → **todos** os serviços, stack **Go** explícita, integrações com `*`.

### O que corrigir em relação à sua versão atual

| Problema na sua arte | Correção |
|---|---|
| Gateway só liga em Identity / Delivery / Files | Gateway liga em **Identity, Clinical, Commerce, Orders, Delivery, Billing, Comms, Support, Files, Search, Audit** (11) |
| 2 cilindros compartilhados (vários serviços no mesmo DB) | **1 Postgres por serviço** (`pg_identity` … `pg_audit`) |
| Setas erradas (ex.: Orders→Support, Billing→Search) | Remover. Só sync essenciais (tracejadas) listadas abaixo |
| Sem stack nos boxes | Cada microsserviço: subtítulo **`Go`** |
| Externos genéricos / Stripe | Coluna **Integrações \*** (próprio **ou** provedor) — sem nome Stripe |
| Auth/mensageria/storage como “externo” | São **serviços internos** (Identity, Comms, Files) + Redis/MinIO\* na plataforma |

### Layout em 5 colunas (da esquerda → direita)

1. **CLIENTES** → 2. **GATEWAY** → 3. **MICROSSERVIÇOS (Go)** → 4. **DADOS** → 5. **INTEGRAÇÕES \***

### Como evitar “espaguete” no Gateway (técnica draw.io)

1. Dos 6 clientes, leve as setas até uma **barra vertical** (collector).  
2. Dessa barra, **uma seta só** entra no Gateway.  
3. Do Gateway, saia uma **barra horizontal (backbone HTTPS /v1)**.  
4. Dessa barra, caia **11 drops paralelos** (um por serviço) — alinhados, sem cruzar.

### Texto de cada caixa de microsserviço

| Serviço | Porta | Subtítulo (copie) |
|---|---|---|
| Identity | :8081 | Go · auth, usuários, papéis |
| Clinical | :8082 | Go · agenda, consulta, triagem, receita |
| Commerce | :8083 | Go · farmácias, produtos |
| Orders | :8084 | Go · pedidos, status |
| Delivery | :8085 | Go · motoboy, rastreio |
| Billing | :8086 | Go · PIX, ledger, split, planos |
| Comms | :8087 | Go · WS, notificações, chat |
| Support | :8088 | Go · tickets, artigos |
| Files | :8089 | Go · upload, PDF |
| Search | :8090 | Go · busca unificada |
| Audit | :8091 | Go · trilha LGPD |

**Gateway :8080** — `Go · roteamento · JWT validate · rate limit · CORS · request-id` (sem banco).

### Databases (cilindros — um para cada)

`pg_identity` · `pg_clinical` · `pg_commerce` · `pg_orders` · `pg_delivery` · `pg_billing` · `pg_comms` · `pg_support` · `pg_files` · `pg_search` · `pg_audit`

Legenda da coluna: **PostgreSQL 16 — 1 database / serviço**

### Infra de plataforma (ainda na coluna DADOS, não é “externo de produto”)

| Componente | Texto |
|---|---|
| Redis | cache + event streams / outbox bus |
| MinIO\* | object storage (Files) — \*próprio ou S3-compat |
| Observability | OTel + métricas/logs (opcional no desenho) |

### Integrações \* (coluna externa)

| Caixa | Texto |
|---|---|
| \* API Pagamentos | PIX/cartão — implementação própria **ou** PSP |
| \* Push Notifications | próprio **ou** provedor |
| \* E-mail transacional | próprio **ou** provedor |

Rodapé: `\* pode ser implementação própria da Vitalis ou provedor externo`

### Setas permitidas (sólidas = sync HTTP)

| De | Para | Rótulo |
|---|---|---|
| Clientes (collector) | Gateway | HTTPS |
| Gateway | **cada** um dos 11 serviços | /v1 |
| Cada serviço | **seu** `pg_*` | SQL |
| Files | MinIO\* | S3 API |
| Billing | \* API Pagamentos | HTTPS + webhooks |
| Comms | \* Push / \* E-mail | envio |

### Setas sync entre serviços (tracejadas, poucas)

| De | Para | Rótulo |
|---|---|---|
| Clinical | Billing | consulta assinatura |
| Orders | Billing | cobrar pedido |
| Orders | Commerce | estoque / catálogo |
| Delivery | Orders | vínculo entrega |

### Async (tracejado via Redis / bus)

- Serviços de domínio → Redis (`eventos outbox`)
- Bus → **Audit** (trilha LGPD) — use **uma barra de eventos** entrando no Audit, não 11 setas cruzadas
- Billing/Orders/Delivery/Clinical → Comms (notificar) via eventos

### NÃO desenhar

- DB compartilhado por vários serviços  
- Orders → Support  
- Billing → Search  
- Identity → Delivery como dependência principal  
- Stripe / Firebase como nomes obrigatórios  
- Gateway com banco próprio  

### Boundary

Retângulo tracejado: **Backend Vitalis — polyrepo · Go + PostgreSQL**

---

## 1.2 C4 — Containers (legado v1 — substituído)

> A imagem `vitalis-02-c4-containers.png` (v1) foi substituída conceitualmente pela **v2**. Use só a v2.

---

## 1.3 Rede / deploy local (`Vitalis-infra`)

### Objetivo
Mostrar **como sobe no Docker Compose**, não o C4.

### Caixa grande
```
Docker network: vitalis_net (bridge)
Repo: Vitalis-infra
```

### Dentro da rede — grupos

**Apps**
gateway, identity, clinical, commerce, orders, delivery, billing, comms, support, files, search, audit

**Data**
postgres (múltiplos databases **ou** N containers), redis, minio

**Ops / dev**
otel-collector, prometheus, grafana, mailhog

### Portas publicadas (legendas)

| Serviço | Host:Container |
|---|---|
| gateway | 8080:8080 |
| identity | 8081:8081 (opcional) |
| clinical | 8082 |
| commerce | 8083 |
| orders | 8084 |
| delivery | 8085 |
| billing | 8086 |
| comms | 8087 |
| support | 8088 |
| files | 8089 |
| search | 8090 |
| audit | 8091 |
| postgres | 5432 (só debug) |
| redis | 6379 |
| minio | 9000 / 9001 |
| mailhog | 8025 |
| grafana | 3000 |

### Nota de rodapé
> Em produção acadêmica/demo: clientes no host apontam `http://localhost:8080`.  
> Cada repo de serviço também pode subir compose unitário para dev isolado.

---

## 1.4a Sequência — Assinatura via PIX (v2)

**Imagem:** `vitalis-04a-seq-pix-v2.png`

### Lifelines (nessa ordem)
`Paciente Web` → `Gateway :8080` → `Identity :8081 (Go)` → `Billing :8086 (Go)` → `* API Pagamentos` → `Clinical :8082 (Go)` → `Comms :8087 (Go)`

### Mensagens

| # | De → Para | Mensagem |
|---|---|---|
| 1 | Paciente → Gateway | `POST /v1/auth/login` |
| 2 | Gateway → Identity | valida credenciais |
| 3 | Identity → Paciente | `JWT` RS256 (via Gateway) |
| 4 | Paciente → Gateway | `POST /v1/subscriptions` `{plano}` |
| 5 | Gateway → Billing | cria assinatura + `PaymentIntent` |
| 6 | Billing → Paciente | QR PIX + `expires_at` + `intentId` |
| 7 | Paciente → * API Pagamentos | paga PIX no banco (fora) |
| 8 | * API Pagamentos → Billing | webhook `payment.approved` (assinado) |
| 9 | Billing (interno) | idempotência + ledger + ativa plano |
| 10 | Billing → (evento) | `subscription.activated` |
| 11 | Clinical | libera gate de plano |
| 12 | → Comms | notificar plano ativo |
| 13 | Comms → Paciente | push/in-app “Plano ativo” |

Notas: Redis outbox · `pg_billing` · `*` = próprio ou PSP

---

## 1.4b Sequência — Pedido + split + entrega (v2)

**Imagem:** `vitalis-04b-seq-pedido-v2.png`

### Lifelines
`Paciente` → `Gateway` → `Commerce :8083 (Go)` → `Orders :8084 (Go)` → `Billing :8086 (Go)` → `Delivery :8085 (Go)` → `Motoboy App` → `Comms :8087 (Go)`

### Mensagens

| # | De → Para | Mensagem |
|---|---|---|
| 1 | Paciente → Commerce | `GET /v1/produtos` (via Gateway) |
| 2 | Commerce → Paciente | catálogo |
| 3 | Paciente → Orders | `POST /v1/pedidos` |
| 4 | Orders | persiste `pg_orders` status=`created` |
| 5 | Orders → (evento) | `order.placed` |
| 6 | Billing | `PaymentIntent` + cobrança via `* API Pagamentos` |
| 7 | * API / webhook | confirma pagamento |
| 8 | Billing → (evento) | `payment.succeeded` + `split.allocated` |
| 9 | Orders | status=`paid` |
| 10 | Delivery | cria entrega + assign motoboy (`pg_delivery`) |
| 11 | Delivery → Motoboy | notifica corrida (via Comms) |
| 12 | Motoboy → Delivery | aceita / status + GPS |
| 13 | Delivery → (evento) | `tracking.updated` |
| 14 | Comms → Paciente | push rastreio |
| 15 | Motoboy → Delivery | `delivered` |
| 16 | Delivery → (evento) | `delivery.completed` |
| 17 | Billing | settlement / repasses (ledger) |
| 18 | Orders | status=`delivered` |
| 19 | Comms → Paciente | “Pedido entregue” |

Sync opcional: Orders ↔ Commerce (estoque). Sem DB compartilhado.

---

## 1.5 Matriz serviço × cliente

Desenhe uma **tabela** no draw.io (ou Excel → print na aba).

Legenda:
`G` Gateway · `I` Identity · `C` Clinical · `Co` Commerce · `O` Orders · `D` Delivery · `B` Billing · `Cm` Comms · `S` Support · `F` Files · `Se` Search · `A` Audit

| Cliente | G | I | C | Co | O | D | B | Cm | S | F | Se | A |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Web Paciente | ● | ● | ● | ● | ● | ● | ● | ● | ● | ● | ● | |
| Web Profissional | ● | ● | ● | | | | ● | ● | ● | ● | ● | |
| Web Farmácias | ● | ● | | ● | ● | | ● | ● | ● | ● | | |
| Web Admin | ● | ● | ● | ● | ● | ● | ● | ● | ● | ● | ● | ● |
| Mobile Paciente | ● | ● | ● | ● | ● | ● | ● | ● | ● | ● | | |
| Mobile Motoboy | ● | ● | | | | ● | | ● | | | | |

Nota sob a tabela: *Todo tráfego HTTP passa pelo Gateway; a coluna G é sempre ●.*

---

## 1.6 Matriz de eventos

### Envelope (caixa de nota no canto)
```json
{
  "id": "evt_...",
  "type": "payment.succeeded",
  "occurred_at": "ISO-8601",
  "producer": "Vitalis-billing",
  "version": 1,
  "correlation_id": "...",
  "data": {}
}
```

### Tabela

| Evento | Producer | Consumers |
|---|---|---|
| `user.registered` | Identity | Billing, Clinical, Audit, Comms |
| `user.updated` | Identity | Orders, Delivery, Comms |
| `subscription.activated` | Billing | Clinical, Comms |
| `subscription.cancelled` | Billing | Clinical, Comms |
| `subscription.past_due` | Billing | Clinical, Comms |
| `appointment.confirmed` | Clinical | Comms |
| `appointment.completed` | Clinical | Billing, Comms |
| `prescription.issued` | Clinical | Comms, Files |
| `order.placed` | Orders | Billing, Commerce |
| `payment.succeeded` | Billing | Orders, Delivery, Comms |
| `payment.failed` | Billing | Orders, Comms |
| `split.allocated` | Billing | (ledger interno; opcional Comms admin) |
| `delivery.assigned` | Delivery | Orders, Comms |
| `tracking.updated` | Delivery | Comms |
| `delivery.completed` | Delivery | Orders, Billing, Comms |
| `ticket.created` | Support | Comms |
| `notification.sent` | Comms | — |

Opcional: segundo desenho **event storming leve** (post-its: comando / evento / política) só para PIX e Pedido.

---

## Ordem para você desenhar hoje

1. Aba `01-Contexto` (20–30 min)  
2. Aba `02-Containers` (45–60 min) — o mais denso  
3. Aba `03-Rede-Compose` (20 min)  
4. Abas `04a` e `04b` (30–40 min)  
5. Abas `05` e `06` (tabelas — 20 min)

Quando terminar cada aba, me manda print ou descreve dúvidas (ex.: “seta X deve ir para Y?”) que eu ajusto o texto/rótulos com você.

---

## Export para monografia

- PNG 2x ou SVG por aba  
- Nome: `vitalis-c4-contexto.png`, `vitalis-c4-containers.png`, etc.  
- Guardar o `.drawio` versionado em `docs/diagramas/`
