#!/usr/bin/env bash
# Восстановление PostgreSQL из базовой копии (pg_basebackup.sh) + WAL-архива (plan.md E8.6).
#
# РАЗРУШИТЕЛЬНО: стирает существующий том данных целевого контейнера. Работает НЕ с боевым
# контейнером/томом по умолчанию — только с параметрами, переданными явно, чтобы случайный запуск
# без аргументов не задел рабочую БД (см. блок проверки аргументов ниже).
#
# Использование:
#   scripts/backup/restore.sh <каталог-базовой-копии> <новый-том-данных> <новый-контейнер> \
#       [--target-time '2026-09-28 12:00:00+00'] [--confirm]
#
# Без --confirm скрипт только печатает план и останавливается — предохранитель для ручного запуска.
# target-time — необязательная точка восстановления (recovery_target_time); без неё — докатка до
# последнего доступного сегмента в архиве («как можно ближе к моменту отказа», RPO ограничен
# archive_timeout из docker-compose.yml).
#
# Переменные окружения: CMDB_WAL_ARCHIVE_VOLUME (том с архивом WAL, по умолчанию cmdb_cmdb_wal_archive),
# CMDB_PG_USER (по умолчанию cmdb), CMDB_PG_IMAGE (по умолчанию postgres:18).
set -euo pipefail

BACKUP_DIR="${1:?использование: restore.sh <каталог-базовой-копии> <новый-том-данных> <новый-контейнер> [--target-time TS] [--confirm]}"
DATA_VOLUME="${2:?нужен том данных — НЕ том рабочей БД, если это не намеренное восстановление на её место}"
CONTAINER="${3:?нужно имя нового контейнера}"
shift 3

TARGET_TIME=""
CONFIRM=0
while [ $# -gt 0 ]; do
    case "$1" in
        --target-time) TARGET_TIME="$2"; shift 2 ;;
        --confirm) CONFIRM=1; shift ;;
        *) echo "restore.sh: неизвестный аргумент $1" >&2; exit 1 ;;
    esac
done

WAL_ARCHIVE_VOLUME="${CMDB_WAL_ARCHIVE_VOLUME:-cmdb_cmdb_wal_archive}"
PGUSER="${CMDB_PG_USER:-cmdb}"
PGIMAGE="${CMDB_PG_IMAGE:-postgres:18}"

echo "План восстановления:"
echo "  базовая копия:      $BACKUP_DIR"
echo "  новый том данных:   $DATA_VOLUME (будет создан заново, если существует — данные в нём теряются)"
echo "  том архива WAL:     $WAL_ARCHIVE_VOLUME (используется для докатки, не изменяется)"
echo "  новый контейнер:    $CONTAINER"
echo "  точка восстановления: ${TARGET_TIME:-<последний доступный WAL>}"

if [ "$CONFIRM" != "1" ]; then
    echo
    echo "Предохранитель: план не выполнен — повторите с --confirm, когда план проверен."
    exit 0
fi

if [ ! -f "$BACKUP_DIR/base.tar.gz" ]; then
    echo "restore.sh: $BACKUP_DIR/base.tar.gz не найден — это не каталог pg_basebackup.sh?" >&2
    exit 1
fi

docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
docker volume rm "$DATA_VOLUME" >/dev/null 2>&1 || true
docker volume create "$DATA_VOLUME" >/dev/null

# Настройки докатки (restore_command всегда; recovery_target_time — только если задан) — собраны
# в обычном файле на хосте, а не внутри -c "..." контейнера: вложенное экранирование одинарных/
# двойных кавычек в одну строку -c слишком хрупко (поймано именно так, при первом реальном прогоне
# скрипта — command substitution внутри уже экранированной строки не пережил кавычки TARGET_TIME).
AUTOCONF="$(mktemp)"
trap 'rm -f "$AUTOCONF"' EXIT
{
    echo "restore_command = 'cp /wal_archive/%f %p'"
    if [ -n "$TARGET_TIME" ]; then
        echo "recovery_target_time = '$TARGET_TIME'"
        echo "recovery_target_action = 'promote'"
    fi
} > "$AUTOCONF"

# Распаковка базовой копии в новый том — отдельным недолгим контейнером с примонтированными
# каталогом копии (только чтение) и новым томом (запись), от имени postgres (владелец итоговых
# файлов данных должен совпадать с тем, от чьего имени потом стартует сервер). Том монтируется в
# ТОЧНОСТИ туда же (/var/lib/postgresql, единый mount), куда его смонтирует рабочий контейнер ниже
# — данные должны лежать в подкаталоге 18/docker КАК ОБЫЧНЫЕ файлы внутри точки монтирования, а не
# сам том быть примонтирован НА этот подкаталог (несовпадение здесь поймано на первом реальном
# прогоне: образ postgres 18 отказывался стартовать, приняв данные за формат до pg_ctlcluster).
docker run --rm \
    -v "$DATA_VOLUME:/var/lib/postgresql" \
    -v "$BACKUP_DIR:/backup:ro" \
    -v "$AUTOCONF:/tmp/extra.conf:ro" \
    --user root \
    "$PGIMAGE" bash -c '
        set -e
        mkdir -p /var/lib/postgresql/18/docker
        tar -xzf /backup/base.tar.gz -C /var/lib/postgresql/18/docker
        mkdir -p /var/lib/postgresql/18/docker/pg_wal
        tar -xzf /backup/pg_wal.tar.gz -C /var/lib/postgresql/18/docker/pg_wal
        touch /var/lib/postgresql/18/docker/recovery.signal
        cat /tmp/extra.conf >> /var/lib/postgresql/18/docker/postgresql.auto.conf
        chown -R postgres:postgres /var/lib/postgresql
    '

docker run -d --name "$CONTAINER" \
    -v "$DATA_VOLUME:/var/lib/postgresql" \
    -v "$WAL_ARCHIVE_VOLUME:/wal_archive:ro" \
    -e POSTGRES_USER="$PGUSER" -e POSTGRES_PASSWORD=cmdb -e POSTGRES_DB=cmdb \
    "$PGIMAGE" >/dev/null

echo "Контейнер $CONTAINER запущен, восстановление WAL идёт в фоне (проверить: docker logs -f $CONTAINER)."
echo "Готовность: docker exec $CONTAINER pg_isready -U $PGUSER"
