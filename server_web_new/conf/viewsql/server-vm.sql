-- Режим "Виртуальные серверы" (/admin/viewexport/server-vm).
-- ЗАГОТОВКА: список колонок и условия отбора нужно уточнить под реальную
-- задачу - при необходимости замените/дополните ниже.
SELECT
    id,
    name,
    name_fqdn,
    vcenter,
    cluster,
    power_state,
    active,
    source_update_type,
    time__mfy
FROM server_spr_records
WHERE type = 1
  AND stat <> 666 -- 666 - запись помечена к уничтожению, исключается из всех запросов
ORDER BY name;
