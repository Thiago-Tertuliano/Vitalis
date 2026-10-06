# Plano do Projeto — Vitalis Microserviços

Plano de execução da reescrita da Rede Médica D'Excelência Vitalis em **arquitetura de microserviços completa**, com **todos os serviços divididos**, stack **Go + PostgreSQL**, visando prática, portfólio e projeto de grande porte (TCC).

| Campo | Valor |
|---|---|
| Versão | 1.1 |
| Data | 2026-09-09 |
| Decisão | Microserviços **completos** + **polyrepo** (1 repo / serviço) |
| Stack | Go 1.22+, PostgreSQL 16, Redis, Docker Compose |
| Doc de arquitetura | `docs/ARQUITETURA_MICROSERVICOS_VITALIS.md` |
| Objetivo | Projeto grande + prática de sistemas distribuídos + demo E2E |

---

## 1. Decisão do projeto

### 1.1 O que foi decidido

- Reescrever o backend em **microserviços**, com **um repositório Git e um database Postgres por serviço** (**polyrepo**, sem monorepo).
- **Não** usar monólito modular como alvo final — o alvo é a plataforma distribuída.
- Contratos centralizados em `Vitalis-contracts`; orquestração local em `Vitalis-infra`.
- Manter frontends atuais (React / React Native) consumindo via **API Gateway**.
- Tratar **`Vitalis-billing`** como serviço vitrine (PIX, ledger, split, idempotência).
- Aceitar complexidade operacional em troca de aprendizado e escala de portfólio.

### 1.2 Critério de sucesso do projeto

O projeto é considerado **entregue** quando:

1. Todos os serviços da seção 2 sobem via `docker compose up`.
2. Fluxo paciente E2E funciona: **login → assinar plano (PIX) → agendar → teleconsulta (sinalização) → prescrição**.
3. Fluxo farmácia E2E funciona: **catálogo → pedido → pagamento (split) → entrega (motoboy) → rastreio**.
4. Cada serviço tem: README, OpenAPI (ou equivalente), migrations Postgres, healthcheck, testes mínimos.
5. Monografia/portfólio documenta C4, eventos e decisões (ADR).

---

## 2. Catálogo completo de serviços (todos divididos)

### 2.1 Serviços obrigatórios (escopo do projeto)

| # | Serviço | Repositório | Database Postgres | Porta sug. | Prioridade |
|---|---|---|---|---|---|
| 0 | API Gateway | `Vitalis-gateway` | — (stateless) | 8080 | P0 |
| 1 | Identity & Access | `Vitalis-identity` | `vitalis_identity` | 8081 | P0 |
| 2 | Clinical | `Vitalis-clinical` | `vitalis_clinical` | 8082 | P0 |
| 3 | Commerce (catálogo) | `Vitalis-commerce` | `vitalis_commerce` | 8083 | P0 |
| 4 | Orders (pedidos) | `Vitalis-orders` | `vitalis_orders` | 8084 | P0 |
| 5 | Delivery (entrega) | `Vitalis-delivery` | `vitalis_delivery` | 8085 | P0 |
| 6 | Billing (pagamentos) | `Vitalis-billing` | `vitalis_billing` | 8086 | P0 |
| 7 | Communication | `Vitalis-comms` | `vitalis_comms` | 8087 | P0 |
| 8 | Support | `Vitalis-support` | `vitalis_support` | 8088 | P1 |
| 9 | Files (upload/PDF) | `Vitalis-files` | `vitalis_files` (+ object storage) | 8089 | P1 |
| 10 | Search | `Vitalis-search` | `vitalis_search` (ou OpenSearch) | 8090 | P2 |
| 11 | Audit / LGPD log | `Vitalis-audit` | `vitalis_audit` | 8091 | P2 |

### 2.2 Infra compartilhada (não é domínio de negócio)

