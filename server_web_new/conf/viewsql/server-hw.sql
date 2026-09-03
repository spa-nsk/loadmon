-- Режим "Аппаратные серверы" (/admin/viewexport/server-hw).
-- ЗАГОТОВКА: список колонок и условия отбора нужно уточнить под реальную
-- задачу - при необходимости замените/дополните ниже. Порядок колонок в
-- SELECT определяет порядок колонок в таблице/выгрузке.
SELECT
    id,
    name,
    name_fqdn,
    serial_number,
    power_state,
    active,
    source_update_type,
    time__mfy
FROM server_spr_records
WHERE type = 0
ORDER BY name;
