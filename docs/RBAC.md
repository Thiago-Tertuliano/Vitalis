# RBAC — Vitalis

Matriz de controle de acesso baseada em papéis (RBAC) alinhada ao **ADR-004** (JWT RS256 + ownership no serviço) e aos contratos OpenAPI em `docs/contracts/openapi/`.

Papéis canônicos no claim JWT `roles`:

| Papel | Descrição |
|---|---|
| `paciente` | Usuário final (web/mobile paciente) |
| `medico` | Profissional de saúde (teleconsulta e prescrição) |
| `farmacia` | Operador da farmácia parceira |
| `motoboy` | Entregador |
| `admin` | Operações da plataforma (break-glass) |

---

## Gateway e identidade propagada

Todo tráfego **público** dos clientes (web/mobile) entra **somente pelo Gateway** (`:8080`). Os serviços de domínio (`:8081`–`:8091`) não devem ser expostos à internet.

Fluxo de autorização (ADR-004):

1. Cliente envia `Authorization: Bearer <access_token>`.
2. Gateway valida o JWT (JWKS do Identity, issuer, expiração).
3. Gateway injeta headers internos confiáveis na rede privada:
   - `X-User-Id` — identificador do usuário autenticado
   - `X-Roles` — papéis do token (lista)
   - `X-Request-Id` — correlação de request
4. O serviço de domínio aplica **RBAC grosseiro** (papel permitido no endpoint) **e** checagem fina de **ownership** (recurso pertence ao ator).
5. Headers `X-User-Id` / `X-Roles` vindos da internet pública **devem ser ignorados**; só o Gateway (rede interna) é fonte de confiança.

Endpoints marcados como **público**, **system** ou **internal** não usam o modelo “usuário + roles” padrão:

| Tipo | Quem chama | Auth |
|---|---|---|
| Público | Cliente sem login | Sem Bearer (ou só rate-limit) |
| Autenticado | Usuário via Gateway | Bearer + headers internos |
| System | Provedor externo (ex.: webhook de pagamento) | Assinatura/segredo do provedor |
| Internal | Serviço → serviço na rede privada | Rede + (opcional) token de serviço; **não** via Gateway público |

---

## Legenda das matrizes

| Símbolo | Significado |
|---|---|
| **●** | Acesso normal — o papel pode chamar o endpoint/serviço no escopo previsto |
| **◐** | Acesso parcial — apenas recursos próprios / do próprio tenant (ownership) |
| **○** | Sem acesso — ou exclusivo de `admin` / system / internal (conforme a célula) |

Nas tabelas de endpoint, colunas de papel usam os mesmos símbolos. Células com nota “system” ou “internal” indicam que o chamador não é um dos cinco papéis de usuário.

---

## 1. Matriz papel × serviço

Visão grosseira: o que cada papel pode fazer em cada bounded context. Detalhe por rota na seção 2.

| Serviço | paciente | medico | farmacia | motoboy | admin | Resumo por célula |
|---|:---:|:---:|:---:|:---:|:---:|---|
| **gateway** | ● | ● | ● | ● | ● | Todos os clientes autenticados passam pelo Gateway. Rotas públicas (login, JWKS, webhook) também. Admin usa o mesmo ingresso. |
| **identity** | ● | ● | ● | ● | ● | Login/registro/refresh públicos ou autenticados; `me` e endereços para qualquer autenticado; alteração de papéis só admin. |
| **clinical** | ◐ | ◐ | ○ | ○ | ● | Paciente: agendar/ver próprias consultas, triagem, vídeo, ler próprias prescrições. Médico: agenda/atendimento próprios + emitir prescrição. Farmácia/motoboy: sem domínio clínico. Admin: break-glass. |
| **commerce** | ● | ○ | ◐ | ○ | ● | Paciente: listar farmácias/catálogo. Farmácia: CRUD produtos e loja própria. Estoque reservar/liberar: uso interno (Orders). Médico/motoboy: sem catálogo operacional. |
| **orders** | ◐ | ○ | ◐ | ○ | ● | Paciente: criar/listar/cancelar próprios pedidos. Farmácia: ver/atualizar status dos pedidos da loja. Motoboy/médico: sem CRUD de pedido (entrega é Delivery). |
| **delivery** | ◐ | ○ | ○ | ◐ | ● | Paciente: rastrear própria entrega. Motoboy: perfil, ofertas, aceitar, status, GPS das próprias corridas. Farmácia/médico: sem operação de rota. |
| **billing** | ◐ | ◐ | ◐ | ○ | ● | Paciente: payment-intents, PIX, assinatura, faturas, carteira. Médico: comissões. Farmácia: settlements da loja. Motoboy: sem billing direto (MVP). Webhook system; gate de assinatura internal. |
| **comms** | ◐ | ◐ | ◐ | ◐ | ● | Notificações e chat do próprio usuário / salas em que participa; realtime autenticado. Admin: moderação/break-glass. |
| **support** | ◐ | ◐ | ◐ | ◐ | ● | Abrir e acompanhar próprios tickets; ler artigos de ajuda (público autenticado). Admin: fila global. |
| **files** | ◐ | ◐ | ◐ | ◐ | ● | Upload/download de objetos autorizados ao ator (ex.: anexo de ticket, PDF de receita). Jobs PDF conforme ownership do documento. |
| **search** | ● | ● | ● | ● | ● | Busca unificada filtrada pelo papel (índices e resultados respeitam RBAC + ownership). |
| **audit** | ○ | ○ | ○ | ○ | ● | Consulta de trilha só admin. Ingestão só internal (serviços). |

