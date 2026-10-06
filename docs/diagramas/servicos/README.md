# Diagramas por microsserviço — Vitalis (2.1 → 2.6)

Cada serviço tem **6 imagens** (uma por entrega) + o `.md` com o texto para o draw.io.

**Total:** 12 serviços × 6 = **72 PNGs** em `docs/diagramas/servicos/`.

| Código | Entrega | Sufixo do arquivo |
|---|---|---|
| 2.1 | C4 Componentes | `*-2.1-components.png` |
| 2.2 | Contexto | `*-2.2-context.png` |
| 2.3 | Modelo de dados / ER | `*-2.3-er.png` ou `gateway-2.3-data.png` |
| 2.4 | State machine | `*-2.4-state.png` |
| 2.5 | Sequência | `*-2.5-sequence.png` |
| 2.6 | Threat / confiança | `*-2.6-threat.png` |

---

## Catálogo completo

### Gateway (`Vitalis-gateway` :8080)
| # | Arquivo | Doc |
|---|---|---|
| 2.1 | [gateway-2.1-components.png](./gateway-2.1-components.png) | [gateway.md](./gateway.md) |
| 2.2 | [gateway-2.2-context.png](./gateway-2.2-context.png) | |
| 2.3 | [gateway-2.3-data.png](./gateway-2.3-data.png) | |
| 2.4 | [gateway-2.4-state.png](./gateway-2.4-state.png) | |
| 2.5 | [gateway-2.5-sequence.png](./gateway-2.5-sequence.png) | |
| 2.6 | [gateway-2.6-threat.png](./gateway-2.6-threat.png) | |

### Identity (`:8081` / `pg_identity`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `identity-2.1-components.png` … `identity-2.6-threat.png` |
| Doc | [identity.md](./identity.md) |

### Clinical (`:8082` / `pg_clinical`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `clinical-2.1-components.png` … `clinical-2.6-threat.png` |
| Doc | [clinical.md](./clinical.md) |

### Commerce (`:8083` / `pg_commerce`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `commerce-2.1-components.png` … `commerce-2.6-threat.png` |
| Doc | [commerce.md](./commerce.md) |

### Orders (`:8084` / `pg_orders`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `orders-2.1-components.png` … `orders-2.6-threat.png` |
| Doc | [orders.md](./orders.md) |

### Delivery (`:8085` / `pg_delivery`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `delivery-2.1-components.png` … `delivery-2.6-threat.png` |
| Doc | [delivery.md](./delivery.md) |

### Billing — vitrine (`:8086` / `pg_billing`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `billing-2.1-components.png` … `billing-2.6-threat.png` |
| Doc | [billing.md](./billing.md) |

### Comms (`:8087` / `pg_comms`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `comms-2.1-components.png` … `comms-2.6-threat.png` |
| Doc | [comms.md](./comms.md) |

### Support (`:8088` / `pg_support`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `support-2.1-components.png` … `support-2.6-threat.png` |
| Doc | [support.md](./support.md) |

### Files (`:8089` / `pg_files` + MinIO\*)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `files-2.1-components.png` … `files-2.6-threat.png` |
| Doc | [files.md](./files.md) |

### Search (`:8090` / `pg_search`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `search-2.1-components.png` … `search-2.6-threat.png` |
| Doc | [search.md](./search.md) |

### Audit (`:8091` / `pg_audit`)
| # | Arquivo |
|---|---|
| 2.1–2.6 | `audit-2.1-components.png` … `audit-2.6-threat.png` |
| Doc | [audit.md](./audit.md) |

---

## Sheets + extras (legado resumido)

Ainda úteis como visão “tudo numa página”:
- `sheet-*.png` (12)
- `extra-billing-er-state.png`, `extra-clinical-er-state.png`, `extra-orders-delivery-er-state.png`, `extra-identity-er.png`

---

## Como usar no draw.io

1. Abra o PNG da seção (ex.: `billing-2.4-state.png`).
2. Abra o `.md` do serviço para copiar rótulos.
3. Uma aba draw.io por imagem (72 abas no total, ou 1 arquivo `.drawio` por serviço com 6 abas).

## Legenda

- Sync = HTTP · Async = evento/Redis · `*` = próprio ou provedor · Stack = **Go + PostgreSQL**
