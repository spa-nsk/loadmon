-- Режим "СХД - виртуальные тома" (/admin/viewexport/storage-vm).
-- ЗАГОТОВКА: назначение режима неоднозначно (связь ВМ <-> СХД/датастор?) -
-- ниже условный вариант через models.VMUsageDatastore (таблица
-- vm_usage_datastore). Замените на нужный запрос под реальную задачу.
SELECT *
FROM vm_usage_datastore
ORDER BY id;