### Notas por serviço (papel × capacidade)

| Serviço | paciente | medico | farmacia | motoboy | admin |
|---|---|---|---|---|---|
| **gateway** | Proxy autenticado + rotas públicas | Idem | Idem | Idem | Idem + rotas admin no backend |
| **identity** | Auth, perfil, endereços | Auth, perfil | Auth, perfil (tenant farmácia) | Auth, perfil | Auth + `PUT .../roles` |
| **clinical** | Consultas/triagens/vídeo/prescrições próprias | Disponibilidade, consultas e prescrições do profissional; dados do paciente no atendimento | — | — | Qualquer recurso (auditoria) |
| **commerce** | Catálogo e detalhe de farmácia/produto | — | Loja e estoque próprios; criar produtos | — | Cadastro/moderação de farmácias |
| **orders** | Pedidos próprios | — | Pedidos da farmácia | — | Todos os pedidos |
| **delivery** | Rastreio do próprio pedido | — | — | Corridas próprias | Todas as entregas |
| **billing** | Pagamentos e plano | Comissões | Liquidações | — | Refunds, settlements, visão global |
| **comms** | Inbox e chats do paciente | Chats clínicos | Chat de pedido | Chat/status de corrida | Moderação |
| **support** | Tickets próprios | Tickets próprios | Tickets próprios | Tickets próprios | Todos + artigos |
| **files** | Anexos próprios | Docs clínicos autorizados | Docs da loja | Comprovantes de entrega | Qualquer objeto |
| **search** | Busca paciente | Busca clínica | Busca catálogo/loja | Busca operacional limitada | Busca irrestrita |
| **audit** | — | — | — | — | Consulta; ingest só serviços |

---

## 2. Matriz papel × endpoint (detalhada)

Contratos: `identity`, `billing`, `clinical`, `commerce`, `orders`, `delivery`, `comms`, `support`, `files`, `search`, `audit`. Health (`/health`, `/ready`) é público em todos os serviços e omite-se aqui.

Colunas: **P** paciente · **M** medico · **F** farmacia · **B** motoboy · **A** admin.

### 2.1 Identity

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| POST | `/v1/auth/login` | ● | ● | ● | ● | ● | Público (qualquer papel após cadastro) |
| POST | `/v1/auth/refresh` | ● | ● | ● | ● | ● | Público com refresh token válido |
| POST | `/v1/auth/logout` | ● | ● | ● | ● | ● | Autenticado (revoga refresh) |
| POST | `/v1/auth/registro` | ● | ● | ● | ● | ● | Público; papel inicial conforme fluxo de cadastro |
| POST | `/v1/auth/esqueci-senha` | ● | ● | ● | ● | ● | Público |
| POST | `/v1/auth/redefinir-senha` | ● | ● | ● | ● | ● | Público (token de reset) |
| GET | `/v1/auth/jwks.json` | ● | ● | ● | ● | ● | Público (Gateway/clientes validam JWT) |
| GET | `/v1/usuarios/me` | ● | ● | ● | ● | ● | Perfil do autenticado |
| PUT | `/v1/usuarios/me` | ● | ● | ● | ● | ● | Atualiza próprio perfil |
| GET | `/v1/usuarios/me/enderecos` | ● | ● | ● | ● | ● | Endereços do próprio usuário |
| POST | `/v1/usuarios/me/enderecos` | ● | ● | ● | ● | ● | Cria endereço próprio |
| PUT | `/v1/usuarios/me/enderecos/{id}` | ◐ | ◐ | ◐ | ◐ | ● | Só endereço próprio |
| DELETE | `/v1/usuarios/me/enderecos/{id}` | ◐ | ◐ | ◐ | ◐ | ● | Só endereço próprio |
| PUT | `/v1/admin/usuarios/{id}/roles` | ○ | ○ | ○ | ○ | ● | Exclusivo admin |

