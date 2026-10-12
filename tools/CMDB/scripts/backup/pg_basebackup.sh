#!/usr/bin/env bash
# Базовая копия PostgreSQL (plan.md E8.6) — снимается с работающего сервера через pg_basebackup
# внутри контейнера; непрерывный WAL-архив (docker-compose.yml: archive_mode/archive_command)
# даёт восстановление на момент времени поверх этой копии, см. restore.sh — RPO ограничен снизу
# archive_timeout (5 мин), не интервалом между запусками этого скрипта.
#
# Использование:
#   scripts/backup/pg_basebackup.sh [каталог-назначения]
# По умолчанию — /var/backups/cmdb/base-<UTC-метка через ISO 8601 без разделителей>.
# Переменные окружения: CMDB_PG_CONTAINER (имя контейнера, по умолчанию cmdb-postgres-1),
# CMDB_PG_USER (по умолчанию cmdb).
set -euo pipefail

CONTAINER="${CMDB_PG_CONTAINER:-cmdb-postgres-1}"
PGUSER="${CMDB_PG_USER:-cmdb}"
DEST="${1:-/var/backups/cmdb/base-$(date -u +%Y%m%dT%H%M%SZ)}"

if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
    echo "scripts/backup/pg_basebackup.sh: контейнер $CONTAINER не найден" >&2
    exit 1
fi

mkdir -p "$(dirname "$DEST")"
mkdir -p "$DEST"

WORKDIR="/tmp/cmdb-basebackup-$$"
docker exec "$CONTAINER" rm -rf "$WORKDIR"
docker exec "$CONTAINER" mkdir -p "$WORKDIR"
# -Ft (tar) + -z (сжатие) + -Xs (стримить нужный для консистентности WAL в тот же снимок, без
# отдельного шага) + -P (прогресс в лог) — стандартный самодостаточный формат: восстановление не
# требует WAL с сервера отдельно для согласованности самой базовой копии (только для докатки
# вперёд по времени, см. restore.sh).
docker exec "$CONTAINER" pg_basebackup -U "$PGUSER" -D "$WORKDIR" -Ft -z -Xs -P
docker cp "$CONTAINER:$WORKDIR/." "$DEST"
docker exec "$CONTAINER" rm -rf "$WORKDIR"

echo "Базовая копия сохранена: $DEST"
ls -la "$DEST"