| Componente | Tecnologia | Função |
|---|---|---|
| Postgres | PostgreSQL 16 | Um database por serviço |
| Redis | Redis 7 | Cache, idempotency TTL, pub/sub, streams |
| Message bus | Redis Streams (fase 1) → RabbitMQ/NATS (fase 2) | Eventos entre serviços |
| Object storage | MinIO (local) | Arquivos / imagens / PDFs |
| Observabilidade | OTel Collector + Prometheus + Grafana | Traces, métricas, dashboards |
| Mail (dev) | Mailhog | E-mails de reset/notificação |

### 2.3 Clientes (fora do backend, mas no plano)

| Cliente | Repo atual | Consome |
|---|---|---|
| Web Paciente | `Vitalis-web-paciente` | Gateway |
| Web Profissional | `Vitalis-web-profissional` | Gateway |
| Web Farmácias | `Vitalis-web-farmacias` | Gateway |
| Web Admin | `Vitalis-web-administrador` | Gateway |
| Mobile Paciente | `Vitalis-mobile-paciente` | Gateway |
| Mobile Motoboy | `Vitalis-mobile-motoboy` | Gateway + Delivery/Comms |

---

## 3. Responsabilidade de cada serviço (resumo executivo)

| Serviço | Faz | Não faz |
|---|---|---|
| **gateway** | Roteamento, JWT validate, rate limit, CORS, request-id | Regra de negócio |
| **identity** | Cadastro, login, refresh, papéis, endereços, docs cadastrais | Consulta clínica, cobrança |
| **clinical** | Agenda, consulta, triagem, videochamada (sinalização), receita | Pagamento, catálogo |
| **commerce** | Farmácias, produtos, categorias, estoque de vitrine | Pedido, entrega, cobrança |
| **orders** | Carrinho/pedido, itens, status do pedido, chat do pedido* | Cobrança, roteiro do motoboy |
| **delivery** | Motoboy, veículo, alocação, rastreamento, status entrega | Catálogo, ledger |
| **billing** | PaymentIntent, PIX/cartão, assinatura, wallet, ledger, split | Estoque, agenda |
| **comms** | Notificações, WS realtime, chat clínico | Persistência clínica |
| **support** | Tickets, artigos de ajuda | Auth |
| **files** | Upload, URLs assinadas, PDF | Domínio de negócio |
| **search** | Índice cross-serviço (médicos, produtos, artigos) | Source of truth |
| **audit** | Trilha imutável de acesso a dado sensível | UI |

\* Chat do pedido pode ser canal no `comms` com `room_type=order`; ownership da regra de pedido fica em `orders`.

---

## 4. Organização dos repositórios (polyrepo)

**Decisão:** **não haverá monorepo.** Cada microserviço (e a infra compartilhada) vive em **repositório Git separado**, no mesmo padrão da org atual (`Vitalis-web-*`, `Vitalis-Backend`, etc.).

### 4.1 Repos de backend (um por serviço)

| Repositório | Conteúdo |
|---|---|
| `Vitalis-gateway` | API Gateway |
| `Vitalis-identity` | Identidade e acesso |
| `Vitalis-clinical` | Domínio clínico / telemedicina |
| `Vitalis-commerce` | Catálogo farmácia |
| `Vitalis-orders` | Pedidos |
| `Vitalis-delivery` | Entrega / motoboy |
| `Vitalis-billing` | Pagamentos (vitrine) |
| `Vitalis-comms` | Notificações + WebSocket |
| `Vitalis-support` | Tickets e ajuda |
| `Vitalis-files` | Upload / PDF / MinIO |
| `Vitalis-search` | Busca unificada |
| `Vitalis-audit` | Auditoria LGPD |

### 4.2 Repos transversais (sem código de domínio)

| Repositório | Conteúdo |
|---|---|
| `Vitalis-infra` | Docker Compose da plataforma, Prometheus/Grafana, scripts `bootstrap`, seeds multi-serviço |
| `Vitalis-contracts` | OpenAPI + AsyncAPI (eventos) versionados — fonte da verdade dos contratos |
| `Vitalis-go-kit` *(opcional)* | Módulo Go privado/compartilhado (`go get`): logging, middleware, health, otel |

> Contratos **não** ficam “só dentro” de um serviço: mudou API/evento → PR em `Vitalis-contracts`, depois cada serviço atualiza a versão consumida.