### 2.2 Billing

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| POST | `/v1/payment-intents` | ● | ○ | ○ | ○ | ● | Paciente cria intent (pedido/assinatura) |
| GET | `/v1/payment-intents/{id}` | ◐ | ○ | ○ | ○ | ● | Só intent próprio |
| POST | `/v1/payment-intents/{id}/pix` | ◐ | ○ | ○ | ○ | ● | Gera PIX do próprio intent |
| POST | `/v1/payment-intents/{id}/cancel` | ◐ | ○ | ○ | ○ | ● | Cancela próprio intent (ou admin) |
| POST | `/v1/payment-intents/{id}/refund` | ○ | ○ | ○ | ○ | ● | Estorno — admin (ou política operacional) |
| POST | `/v1/subscriptions` | ● | ○ | ○ | ○ | ● | Assinar/trocar plano (paciente) |
| GET | `/v1/subscriptions/me` | ● | ○ | ○ | ○ | ● | Assinatura ativa do paciente |
| POST | `/v1/subscriptions/{id}/cancel` | ◐ | ○ | ○ | ○ | ● | Cancela própria assinatura |
| GET | `/v1/invoices` | ◐ | ○ | ○ | ○ | ● | Faturas do próprio usuário |
| GET | `/v1/wallets/me` | ● | ○ | ○ | ○ | ● | Saldo da carteira do paciente |
| POST | `/v1/wallets/topup` | ● | ○ | ○ | ○ | ● | Creditar carteira própria |
| GET | `/v1/settlements` | ○ | ○ | ◐ | ○ | ● | Farmácia: liquidações da loja; admin: todas |
| GET | `/v1/commissions` | ○ | ◐ | ○ | ○ | ● | Médico: próprias comissões |
| GET | `/v1/internal/subscriptions/active` | ○ | ○ | ○ | ○ | ○ | **Internal** — Clinical consulta gate de plano |
| POST | `/v1/webhooks/pagamentos` | ○ | ○ | ○ | ○ | ○ | **System** — provedor de pagamento (via Gateway) |

### 2.3 Clinical

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/especialidades` | ● | ● | ○ | ○ | ● | Catálogo público autenticado (paciente/médico) |
| GET | `/v1/medicos` | ● | ● | ○ | ○ | ● | Busca de profissionais |
| GET | `/v1/disponibilidade` | ● | ● | ○ | ○ | ● | Paciente consulta slots; médico vê/gerencia agenda própria no serviço |
| GET | `/v1/consultas` | ◐ | ◐ | ○ | ○ | ● | Paciente: próprias; médico: próprias |
| POST | `/v1/consultas` | ● | ○ | ○ | ○ | ● | Agendar (exige assinatura ativa via Billing internal) |
| GET | `/v1/consultas/{id}` | ◐ | ◐ | ○ | ○ | ● | Participantes da consulta (paciente ou médico do atendimento) |
| PATCH | `/v1/consultas/{id}` | ◐ | ◐ | ○ | ○ | ● | Status: paciente (cancelar) / médico (fluxo clínico) |
| POST | `/v1/triagens` | ● | ○ | ○ | ○ | ● | Paciente envia triagem |
| GET | `/v1/triagens/{id}` | ◐ | ◐ | ○ | ○ | ● | Paciente dono; médico no contexto do atendimento |
| POST | `/v1/videochamadas/sala` | ◐ | ◐ | ○ | ○ | ● | Entrar na sala da própria consulta |
| GET | `/v1/prescricoes` | ◐ | ◐ | ○ | ○ | ● | Paciente: próprias; médico: emitidas por si |
| POST | `/v1/prescricoes` | ○ | ● | ○ | ○ | ● | Emitir — só médico |
| GET | `/v1/prescricoes/{id}` | ◐ | ◐ | ○ | ○ | ● | Paciente destinatário ou médico emissor |

### 2.4 Commerce

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/farmacias` | ● | ○ | ● | ○ | ● | Listagem para paciente; farmácia vê contexto operacional |
| POST | `/v1/farmacias` | ○ | ○ | ○ | ○ | ● | Cadastro de farmácia — admin (onboarding) |
| GET | `/v1/farmacias/{id}` | ● | ○ | ◐ | ○ | ● | Detalhe público; farmácia edita só a própria no fluxo de gestão |
| GET | `/v1/categorias` | ● | ○ | ● | ○ | ● | Catálogo |
| GET | `/v1/produtos` | ● | ○ | ◐ | ○ | ● | Paciente: vitrine; farmácia: produtos da loja |
| POST | `/v1/produtos` | ○ | ○ | ● | ○ | ● | Criar produto — farmácia (loja própria) |
| GET | `/v1/produtos/{id}` | ● | ○ | ◐ | ○ | ● | Detalhe; farmácia valida tenant |
| PUT | `/v1/produtos/{id}` | ○ | ○ | ◐ | ○ | ● | Atualizar — só produto da própria farmácia |
| POST | `/v1/estoque/reservar` | ○ | ○ | ○ | ○ | ○ | **Internal** — Orders reserva estoque |
| POST | `/v1/estoque/liberar` | ○ | ○ | ○ | ○ | ○ | **Internal** — Orders libera reserva |

