-- =============================================================================
-- Мощность (Вт) на server_hardware и blade_systems для агрегации по шкафу/ЦОД.
-- Поля заполняются из OneView (вариант C): enclosure power + environmentalConfiguration.
-- Обычно колонки добавляет orm.RunSyncdb; скрипт — для ручного применения.
-- =============================================================================

-- --- PostgreSQL ---
ALTER TABLE server_hardware
    ADD COLUMN IF NOT EXISTS power_avg24h_w DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE server_hardware
    ADD COLUMN IF NOT EXISTS power_peak24h_w DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE server_hardware
    ADD COLUMN IF NOT EXISTS power_rated_max_w DOUBLE PRECISION NOT NULL DEFAULT 0;

ALTER TABLE blade_systems
    ADD COLUMN IF NOT EXISTS power_avg24h_w DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE blade_systems
    ADD COLUMN IF NOT EXISTS power_peak24h_w DOUBLE PRECISION NOT NULL DEFAULT 0;
ALTER TABLE blade_systems
    ADD COLUMN IF NOT EXISTS power_rated_max_w DOUBLE PRECISION NOT NULL DEFAULT 0;

-- --- SQLite3 (повторный ALTER упадёт — нормально) ---
-- ALTER TABLE server_hardware ADD COLUMN power_avg24h_w REAL NOT NULL DEFAULT 0;
-- ALTER TABLE server_hardware ADD COLUMN power_peak24h_w REAL NOT NULL DEFAULT 0;
-- ALTER TABLE server_hardware ADD COLUMN power_rated_max_w REAL NOT NULL DEFAULT 0;
-- ALTER TABLE blade_systems ADD COLUMN power_avg24h_w REAL NOT NULL DEFAULT 0;
-- ALTER TABLE blade_systems ADD COLUMN power_peak24h_w REAL NOT NULL DEFAULT 0;
-- ALTER TABLE blade_systems ADD COLUMN power_rated_max_w REAL NOT NULL DEFAULT 0;
