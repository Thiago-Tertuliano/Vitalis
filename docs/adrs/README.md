# ADRs — Índice

Decisões de arquitetura da reescrita Vitalis (microserviços Go).

| ADR | Título | Status | Arquivo |
|---|---|---|---|
| 001 | Microserviços completos + polyrepo | Aceito | [ADR-001-microservicos-polyrepo.md](./ADR-001-microservicos-polyrepo.md) |
| 002 | Stack Go + PostgreSQL (+ Redis, MinIO, Docker local) | Aceito | [ADR-002-stack-go-postgres.md](./ADR-002-stack-go-postgres.md) |
| 003 | Comunicação HTTP + eventos + Gateway | Aceito | [ADR-003-comunicacao-http-eventos.md](./ADR-003-comunicacao-http-eventos.md) |
| 004 | AuthN/AuthZ JWT RS256 + RBAC | Aceito | [ADR-004-auth-rbac.md](./ADR-004-auth-rbac.md) |
| 005 | Billing vitrine (PIX, cartão, carteira, ledger, split) | Aceito | [ADR-005-billing-vitrine.md](./ADR-005-billing-vitrine.md) |

## Próximo (documentação de produto/contratos)

Concluído neste workspace:

- OpenAPI: `docs/contracts/openapi/`
- Eventos: `docs/contracts/events/CATALOGO_EVENTOS.md`
- Glossário: `docs/GLOSSARIO.md`
- RBAC: `docs/RBAC.md`

Próximo passo natural: templates de repo + `Vitalis-infra` / começar código.
