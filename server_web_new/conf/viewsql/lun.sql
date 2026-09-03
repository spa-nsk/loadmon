-- Режим "LUN" (/admin/viewexport/lun).
-- ЗАГОТОВКА: в проекте нет отдельной модели "LUN" - ближайший кандидат по
-- смыслу models.StorageVolumes (таблица storage_volumes). Уточните запрос
-- под реальную задачу/структуру данных, если имелось в виду что-то другое.
SELECT *
FROM storage_volumes
ORDER BY id;
