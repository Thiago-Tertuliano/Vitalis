# Template de microserviço Go — Vitalis

Scaffold oficial alinhado aos ADRs:

- **ADR-001** polyrepo / 1 serviço = 1 repo  
- **ADR-002** Go 1.27 · chi · goose · Postgres · Redis · Docker local  
- **ADR-003** outbox + Redis Streams · envelope de eventos  
- **ADR-004** headers `X-User-Id` / `X-Roles` / `X-Request-Id` (Gateway)

## Layout

```
cmd/api          # HTTP API
cmd/worker       # publica outbox → Redis Streams
internal/
  config/
  domain/        # entidades
  app/           # casos de uso
  port/          # interfaces
  adapter/http/  # chi + middleware
  adapter/postgres/
  adapter/redis/
  outbox/
  platform/logger/
migrations/      # goose
api/             # espelho OpenAPI (copie de docs/contracts)
Dockerfile
docker-compose.yml
```

## Como criar um serviço novo

### Opção A — script (Git Bash / Linux / macOS)

```bash
cd templates/vitalis-go-service
chmod +x scripts/new-service.sh
./scripts/new-service.sh Vitalis-billing github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-billing
```

### Opção B — manual (Windows)

1. Copie a pasta `templates/vitalis-go-service` para `Vitalis-<nome>`.
2. Em `go.mod` e imports, troque  
   `github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template`  
   pelo module do novo repo.
3. Ajuste `SERVICE_NAME`, portas e `POSTGRES_DB` no `.env` / Compose.
4. Remova o domínio `examples` e implemente o bounded context (Identity, Billing, …).
5. Copie o OpenAPI de `docs/contracts/openapi/<servico>.yaml` para `api/openapi.yaml`.

## Subir local

```bash
cp .env.example .env
go mod tidy
# migrations (instale goose: go install github.com/pressly/goose/v3/cmd/goose@latest)
set DATABASE_URL=postgres://vitalis:vitalis@localhost:5432/vitalis_service?sslmode=disable
make docker-up
# ou: docker compose up postgres redis -d && make migrate-up && make run-api
```

Smoke test:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl -X POST http://localhost:8080/v1/examples -H "Content-Type: application/json" -d "{\"name\":\"ok\"}"
```

## DoD mínimo do template (por serviço)

- [ ] `/health` e `/ready`
- [ ] Migrations goose versionadas
- [ ] OpenAPI em `api/`
- [ ] Outbox + worker
- [ ] Dockerfile + compose unitário
- [ ] CI build/test
- [ ] Sem segredos commitados (só `.env.example`)

## Próximos serviços sugeridos a partir deste template

1. `Vitalis-identity`  
2. `Vitalis-gateway` (variante: sem Postgres de domínio / só proxy)  
3. `Vitalis-billing`