### 2.5 Orders

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/pedidos` | ◐ | ○ | ◐ | ○ | ● | Paciente: próprios; farmácia: da loja |
| POST | `/v1/pedidos` | ● | ○ | ○ | ○ | ● | Criar pedido — paciente |
| GET | `/v1/pedidos/{id}` | ◐ | ○ | ◐ | ○ | ● | Ownership paciente ou farmácia do pedido |
| POST | `/v1/pedidos/{id}/cancelar` | ◐ | ○ | ◐ | ○ | ● | Paciente ou farmácia (regras de status); admin break-glass |
| PATCH | `/v1/pedidos/{id}/status` | ○ | ○ | ◐ | ○ | ● | Farmácia avança status operacional; admin override |

### 2.6 Delivery

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/motoboys/me` | ○ | ○ | ○ | ● | ● | Perfil do motoboy autenticado |
| PATCH | `/v1/motoboys/me/status` | ○ | ○ | ○ | ● | ● | online / offline / busy |
| GET | `/v1/entregas` | ◐ | ○ | ○ | ◐ | ● | Paciente: entregas dos próprios pedidos; motoboy: ofertas/corridas próprias |
| GET | `/v1/entregas/{id}` | ◐ | ○ | ○ | ◐ | ● | Ownership paciente (pedido) ou motoboy (corrida) |
| POST | `/v1/entregas/{id}/aceitar` | ○ | ○ | ○ | ● | ● | Aceitar corrida — motoboy |
| PATCH | `/v1/entregas/{id}/status` | ○ | ○ | ○ | ◐ | ● | Atualizar status da própria corrida |
| GET | `/v1/entregas/{id}/rastreamento` | ◐ | ○ | ○ | ◐ | ● | Paciente lê pontos; motoboy da corrida também |
| POST | `/v1/entregas/{id}/rastreamento` | ○ | ○ | ○ | ● | ● | Enviar GPS — motoboy da corrida |

### 2.7 Comms

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/notificacoes` | ◐ | ◐ | ◐ | ◐ | ● | Inbox do próprio usuário |
| PATCH | `/v1/notificacoes/{id}/lida` | ◐ | ◐ | ◐ | ◐ | ● | Só notificação própria |
| GET | `/v1/chat/salas` | ◐ | ◐ | ◐ | ◐ | ● | Salas em que o usuário é participante |
| POST | `/v1/chat/salas` | ◐ | ◐ | ◐ | ◐ | ● | Criar sala no contexto permitido (consulta/pedido/entrega) |
| GET | `/v1/chat/salas/{id}/mensagens` | ◐ | ◐ | ◐ | ◐ | ● | Histórico se membro da sala |
| POST | `/v1/chat/salas/{id}/mensagens` | ◐ | ◐ | ◐ | ◐ | ● | Enviar se membro da sala |
| GET | `/v1/realtime` | ● | ● | ● | ● | ● | WebSocket autenticado (proxy Gateway → Comms) |

### 2.8 Support

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/tickets` | ◐ | ◐ | ◐ | ◐ | ● | Próprios tickets; admin vê todos |
| POST | `/v1/tickets` | ● | ● | ● | ● | ● | Abrir ticket |
| GET | `/v1/tickets/{id}` | ◐ | ◐ | ◐ | ◐ | ● | Só ticket próprio (ou admin) |
| POST | `/v1/tickets/{id}/comentarios` | ◐ | ◐ | ◐ | ◐ | ● | Comentar no próprio ticket |
| GET | `/v1/artigos-ajuda` | ● | ● | ● | ● | ● | Base de ajuda (leitura autenticada) |
| GET | `/v1/artigos-ajuda/{id}` | ● | ● | ● | ● | ● | Ler artigo |

