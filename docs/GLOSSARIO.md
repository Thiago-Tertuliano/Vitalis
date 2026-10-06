# Glossário ubíquo — Vitalis

Termos em **português**, com nome técnico em inglês quando existir. Use estes nomes nos ADRs, OpenAPI, código e monografia.

---

## A — Plataforma e arquitetura

| Termo (PT) | Técnico (EN) | Significado | Exemplo |
|---|---|---|---|
| Microsserviço | Microservice | Processo/deploy independente com DB próprio | `Vitalis-billing` |
| Polyrepo | Polyrepo | Um repositório Git por serviço/cliente | `Vitalis-orders` separado de `Vitalis-delivery` |
| Gateway | API Gateway | Única porta pública HTTP/WS dos clientes | `:8080` |
| Contrato | Contract / OpenAPI | Especificação da API versionada | `docs/contracts/openapi/billing.yaml` |
| Evento de domínio | Domain event | Fato que já aconteceu, publicado no barramento | `payment.succeeded` |
| Envelope de evento | Event envelope | Metadados padrão em torno do `data` | `id`, `type`, `producer`, `version`… |
| Outbox transacional | Transactional Outbox | Grava evento na mesma TX do Postgres antes de publicar | tabela `outbox_events` |
| Barramento | Event bus | Infra de mensagens entre serviços | Redis Streams |
| Database por serviço | Database-per-service | Cada serviço só escreve no seu Postgres | `pg_clinical` |
| Integração plugável | Pluggable integration (`*`) | Pode ser própria ou provedor externo | `* API Pagamentos` |

---

## B — Identidade e acesso

| Termo (PT) | Técnico (EN) | Significado | Exemplo |
|---|---|---|---|
| Usuário | User | Conta autenticável | paciente João |
| Papel | Role | Função RBAC no token | `medico`, `farmacia` |
| Token de acesso | Access token (JWT) | Credencial curta para APIs | Bearer no header |
| Token de atualização | Refresh token | Credencial longa, httpOnly cookie (web) | rotação no logout/refresh |
| JWKS | JWKS | Conjunto de chaves públicas para validar RS256 | `GET /v1/auth/jwks.json` |
| Autenticação | Authentication (AuthN) | Quem é você | login |
| Autorização | Authorization (AuthZ) | O que você pode fazer | RBAC + dono do recurso |
| Dono do recurso | Ownership check | Checagem fina no serviço | pedido só do próprio user_id |

**Papéis oficiais:** `paciente`, `medico`, `farmacia`, `motoboy`, `admin`.

---

## C — Clínico

| Termo (PT) | Técnico (EN) | Significado | Exemplo |
|---|---|---|---|
| Consulta | Appointment | Agendamento/atendimento | teleconsulta 15:00 |
| Triagem | Triage | Questionário inicial | triagem online pré-consulta |
| Prescrição | Prescription | Receita médica emitida | PDF + itens |
| Sessão de vídeo | Video session | Sala de teleconsulta | WebRTC sinalizada pelo Clinical |
| Plano ativo (gate) | Plan gate | Bloqueio de feature se assinatura inativa | Clinical consulta Billing |
| Especialidade | Specialty | Área médica | cardiologia |
| Disponibilidade | Availability slot | Horário livre do médico | slot 10:30 |

---

## D — Farmácia, pedido e entrega

| Termo (PT) | Técnico (EN) | Significado | Exemplo |
|---|---|---|---|
| Farmácia parceira | Pharmacy | Loja no catálogo | Farmácia Centro |
| Produto | Product | Item do catálogo | dipirona 500mg |
| Estoque de vitrine | Inventory | Quantidade disponível/reservada | qty 12, reserved 2 |
| Pedido | Order | Compra de medicamentos | pedido #123 |
| Item do pedido | Order item | Linha do pedido | 2x produto X |
| Entrega | Delivery | Corrida logística do pedido | status `in_transit` |
| Motoboy | Courier | Entregador | app motoboy |
| Rastreamento | Tracking | Pontos GPS / status | lat/lng periodicos |
| Separação | Separating | Farmácia preparando o pedido | status `separating` |

