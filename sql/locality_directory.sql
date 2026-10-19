-- =============================================================================
-- Справочник населённых пунктов (cmdb_locality) и поле "Код населённого пункта"
-- площадок/ЦОД (cmdb_datacenter_site.locality_code).
-- Обычно таблицы создаёт orm.RunSyncdb; скрипт - для ручного применения.
-- Данные загружаются командой:
--   go run ./tools/localityload -file itilliumimport/parsing/oktmo_data_20261001.csv
-- (источник - ОКТМО Росстата, https://rosstat.gov.ru/opendata/7708234640-oktmo;
-- код = ОКТМО, 11 знаков). Связь площадки со справочником - по коду, не по Id.
-- =============================================================================

-- --- PostgreSQL ---
CREATE TABLE IF NOT EXISTS cmdb_locality (
    id           BIGSERIAL PRIMARY KEY,
    oktmo        VARCHAR(11)  NOT NULL UNIQUE,
    region_code  VARCHAR(2)   NOT NULL,
    region_name  VARCHAR(255) NOT NULL,
    municipality VARCHAR(500),
    type_short   VARCHAR(16),
    name         VARCHAR(500) NOT NULL,
    full_name    VARCHAR(500) NOT NULL
);
CREATE INDEX IF NOT EXISTS cmdb_locality_region_code ON cmdb_locality (region_code);
CREATE INDEX IF NOT EXISTS cmdb_locality_name ON cmdb_locality (name);

ALTER TABLE cmdb_datacenter_site ADD COLUMN IF NOT EXISTS locality_code VARCHAR(11);
CREATE INDEX IF NOT EXISTS cmdb_datacenter_site_locality_code ON cmdb_datacenter_site (locality_code);
ALTER TABLE cmdb_datacenter_site ADD COLUMN IF NOT EXISTS room VARCHAR(255);

-- Контроль: площадки с кодом, которого нет в справочнике
-- SELECT s.id, s.name, s.locality_code FROM cmdb_datacenter_site s
--  WHERE COALESCE(s.locality_code,'') <> ''
--    AND NOT EXISTS (SELECT 1 FROM cmdb_locality l WHERE l.oktmo = s.locality_code);
