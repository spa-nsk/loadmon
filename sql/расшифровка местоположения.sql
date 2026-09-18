-- Таблицы: server_hardware, cmdb_rack_cabinet, cmdb_datacenter_site.
-- Столбцы стандартные (snake_case от имени поля, без переопределений column(...)
-- у этих трёх FK/Name полей).
--
-- Если нужна конкретная запись — добавь AND sh.id = :id в WHERE.
-- LEFT JOIN оставлен на случай, если RackCabinetId указывает на шкаф без
-- привязанного DatacenterSiteId (поле у RackCabinet nullable) — тогда
-- datacenter_name будет NULL, но строка по шкафу всё равно вернётся.

SELECT
    sh.id            AS server_hardware_id,
    ds.name          AS datacenter_name,
    rc.name          AS rack_cabinet_name
FROM server_hardware sh
LEFT JOIN cmdb_rack_cabinet rc
    ON rc.id = sh.rack_cabinet_id
LEFT JOIN cmdb_datacenter_site ds
    ON ds.id = rc.datacenter_site_id
WHERE sh.rack_cabinet_id IS NOT NULL
ORDER BY sh.id;
