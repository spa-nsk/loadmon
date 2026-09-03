-- Режим "Кластеры" (/admin/viewexport/cluster).
-- ЗАГОТОВКА: список колонок нужно уточнить под реальную структуру таблицы
-- cluster_spr_records (см. models.ClusterSprRecords) под задачу.
SELECT *
FROM cluster_spr_records
ORDER BY name;