### 4.3 Layout interno de cada repo de serviço

Cada repositório de domínio segue o mesmo molde (independente):

```
Vitalis-billing/          # exemplo
├── cmd/api/
├── cmd/worker/
├── internal/domain/
├── internal/app/
├── internal/port/
├── internal/adapter/http/
├── internal/adapter/postgres/
├── internal/adapter/redis/
├── migrations/
├── api/openapi.yaml       # espelho local; canônico em Vitalis-contracts
├── Dockerfile
├── docker-compose.yml     # sobe SÓ este serviço + seu Postgres (+ deps mínimas)
├── .env.example
├── Makefile
├── go.mod
└── README.md
```

### 4.4 Como desenvolver sem monorepo

1. **Dev de um serviço:** `docker compose up` **dentro do repo** (serviço + seu DB).
2. **Dev da plataforma:** clone dos repos + `Vitalis-infra` com Compose que sobe **todos** os containers e aponta para as imagens/builds de cada serviço.
3. **CI:** cada repo tem seu próprio GitHub Actions (`test`, `lint`, `docker build`).
4. ** liberar versão:** tag semver por repo (`v0.1.0`); infra pina tags/digests no Compose de demo.
5. **Código compartilhado:** preferir `Vitalis-go-kit` como módulo Go; evitar copiar-colar middleware.

### 4.5 Frontends (inalterados)

Continuam nos repos atuais e apontam `VITE_API_URL` / equivalente para o Gateway.

---

## 5. Stack padrão de cada microserviço Go

Todo serviço de domínio segue o mesmo molde:

```
cmd/api/main.go
cmd/worker/main.go          # se publicar/consumir eventos
internal/domain/
internal/app/
internal/port/
internal/adapter/http/
internal/adapter/postgres/
internal/adapter/redis/
migrations/
api/openapi.yaml
Dockerfile
README.md
```

| Peça | Padrão |
|---|---|
| HTTP | chi |
| DB | pgx + Postgres |
| Migrate | goose ou golang-migrate |
| Config | envconfig / cleanenv |
| Logs | slog |
| Testes | testify + testcontainers (Postgres) |
| Eventos | outbox + Redis Streams |

---

## 6. Roadmap por fases (plano de execução)

Duração estimada para projeto grande / TCC com dedicação consistente: **~16–20 semanas**. Ajuste conforme o time.

### Visão das fases

```
F0 Fundação → F1 Identity+Gateway → F2 Billing ★
→ F3 Clinical → F4 Commerce+Orders → F5 Delivery
→ F6 Comms → F7 Support+Files → F8 Search+Audit → F9 Hardening+Demo
```

---

### Fase 0 — Fundação (Semanas 1–2)

**Meta:** padrões polyrepo + infra de plataforma.

| Entrega | DoD |
|---|---|
| Org/repos criados | `Vitalis-infra`, `Vitalis-contracts`, `Vitalis-gateway`, `Vitalis-identity` (mínimo) |
| `Vitalis-infra` | Compose da plataforma: Postgres (multi-DB), Redis, MinIO, Mailhog; rede Docker compartilhada |
| `Vitalis-contracts` | Pasta inicial OpenAPI/AsyncAPI + README de versionamento |
| Template de serviço | README-template + Dockerfile-template + CI template replicável nos próximos repos |
| `Vitalis-go-kit` *(opcional)* | Logger, middleware request-id, health, error JSON |
| ADR-001 | Decisão: microserviços + Go + Postgres + **polyrepo** |

**Milestone M0:** `Vitalis-infra` sobe dependências saudáveis; gateway/identity stubs conectam na rede.

---

### Fase 1 — Gateway + Identity (Semanas 3–4)

**Meta:** autenticação unificada.

| Serviço | Entregas |
|---|---|
| `vitalis-identity` | Register, login, refresh, logout, `/me`, papéis, endereço |
| `vitalis-gateway` | Proxy para identity; validação JWT RS256; CORS; rate limit |
| Eventos | `user.registered`, `user.updated`, `user.role_changed` |
| Seeds | Paciente, médico, farmácia, admin, motoboy de teste |

