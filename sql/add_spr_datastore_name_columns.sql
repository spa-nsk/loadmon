-- =============================================================================
-- Колонки datacenter_name в справочниках VM и кластеров (models.VMSprRecord,
-- models.ClusterSprRecords) для cmdb.UpdateVMDatacenterNames /
-- cmdb.UpdateClusterDatacenterNames (см. cmdb/vmdatacentername.go,
-- cmdb/clusterdatacentername.go).
--
-- ПЕРЕИМЕНОВАНО (2026-09-30) из datastore_name - опечатка, колонка не имеет
-- отношения к датасторам vSphere, только к ЦОД. Если колонка datastore_name
-- уже была создана предыдущей версией скрипта/syncdb - её можно удалить
-- вручную (DROP COLUMN datastore_name), она больше не используется.
--
-- Обычно недостающие колонки добавляет orm.RunSyncdb при старте приложения;
-- скрипт — для ручного применения на БД (postgres/sqlite3), если нужно
-- обновить схему до деплоя или без перезапуска syncdb.
-- =============================================================================

-- --- PostgreSQL (рекомендуется IF NOT EXISTS) ---
ALTER TABLE vm_spr_record
    ADD COLUMN IF NOT EXISTS datacenter_name VARCHAR(512) NOT NULL DEFAULT '';

ALTER TABLE cluster_spr_records
    ADD COLUMN IF NOT EXISTS datacenter_name VARCHAR(512) NOT NULL DEFAULT '';

-- --- SQLite3 (выполнять отдельно; при повторном запуске ALTER упадёт — это нормально) ---
-- ALTER TABLE vm_spr_record ADD COLUMN datacenter_name varchar(512) NOT NULL DEFAULT '';
-- ALTER TABLE cluster_spr_records ADD COLUMN datacenter_name varchar(512) NOT NULL DEFAULT '';
