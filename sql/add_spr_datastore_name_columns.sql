-- =============================================================================
-- Колонки datastore_name в справочниках VM и кластеров (models.VMSprRecord,
-- models.ClusterSprRecords) для cmdb.UpdateVMDatastoreNames /
-- cmdb.UpdateClusterDatastoreNames (см. cmdb/vmdatastorename.go,
-- cmdb/clusterdatastorename.go).
--
-- Обычно недостающие колонки добавляет orm.RunSyncdb при старте приложения;
-- скрипт — для ручного применения на БД (postgres/sqlite3), если нужно
-- обновить схему до деплоя или без перезапуска syncdb.
-- =============================================================================

-- --- PostgreSQL (рекомендуется IF NOT EXISTS) ---
ALTER TABLE vm_spr_record
    ADD COLUMN IF NOT EXISTS datastore_name VARCHAR(512) NOT NULL DEFAULT '';

ALTER TABLE cluster_spr_records
    ADD COLUMN IF NOT EXISTS datastore_name VARCHAR(512) NOT NULL DEFAULT '';

-- --- SQLite3 (выполнять отдельно; при повторном запуске ALTER упадёт — это нормально) ---
-- ALTER TABLE vm_spr_record ADD COLUMN datastore_name varchar(512) NOT NULL DEFAULT '';
-- ALTER TABLE cluster_spr_records ADD COLUMN datastore_name varchar(512) NOT NULL DEFAULT '';