**Milestone M1:** login no web paciente via Gateway → JWT válido.

---

### Fase 2 — Billing vitrine (Semanas 5–7)

**Meta:** coração financeiro do projeto (prioridade de prática).

| Entrega | Detalhe |
|---|---|
| PaymentIntent + state machine | created → pending → captured/failed/expired/refunded |
| PIX (dev) | Adapter real (MP) **ou** sandbox + webhook simulator |
| Cartão | Adapter Stripe/MP (sandbox) |
| Ledger | Partidas dobradas; `NUMERIC`; sem float |
| Assinaturas | Criar, trocar plano, cancelar, `past_due` |
| Wallet | Crédito/débito interno |
| Split | Regras plataforma / profissional / farmácia / motoboy |
| Idempotency-Key | Obrigatório em POSTs |
| Outbox + worker | Publica `payment.*`, `subscription.*` |
| Testes | Unit + integration (testcontainers) + webhook duplicado |

**Milestone M2:** assinar plano via PIX (sandbox) e emitir `subscription.activated`.

**Demo parcial:** Postman/Insomnia + QR PIX + webhook.

---

### Fase 3 — Clinical (Semanas 8–10)

**Meta:** jornada de saúde.

| Entrega | Detalhe |
|---|---|
| Especialidades / médicos | CRUD leitura + perfil |
| Disponibilidade / agenda | Slots |
| Consultas | Criar, confirmar, cancelar, completar |
| Triagem | Questionário + resultado |
| Videochamada | Sala/sinalização (WebRTC client nos frontends) |
| Prescrição | Emitir + listar |
| Gate de plano | Consome Billing (sync curto ou cache de evento) |

**Eventos:** `appointment.*`, `triage.completed`, `prescription.issued`, `video.session.*`

**Milestone M3:** paciente com plano ativo agenda e abre sala de teleconsulta.

---

### Fase 4 — Commerce + Orders (Semanas 11–12)

**Meta:** farmácia até o pedido pago.

| Serviço | Entregas |
|---|---|
| `vitalis-commerce` | Farmácias, categorias, produtos, estoque vitrine |
| `vitalis-orders` | Criar pedido, itens, status, cancelamento |
| Integração Billing | `order.placed` → PaymentIntent → `payment.succeeded` → `order.paid` |

**Milestone M4:** pedido farmácia pago (PIX/cartão sandbox) com split alocado.

---

### Fase 5 — Delivery (Semana 13)

**Meta:** logística separada de pedidos.

| Entrega | Detalhe |
|---|---|
| Motoboy / veículo | Cadastro e disponibilidade |
| Alocação | `order.paid` → cria entrega → assign motoboy |
| Rastreamento | Pontos / status |
| App motoboy | Consome Delivery + Identity |

**Eventos:** `delivery.assigned|started|completed`, `tracking.updated`

**Milestone M5:** pedido pago gera entrega e status chega ao paciente.

---

### Fase 6 — Comms (Semana 14)

**Meta:** tempo real e notificações.

| Entrega | Detalhe |
|---|---|
| Notificações in-app | Lista + marcar lida |
| WebSocket | Presença + push de eventos |
| Chat clínico | Paciente ↔ médico |
| Fan-out | Consome quase todos os eventos de domínio |
| E-mail / push | Mailhog + Firebase (se der tempo) |

**Milestone M6:** notificação em tempo real quando consulta é confirmada / pedido muda status.

---

### Fase 7 — Support + Files (Semana 15)

| Serviço | Entregas |
|---|---|
| `vitalis-support` | Tickets, comentários, artigos ajuda |
| `vitalis-files` | Upload perfil/produto, URL assinada MinIO, PDF simples |

**Milestone M7:** upload de imagem + abrir ticket no admin/paciente.

---

### Fase 8 — Search + Audit (Semana 16)

| Serviço | Entregas |
|---|---|
| `vitalis-search` | Indexar médicos/produtos/artigos via eventos |
| `vitalis-audit` | Log de acesso a recursos clínicos/financeiros sensíveis |

**Milestone M8:** busca unificada no Gateway + trilha de auditoria consultável pelo admin.

