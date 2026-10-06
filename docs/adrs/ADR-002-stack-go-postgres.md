# ADR-002 — Stack: Go + PostgreSQL (+ Redis, MinIO, Docker local)

| Campo | Valor |
|---|---|
| Status | Aceito |
| Data | 2026-09-12 |
| Decisores | Equipe Vitalis |
| Relacionados | ADR-001 (microserviços + polyrepo), ADR-003 (comunicação) |

---

## Contexto

Com a decisão do ADR-001 (microserviços completos em polyrepo e reescrita do legado Node), é necessário **fixar a stack técnica** do backend novo para:

- padronizar todos os 12 serviços;
- evitar misturar Node/MySQL no alvo;
- permitir desenvolvimento e validação **apenas em ambiente Docker próprio** (sem cloud obrigatória);
- construir **todos** os microsserviços de ponta a ponta (sem “fases” que cortam Search/Audit do escopo).

---

## Decisão

### 1. Linguagem

- **Go** em todos os microsserviços de backend e no Gateway.
- Versão padronizada: **Go 1.27.x** (estável mais recente na data desta decisão: **1.27.1**).
- Política: acompanhar a **última linha estável** do Go no projeto; pin da versão no `go.mod` / CI / imagens Docker do time.
- **Proibido:** Node.js / Express no backend novo.

### 2. Banco de dados

- **PostgreSQL** como banco padrão de domínio.
- **Database-per-service** à risca: um database dedicado por serviço (`pg_identity`, `pg_clinical`, `pg_billing`, …).
- **Proibido:**
  - MySQL como DB padrão dos microserviços novos;
  - schema compartilhado de escrita entre serviços;
  - “um Postgres com tabelas de todos” sem isolamento.

### 3. HTTP framework (Go)

- Padrão oficial do projeto: **chi**.
- Motivo da escolha (avaliação do time): leve, idiomático, middleware explícito, adequado a Gateway e APIs; você já tem experiência com **chi** e **echo**.
- **Echo** fica como alternativa aceitável apenas se um serviço específico justificar — o default em templates/CI é **chi**.

### 4. Migrations

- Padrão oficial: **goose**.
- Cada repo de serviço versiona suas migrations no próprio repositório.
- Motivo: simples, Go-friendly, encaixa bem em polyrepo.

### 5. Redis (plataforma)

- **Redis entra como padrão da plataforma** desde o início (não “depois”).
- Usos: cache, rate-limit (quando aplicável), **streams / barramento de eventos** e apoio a outbox/consumers.
- Escopo: todos os serviços que publicam/consomem eventos usam o padrão Redis da plataforma.

### 6. Object storage (Files)

- **MinIO local** com API compatível **S3** para desenvolvimento, testes e validação do `Vitalis-files`.
- Modelado como integração `*` (próprio / S3-compat), validável sem cloud externa obrigatória.

### 7. Dinheiro / tipos

- Valores monetários: **`NUMERIC` no Postgres** + tipo decimal seguro no Go.
- **Proibido:** `float` / `float64` para dinheiro.

### 8. Ambiente de execução

- **Não há compromisso de deploy em cloud** neste redesign.
- Se rodar, roda no **ambiente Docker próprio** do time (`Vitalis-infra` / Compose).
- Demo, testes com ≥10 usuários simultâneos e validação E2E ocorrem nesse ambiente local/Docker.

### 9. Escopo de construção

- **Sem cortes por “fase 1 / fase 2” no sentido de adiar serviços do catálogo.**
- O projeto constrói e valida os **12 microsserviços** (+ clientes) de ponta a ponta, com testes e validação completa.
- A ordem de implementação pode ser sequencial por dependência, mas o **alvo entregue** inclui o conjunto completo definido no ADR-001.

---

## Critério de sucesso deste ADR

- [ ] Todos os serviços novos compilam em **Go 1.27.x** com template chi + goose + Postgres próprio.
- [ ] Compose sobe Postgres (multi-DB), Redis e MinIO local.
- [ ] Nenhum serviço novo depende de Node ou MySQL.
- [ ] Billing (e demais) usam dinheiro sem float.

---

## Consequências

### Positivas

- Stack uniforme e alinhada a microserviços rigorosos.
- Ambiente reproduzível só com Docker.
- Redis/MinIO desde o início evitam retrabalho de “encaixar depois”.

### Negativas / trade-offs

- Usar sempre a última Go exige disciplina de atualização no CI.
- Construir os 12 serviços E2E aumenta esforço (aceito pelo ADR-001).
- Sem cloud: responsabilidade total de Compose/observabilidade local.

### Mitigações

- Pin de versão Go + imagem base única no `Vitalis-infra`.
- Template de serviço (chi + goose + health + outbox Redis).
- Documentar port map e env matrix no infra.

---

## Alternativas consideradas

| Alternativa | Decisão |
|---|---|
| Go 1.22 “LTS conservador” | Rejeitado — time prefere última estável (1.27.x) |
| Echo como default | chi escolhido como padrão; echo só se necessário |
| golang-migrate | goose escolhido por simplicidade em polyrepo |
| Filesystem local sem MinIO | Rejeitado — MinIO S3-compat para validar Files de verdade |
| Manter MySQL do legado | Rejeitado — Postgres por serviço |
| Deploy cloud agora | Fora de escopo — só Docker próprio |

---

## Notas

- Comunicação sync/async e contratos de eventos → **ADR-003**.
- JWT/RBAC → **ADR-004**.
- Detalhes de Billing (ledger, PIX, split) → **ADR-005**.