### 2.9 Files

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| POST | `/v1/objetos` | ● | ● | ● | ● | ● | Metadata + URL de upload (escopo do ator) |
| GET | `/v1/objetos/{id}` | ◐ | ◐ | ◐ | ◐ | ● | Metadata se dono ou autorizado |
| POST | `/v1/objetos/{id}/url-assinada` | ◐ | ◐ | ◐ | ◐ | ● | Download assinada — ownership |
| POST | `/v1/objetos/{id}/confirmar` | ◐ | ◐ | ◐ | ◐ | ● | Confirmar upload próprio |
| POST | `/v1/pdf/jobs` | ◐ | ● | ○ | ○ | ● | Enfileirar PDF (ex.: receita — médico; paciente pode solicitar cópia autorizada) |
| GET | `/v1/pdf/jobs/{id}` | ◐ | ◐ | ○ | ○ | ● | Status do job do solicitante |

### 2.10 Search

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/search` | ● | ● | ● | ● | ● | Resultados filtrados por papel e ownership |

### 2.11 Audit

| Método | Endpoint | P | M | F | B | A | Observação |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| GET | `/v1/audit/events` | ○ | ○ | ○ | ○ | ● | Consulta de trilha — só admin |
| POST | `/v1/internal/audit/ingest` | ○ | ○ | ○ | ○ | ○ | **Internal** — ingestão pelos serviços |

---

## 3. Regras de ownership (autorização fina)

RBAC no JWT **não basta**. Cada serviço deve garantir:

| Papel | Regra |
|---|---|
| **paciente** | Vê e altera apenas **próprios** pedidos, consultas, triagens, prescrições recebidas, tickets, notificações, payment-intents, assinaturas, faturas e entregas ligadas aos seus pedidos. |
| **farmacia** | Opera apenas a **própria loja**: produtos, estoque contextual, pedidos da farmácia, settlements da loja, chats de pedido da loja. |
| **motoboy** | Opera apenas **próprias corridas**: perfil `motoboys/me`, aceitar/atualizar status, GPS e chats de entrega atribuídos a si. |
| **medico** | Opera **próprias** consultas e prescrições emitidas; pode ler dados do **paciente do atendimento** (consulta/triagem/prescrição daquele vínculo), sem acesso a histórico clínico alheio fora do atendimento. |
| **admin** | **Break-glass**: acesso amplo para suporte, moderação, estornos, alteração de papéis e auditoria. Toda ação sensível deve gerar evento em **Audit** (quem, o quê, quando, request id). |

### Princípios de implementação

1. Comparar `X-User-Id` (e tenant farmácia / vínculo médico-consulta) com o dono do recurso no banco.
2. Negar com `403` (autenticado sem permissão) vs `401` (sem autenticação) vs `404` quando a política for anti-enumeração.
3. Endpoints `/v1/internal/*` e webhooks **não** aceitam papéis de usuário via Gateway público; validar rede + credencial de serviço / assinatura.
4. Testes negativos obrigatórios: usuário A não lê recurso de B (mesmo papel).

---

## 4. Resumo operacional

| Camada | Responsabilidade |
|---|---|
| Gateway | Único ingresso público; valida JWT; injeta `X-User-Id`, `X-Roles`, `X-Request-Id` |
| Identity | Emite tokens; papéis canônicos; admin altera roles |
| Serviço de domínio | RBAC da matriz + ownership da seção 3 |
| Audit | Trilha admin; ingest só interno |

Referências: [ADR-004](./adrs/ADR-004-auth-rbac.md), [Glossário](./GLOSSARIO.md), [contratos OpenAPI](./contracts/openapi/).
