-- =============================================================================
-- Справочник Площадок/ЦОД (cmdb_datacenter_site): поля "Адрес" и "Этаж".
-- Обычно колонки добавляет orm.RunSyncdb; скрипт - для ручного применения.
--
-- Формат данных после импорта из Итиллиум (itilliumimport):
--   name       = '<адрес> <код комнаты>'
--   annotation = 'ЦОД по адресу: <адрес> этаж:<этаж> комната: <код комнаты>'
-- Адрес = name без хвоста ' <код комнаты>' (комната берётся из annotation после
-- 'комната:'; если её нет или name на неё не оканчивается - адресом считается
-- весь name). Этаж = текст между 'этаж:' и 'комната' в annotation.
-- Перезаписываются только пустые address/floor (ручной ввод не затирается).
-- =============================================================================

-- --- PostgreSQL ---
ALTER TABLE cmdb_datacenter_site ADD COLUMN IF NOT EXISTS address VARCHAR(512);
ALTER TABLE cmdb_datacenter_site ADD COLUMN IF NOT EXISTS floor   VARCHAR(64);

WITH parsed AS (
    SELECT
        id,
        btrim(name) AS nm,
        NULLIF(btrim(substring(annotation FROM 'комната:\s*(.*)$')), '') AS room,
        NULLIF(btrim(substring(annotation FROM 'этаж:\s*(.*?)\s*комната')), '') AS flr
    FROM cmdb_datacenter_site
)
UPDATE cmdb_datacenter_site s
SET address = CASE
                  WHEN COALESCE(s.address, '') <> '' THEN s.address
                  WHEN p.room IS NOT NULL AND right(p.nm, length(p.room) + 1) = ' ' || p.room
                      THEN btrim(left(p.nm, length(p.nm) - length(p.room)))
                  ELSE p.nm
              END,
    floor   = CASE WHEN COALESCE(s.floor, '') <> '' THEN s.floor ELSE p.flr END
FROM parsed p
WHERE p.id = s.id;

-- Контроль результата:
-- SELECT id, name, address, floor, annotation FROM cmdb_datacenter_site ORDER BY name;

-- --- SQLite3 (ALTER повторно упадёт - нормально; разбор делается в Postgres) ---
-- ALTER TABLE cmdb_datacenter_site ADD COLUMN address VARCHAR(512);
-- ALTER TABLE cmdb_datacenter_site ADD COLUMN floor VARCHAR(64);