---

### Fase 9 — Hardening, frontends e demo final (Semanas 17–20)

| Frente | Entregas |
|---|---|
| Frontends | Apontar para Gateway; ajustar auth; fluxos E2E críticos |
| Observabilidade | Grafana com RED metrics por serviço |
| Segurança | Secrets, rate limit, revisão LGPD, sem `.env` commitado |
| Docs | C4, sequência PIX, sequência split, ADRs |
| Demo day | Script de 10–12 min + seed resetável |
| Carga leve | k6 ou vegetta em Identity/Billing/Clinical |

**Milestone M9 (FINAL):** demo E2E completa + compose estável + documentação.

---

## 7. Backlog épico (ordem de implementação)

1. Fundação polyrepo (`Vitalis-infra` + `Vitalis-contracts` + template)  
2. Identity (`Vitalis-identity`)  
3. Gateway (`Vitalis-gateway`)  
4. **Billing** (`Vitalis-billing` — PaymentIntent, ledger, PIX, assinatura, split)  
5. Clinical (`Vitalis-clinical`)  
6. Clinical (triagem + video + receita)  
7. Commerce (`Vitalis-commerce`)  
8. Orders (`Vitalis-orders`) + integração pagamento  
9. Delivery (`Vitalis-delivery`)  
10. Comms (`Vitalis-comms`)  
11. Support (`Vitalis-support`)  
12. Files (`Vitalis-files`)  
13. Search (`Vitalis-search`)  
14. Audit (`Vitalis-audit`)  
15. Integração frontends + demo via `Vitalis-infra`  

---

## 8. Contratos e comunicação

### 8.1 Síncrono (REST via Gateway)

```
Cliente → Gateway → Serviço
```

Chamadas serviço→serviço **curtas** e raras:

- Clinical → Billing: `GET /internal/subscriptions/active?user_id=`
- Orders → Commerce: reservar/consultar estoque
- Orders → Billing: criar cobrança (ou só via evento — preferir evento)
- Delivery → Orders: atualizar vínculo entrega

### 8.2 Assíncrono (eventos)

Padrão: **Transactional Outbox** em todo serviço que publica.

Eventos mínimos do projeto:

| Evento | Producer | Consumers |
|---|---|---|
| `user.registered` | identity | billing, clinical, audit |
| `subscription.activated` | billing | clinical, comms |
| `appointment.confirmed` | clinical | comms |
| `prescription.issued` | clinical | comms, files |
| `order.placed` | orders | billing, commerce |
| `payment.succeeded` | billing | orders, delivery, comms |
| `delivery.completed` | delivery | orders, billing (settle), comms |
| `ticket.created` | support | comms |

### 8.3 Versionamento

- APIs: `/v1/...`
- Eventos: campo `version` no envelope
- Breaking change = nova versão + período de convivência

---

## 9. Ambientes

| Ambiente | Uso | Como sobe |
|---|---|---|
| `local` | Dev diário | Docker Compose |
| `demo` | Banca / portfólio | Compose ou VM única |
| `ci` | Testes | Compose service + testcontainers |

**Não é obrigatório Kubernetes no TCC.** Compose bem feito + documentação já prova o porte. K8s entra como evolução se sobrar tempo (Fase 9+).

---

## 10. Qualidade e Definition of Done (por serviço)

Um serviço só entra como “pronto” se:

- [ ] `GET /health` e `GET /ready` ok  
- [ ] Migrations versionadas  
- [ ] OpenAPI publicado em `docs/openapi/<servico>.yaml`  
- [ ] Testes unitários do domínio feliz + 1 integração Postgres  
- [ ] Dockerfile + serviço no `docker-compose.yml`  
- [ ] README com: responsabilidade, portas, env vars, eventos  
- [ ] Logs com `request_id`  
- [ ] Sem segredo commitado  

Billing, além disso:

- [ ] Idempotência coberta por teste  
- [ ] Webhook duplicado não duplica ledger  
- [ ] Split gera lançamentos balanceados  

---

## 11. Papéis sugeridos (se houver time)

