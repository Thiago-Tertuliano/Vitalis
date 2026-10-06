#!/usr/bin/env bash
# Cria um novo microserviço a partir deste template.
# Uso:
#   ./scripts/new-service.sh Vitalis-billing github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-billing
set -euo pipefail

NAME="${1:-}"
MODULE="${2:-}"
if [[ -z "$NAME" || -z "$MODULE" ]]; then
  echo "uso: $0 <NomeRepo> <module-path>"
  echo "ex.: $0 Vitalis-billing github.com/Rede-Medica-D-Excelencia-Vitalis/Vitalis-billing"
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$(cd "$ROOT/../.." && pwd)/$NAME"
if [[ -e "$DEST" ]]; then
  echo "destino já existe: $DEST"
  exit 1
fi

mkdir -p "$DEST"
cp -R "$ROOT"/. "$DEST"/
rm -rf "$DEST/.git" 2>/dev/null || true

# Linux/macOS sed; no Windows use Git Bash ou ajuste manualmente o go.mod
if sed --version >/dev/null 2>&1; then
  find "$DEST" -type f \( -name '*.go' -o -name 'go.mod' -o -name 'README.md' -o -name '.env.example' \) \
    -exec sed -i "s|github.com/Rede-Medica-D-Excelencia-Vitalis/vitalis-service-template|$MODULE|g" {} +
  find "$DEST" -type f \( -name '*.go' -o -name 'go.mod' -o -name 'README.md' -o -name '.env.example' -o -name 'Makefile' \) \
    -exec sed -i "s|vitalis-service-template|$NAME|g" {} +
else
  echo "Ajuste manualmente o module path em go.mod para: $MODULE"
fi

echo "Criado: $DEST"
echo "Próximos passos:"
echo "  cd \"$DEST\""
echo "  cp .env.example .env"
echo "  go mod tidy"
echo "  docker compose up --build"