---

## E — Pagamentos e billing

| Termo (PT) | Técnico (EN) | Significado | Exemplo |
|---|---|---|---|
| Intenção de pagamento | PaymentIntent | Cobrança em andamento | PIX pendente |
| PIX | PIX | Meio de pagamento instantâneo BR | QR Code |
| Cartão | Card payment | Cobrança via token do provedor | cartão aprovado |
| Carteira interna | Wallet | Saldo interno Vitalis | crédito R$ 50 |
| Assinatura | Subscription | Plano recorrente do paciente | plano Mensal |
| Fatura | Invoice | Documento de cobrança do plano | fatura setembro |
| Livro-razão / Ledger | Double-entry ledger | Lançamentos débito/crédito | ΣD = ΣC |
| Partida dobrada | Double-entry | Toda movimentação tem contrapartida | captura + receita |
| Rateio / Split | Split / marketplace split | Divisão do valor entre atores | 8% Vitalis + farmácia + motoboy |
| Liquidação | Settlement | Liberação de repasse | payout farmácia |
| Comissão médica | Commission | Parte do médico (consulta/assinatura) | após `appointment.completed` |
| Chave de idempotência | Idempotency-Key | Evita cobrança duplicada | header obrigatório |
| Webhook | Webhook | Callback do provedor de pagamento | `payment.approved` |
| Adapter fake/sandbox | Fake payment adapter | Simula PSP localmente | QR e webhook simulados |
| Centavos | Amount in cents | Valor inteiro em centavos de BRL | `1500` = R$ 15,00 |

### Regras de split (lembrete ADR-005)

- **Pedido farmácia:** Vitalis + farmácia + motoboy (**sem médico**).
- **Consulta/assinatura:** médico recebe parcela conforme regra do plano/consulta.

---

## F — Comunicação, arquivos, busca, auditoria

| Termo (PT) | Técnico (EN) | Significado | Exemplo |
|---|---|---|---|
| Notificação | Notification | Aviso in-app/push/e-mail | “Plano ativo” |
| Chat | Chat room / message | Mensagens entre atores | paciente ↔ médico |
| Tempo real | WebSocket / realtime | Canal persistente | `/v1/realtime` |
| Objeto / arquivo | File object | Metadata + blob no MinIO | receita.pdf |
| URL assinada | Signed URL | Link temporário de download/upload | TTL 5 min |
| Busca unificada | Search index | Índice, não fonte da verdade | `q=dipirona` |
| Trilha de auditoria | Audit trail | Log append-only de acesso/ação | quem viu prontuário |
| Ticket | Support ticket | Chamado de suporte | “não consigo pagar” |
| Artigo de ajuda | Help article | Conteúdo da central de ajuda | “Como remarcar” |

---

## G — Atores (personas)

| Ator (PT) | Role | Cliente típico |
|---|---|---|
| Paciente | `paciente` | Web/Mobile Paciente |
| Médico / Profissional | `medico` | Web Profissional |
| Farmácia | `farmacia` | Web Farmácias |
| Motoboy | `motoboy` | Mobile Motoboy |
| Administrador | `admin` | Web Admin |

---

## H — Erros de API (modelo)

Resposta padrão de erro:

```json
{
  "erro": "Assinatura inativa",
  "codigo": "PLAN_REQUIRED",
  "request_id": "req_123",
  "detalhes": {}
}
```

| codigo (exemplo) | Quando |
|---|---|
| `UNAUTHORIZED` | Sem/ inválido JWT |
| `FORBIDDEN` | Role ou ownership negado |
| `PLAN_REQUIRED` | Feature clínica sem plano |
| `CONFLICT` | Estado inválido / estoque |
| `IDEMPOTENCY_REPLAY` | Mesma key já processada |
| `VALIDATION_ERROR` | Payload inválido |