| Papel | Foco |
|---|---|
| Arquiteto / Tech lead | Contratos (`Vitalis-contracts`), ADRs, Gateway, `Vitalis-go-kit`, `Vitalis-infra` |
| Eng. Billing | Pagamentos (vitrine) |
| Eng. Clinical | Telemedicina |
| Eng. Commerce/Orders/Delivery | Marketplace + logística |
| Eng. Comms/Support | Realtime + tickets |
| Frontend | Integração dos apps existentes |
| Docs/QA | Seeds, roteiro demo, testes E2E |

Solo: siga a ordem das fases; não paralelize mais de 2 serviços.

---

## 12. Riscos e mitigações

| Risco | Impacto | Mitigação |
|---|---|---|
| Escopo estourar | Projeto incompleto | Congelar Search/Audit se atrasar; M2+M3+M4 são inegociáveis |
| Complexidade de rede | Demo instável | `Vitalis-infra` + healthchecks + script `make demo` |
| Drift entre repos | Contratos divergentes | OpenAPI/eventos só mudam via `Vitalis-contracts` + CI de compatibilidade |
| PSP real difícil | Billing trava | Sandbox + webhook simulator desde o dia 1 |
| Frontends legados | Integração lenta | BFF mínimo no Gateway; adaptar uma tela por fluxo |
| Cansaço do time | Abandono | Milestones quinzenais com demo interna |
| Monólito distribuído | Pior dos mundos | Nunca compartilhar tabelas entre serviços |

---

## 13. Entregáveis acadêmicos / portfólio

1. Este plano + doc de arquitetura  
2. ADRs (microserviços, Go/Postgres, billing ledger, outbox)  
3. Diagramas C4 (contexto, containers, billing component)  
4. Sequências: assinatura PIX; pedido com split e entrega  
5. Vídeo demo 10–12 min  
6. Relatório de testes (Identity, Billing, Clinical, Orders)  
7. Capítulo LGPD (dados clínicos vs financeiros)

---

## 14. Cronograma resumido

| Semanas | Fase | Milestone |
|---|---|---|
| 1–2 | F0 Fundação | M0 Compose |
| 3–4 | F1 Identity + Gateway | M1 Login |
| 5–7 | F2 Billing ★ | M2 Assinatura PIX |
| 8–10 | F3 Clinical | M3 Teleconsulta |
| 11–12 | F4 Commerce + Orders | M4 Pedido pago |
| 13 | F5 Delivery | M5 Entrega |
| 14 | F6 Comms | M6 Realtime |
| 15 | F7 Support + Files | M7 Tickets/upload |
| 16 | F8 Search + Audit | M8 Busca/auditoria |
| 17–20 | F9 Hardening + Demo | M9 Final |

---

## 15. Próxima ação imediata (agora)

1. Criar na org os repos: `Vitalis-infra`, `Vitalis-contracts`, `Vitalis-gateway`, `Vitalis-identity`.  
2. Subir Compose em `Vitalis-infra` (Postgres multi-DB + Redis + MinIO).  
3. Scaffold Go em `Vitalis-identity` e `Vitalis-gateway` (repos separados).  
4. Publicar ADR-001 (polyrepo + Go + Postgres) em `docs/` deste workspace ou em `Vitalis-contracts`.  
5. Rascunhar OpenAPI do Billing em `Vitalis-contracts` (contrato da vitrine).

---

## 16. Relação com o documento de arquitetura

| Documento | Papel |
|---|---|
| `ARQUITETURA_MICROSERVICOS_VITALIS.md` | O **quê** e **por quê** (desenho técnico, billing detalhado) |
| `PLANO_PROJETO_MICROSERVICOS_VITALIS.md` (este) | O **quando** e **como executar** (fases, DoD, milestones) |

**Diferença explícita deste plano:** orders e delivery são **serviços e repos separados**; files, search e audit entram no escopo; organização é **polyrepo** (sem monorepo).

---

*Plano v1.1 — Vitalis em microserviços completos (Go + PostgreSQL), **polyrepo** (1 repo por serviço). Foco: prática, porte de projeto e demo E2E defensável.*
