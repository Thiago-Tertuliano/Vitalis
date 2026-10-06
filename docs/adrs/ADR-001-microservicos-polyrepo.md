# ADR-001 — Microserviços completos + polyrepo

| Campo | Valor |
|---|---|
| Status | Aceito |
| Data | 2026-09-12 |
| Decisores | Equipe Vitalis (Thiago + time) |
| Relacionados | ADR-002 (stack), Plano de Projeto, Arquitetura |

---

## Contexto

A Vitalis hoje possui um backend monolítico modular em Node.js/MySQL (`Vitalis-Backend`) e vários clientes (web/mobile). O ecossistema de produto já é multi-ator (paciente, profissional, farmácia, admin, motoboy).

O time deseja **redesenhar** a plataforma não apenas por prática técnica, mas para:

1. **Demonstrar conhecimento** em um ecossistema funcional completo (TCC / portfólio).
2. **Organizar o domínio** em vários microserviços reais (não monólito modular como destino).
3. **Preparar escala futura** (possível interesse de startup / lançamento).
4. **Permitir trabalho em paralelo** entre pessoas/repos, se o time crescer.

Há risco de complexidade operacional excessiva; por isso o sucesso precisa ser mensurável (sobe tudo + fluxos + carga mínima).

---

## Decisão

1. **Arquitetura-alvo: microserviços completos** — 12 peças de backend:
   - `Vitalis-gateway`
   - `Vitalis-identity`
   - `Vitalis-clinical`
   - `Vitalis-commerce`
   - `Vitalis-orders`
   - `Vitalis-delivery`
   - `Vitalis-billing`
   - `Vitalis-comms`
   - `Vitalis-support`
   - `Vitalis-files`
   - `Vitalis-search`
   - `Vitalis-audit`

2. **Clientes (frontends):** mantidos/evoluídos como **6 clientes** separados (polyrepo de UI), consumindo o Gateway:
   - Web Paciente, Web Profissional, Web Farmácias, Web Admin
   - Mobile Paciente, Mobile Motoboy

3. **Organização Git: polyrepo** — **um repositório por serviço/cliente**, mais repositórios transversais (`Vitalis-infra`, `Vitalis-contracts`, opcionalmente `Vitalis-go-kit`). **Não** haverá monorepo.

4. **Relação com o legado Node:**
   - A lógica de negócio útil do `Vitalis-Backend` será **absorvida / reimplementada em Go**.
   - O monólito Node **não** permanece como backend paralelo em operação.
   - O histórico Git do legado permanece como arquivo; código abandonado/não usado pode ser descartado na nova base.

5. **Não é destino:** monólito modular “para sempre”. Microserviços são o alvo explícito deste redesign.

---

## Critério de sucesso (Definition of Done deste ADR)

Este ADR é considerado **cumprido na prática** quando:

- [ ] Todos os microserviços de backend sobem de forma coordenada (via `Vitalis-infra` / Compose).
- [ ] Os fluxos principais funcionam de ponta a ponta (no mínimo: autenticação, assinatura/pagamento, jornada clínica e pedido/entrega conforme escopo MVP).
- [ ] Há validação de uso com **pelo menos 10 usuários simultâneos** exercitando o sistema.
- [ ] A suíte de testes cobre o serviço de forma completa o suficiente para demo/defesa (não apenas “sobe o container”).

---

## Consequências

### Positivas

- Fronteiras de domínio claras e alinhadas ao produto multi-ator.
- Repos independentes → CI/deploy e ownership por serviço.
- Narrativa forte para TCC/portfólio (“plataforma distribuída”).
- Base pronta para escala e time paralelo no futuro.

### Negativas / trade-offs

- Maior custo operacional (rede, contratos, observabilidade, Compose).
- Necessidade rígida de contratos (`Vitalis-contracts`) para evitar drift entre repos.
- Risco de “monólito distribuído” se houver DB compartilhado — **proibido** pelo desenho (1 Postgres por serviço; ver ADR-002).
- Reescrita em nova stack implica esforço; legado Node só como referência de regras.

### Mitigações

- Ordem de entrega por fases (Identity/Gateway → Billing → Clinical → Commerce/Orders/Delivery → …).
- Contratos OpenAPI/AsyncAPI versionados.
- Critério de sucesso com carga mínima (10 usuários) e fluxos E2E obrigatórios.

---

## Alternativas consideradas

| Alternativa | Por que não |
|---|---|
| Monólito modular em Go | Mais simples, mas não atende o objetivo de microserviços + prática/portfólio + paralelismo |
| Monorepo | Rejeitado explicitamente pela equipe |
| Manter Node + extrair serviços aos poucos em paralelo longo | Duas stacks/backends aumentam custo; decisão é reescrever em Go e desligar o monólito |

---

## Notas

- Detalhe de stack (Go, Postgres, etc.) → **ADR-002**.
- Comunicação entre serviços e outbox → **ADR-003**.
- AuthN/AuthZ → **ADR-004**.
- Billing (vitrine) → **ADR-005**.
