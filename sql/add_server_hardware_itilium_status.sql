-- Статус сервера по данным Итилиум (rack и blade):
-- 0 - не указано, 1 - в работе, 2 - выведен, 3 - списан, 4 - свободно, 5 - на складе, 6 - зарезервирован.
-- Обычно колонку добавляет orm.RunSyncdb; скрипт - для ручного применения.

-- --- PostgreSQL ---
ALTER TABLE server_hardware
    ADD COLUMN IF NOT EXISTS itilium_status INTEGER NOT NULL DEFAULT 0;

-- --- SQLite3 (повторный ALTER упадёт - нормально) ---
-- ALTER TABLE server_hardware ADD COLUMN itilium_status INTEGER NOT NULL DEFAULT 0;
