# Contratos Vitalis (`Vitalis-contracts`)

Fonte da verdade de APIs e eventos do redesign em microserviços.

| Pasta | Conteúdo |
|---|---|
| `openapi/` | OpenAPI 3.0.3 de **todos** os 12 serviços |
| `events/CATALOGO_EVENTOS.md` | Catálogo async (Redis Streams + outbox) |

## OpenAPI

| Arquivo | Serviço | Porta |
|---|---|---|
| `gateway.yaml` | Gateway (+ tabela de roteamento) | 8080 |
| `identity.yaml` | Identity | 8081 |
| `clinical.yaml` | Clinical | 8082 |
| `commerce.yaml` | Commerce | 8083 |
| `orders.yaml` | Orders | 8084 |
| `delivery.yaml` | Delivery | 8085 |
| `billing.yaml` | Billing | 8086 |
| `comms.yaml` | Comms | 8087 |
| `support.yaml` | Support | 8088 |
| `files.yaml` | Files | 8089 |
| `search.yaml` | Search | 8090 |
| `audit.yaml` | Audit | 8091 |

## Regras

1. Clientes públicos → **somente Gateway**.
2. Mudança de contrato = atualizar este pacote **antes/junto** do código.
3. Erro padrão: schema `Error` (`erro`, `codigo`, `request_id`).
4. Billing: `Idempotency-Key` obrigatório em POSTs de cobrança/assinatura (ADR-005).
5. Regenerar YAMLs: `python docs/contracts/_gen_openapi.py` (se precisar).

## Docs relacionadas

- Glossário: `docs/GLOSSARIO.md`
- RBAC: `docs/RBAC.md`
- ADRs: `docs/adrs/`
